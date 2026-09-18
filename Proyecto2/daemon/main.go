// Daemon gestionador — Proyecto 2 SO1.
//
// Todo el sistema vive en este unico archivo (mas ebpf_kill.c, que por
// naturaleza necesita ser un archivo C aparte) para que se pueda leer de
// arriba a abajo sin saltar entre paquetes. Esta dividido en secciones
// con comentarios tipo banner; no es un solo bloque de codigo sin orden.
//
// Requiere privilegios de root (insmod, eBPF y kill directo lo piden).
//
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang ebpfkill ebpf_kill.c
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/redis/go-redis/v9"
)

// Configuración general
const (
	ProcInfoPath = "/proc/continfo_pr2_so1_202307716"
	LoopInterval = 30 * time.Second
	MinBajoConsumo = 3
	MinAltoConsumo = 2
	LabelKey       = "perfil"
	DeployScriptPath  = "../scripts/cronjob.sh"
	DockerComposeFile = "../grafana-valkey/docker-compose.yml"
	CronSchedule = "*/1 * * * *"
	CronMarker   = "# proyecto2-so1-cronjob"
	ValkeyAddr = "localhost:6379"
	KillConfirmTimeout = 15 * time.Second
)

// Tipos de datos que representan la informacion de procesos y contenedores
type ProcesoKernel struct {
	PID         int     `json:"PID"`
	Name        string  `json:"Name"`
	Cmdline     string  `json:"Cmdline,omitempty"`
	ContainerID string  `json:"ContainerID,omitempty"`
	VSZKB       uint64  `json:"VSZ_KB"`
	RSSKB       uint64  `json:"RSS_KB"`
	MemoryUsage float64 `json:"Memory_Usage"`
	CPUUsage    float64 `json:"CPU_Usage"`
}

type MemoriaKernel struct {
	TotalRAMKB uint64          `json:"TotalRAM_KB"`
	FreeRAMKB  uint64          `json:"FreeRAM_KB"`
	UsedRAMKB  uint64          `json:"UsedRAM_KB"`
	Processes  []ProcesoKernel `json:"Processes"`
}

type Container struct {
	ID      string
	Name    string
	Profile string // bajo | alto | intruso
}

type KillEvent struct {
	SenderPID uint32
	TargetPID uint32
	Signal    int32
}

// Lectura de /proc
func readMemoriaKernel(path string) (*MemoriaKernel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leyendo %s: %w", path, err)
	}
	var snap MemoriaKernel
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parseando %s: %w", path, err)
	}
	return &snap, nil
}

// byContainer agrupa los procesos por ContainerID (suma RSS/VSZ, promedia CPU/Mem).
func (s *MemoriaKernel) byContainer() map[string]ProcesoKernel {
	agg := make(map[string]ProcesoKernel)
	counts := make(map[string]int)
	for _, pr := range s.Processes {
		if pr.ContainerID == "" {
			continue
		}
		cur := agg[pr.ContainerID]
		cur.ContainerID = pr.ContainerID
		cur.VSZKB += pr.VSZKB
		cur.RSSKB += pr.RSSKB
		cur.MemoryUsage += pr.MemoryUsage
		cur.CPUUsage += pr.CPUUsage
		agg[pr.ContainerID] = cur
		counts[pr.ContainerID]++
	}
	for id, cur := range agg {
		n := float64(counts[id])
		cur.MemoryUsage /= n
		cur.CPUUsage /= n
		agg[id] = cur
	}
	return agg
}

// Docker
func listLabeledContainers() ([]Container, error) {
	cmd := exec.Command("docker", "ps",
		"--filter", "label="+LabelKey,
		"--format", "{{.ID}}|{{.Names}}|{{.Label \""+LabelKey+"\"}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}
	var result []Container
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		result = append(result, Container{ID: parts[0], Name: parts[1], Profile: parts[2]})
	}
	return result, nil
}

// hostPID obtiene el PID en el host del proceso principal del contenedor.
// Lo usamos para matar directo con syscall.Kill (no "docker kill"), asi
// la sonda eBPF atribuye el kill() al Daemon y no a dockerd.
func hostPID(containerID string) (int, error) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", containerID)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("docker inspect %s: %w", containerID, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(out.String()))
	if err != nil {
		return 0, fmt.Errorf("PID invalido para %s: %w", containerID, err)
	}
	return pid, nil
}

func removeContainer(containerID string) error {
	return exec.Command("docker", "rm", "-f", containerID).Run()
}

// Cronjob

func installCronjob() error {
	current, err := currentCrontab()
	if err != nil {
		return err
	}
	if strings.Contains(current, CronMarker) {
		return nil
	}
		absScript, err := filepath.Abs(DeployScriptPath)
	if err != nil {
		return fmt.Errorf("resolviendo ruta absoluta: %w", err)
	}
	line := fmt.Sprintf("%s bash %s %s\n", CronSchedule, absScript, CronMarker)
	return writeCrontab(current + line)
}

func removeCronjob() error {
	current, err := currentCrontab()
	if err != nil {
		return err
	}
	var kept []string
	for _, line := range strings.Split(current, "\n") {
		if strings.Contains(line, CronMarker) {
			continue
		}
		kept = append(kept, line)
	}
	return writeCrontab(strings.Join(kept, "\n"))
}

func currentCrontab() (string, error) {
	cmd := exec.Command("crontab", "-l")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "no crontab") {
			return "", nil
		}
		return "", fmt.Errorf("crontab -l: %w (%s)", err, stderr.String())
	}
	return out.String(), nil
}

func writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("crontab -: %w (%s)", err, stderr.String())
	}
	return nil
}

// Grafana/Valkey

func startGrafanaStack() error {
	cmd := exec.Command("docker", "compose", "-f", DockerComposeFile, "up", "-d")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("levantando docker-compose: %w\n%s", err, out)
	}
	return nil
}

// Esquema de llaves en Valkey, que se consultan desde Grafana.
//
//	ram:total / ram:free / ram:used  -> strings en KB
//	ram:history                      -> sorted set, score=timestamp ms
//	top:ram / top:cpu                -> sorted set, ZADD GT (picos historicos)
//	containers:deleted:count         -> contador
//	containers:deleted:history       -> sorted set, score=timestamp ms
//	ebpf:events:count                -> contador
type valkeyClient struct {
	rdb *redis.Client
}

func newValkeyClient() *valkeyClient {
	return &valkeyClient{rdb: redis.NewClient(&redis.Options{Addr: ValkeyAddr})}
}

func (v *valkeyClient) saveMemoriaKernel(ctx context.Context, snap *MemoriaKernel) error {
	now := float64(time.Now().UnixMilli())
	pipe := v.rdb.Pipeline()
	pipe.Set(ctx, "ram:total", snap.TotalRAMKB, 0)
	pipe.Set(ctx, "ram:free", snap.FreeRAMKB, 0)
	pipe.Set(ctx, "ram:used", snap.UsedRAMKB, 0)
	pipe.ZAdd(ctx, "ram:history", redis.Z{Score: now, Member: snap.UsedRAMKB})
	for id, pr := range snap.byContainer() {
		pipe.ZAdd(ctx, "top:ram", redis.Z{Score: float64(pr.RSSKB), Member: id})
		pipe.ZAdd(ctx, "top:cpu", redis.Z{Score: pr.CPUUsage, Member: id})
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (v *valkeyClient) recordDeleted(ctx context.Context, containerID string) error {
	pipe := v.rdb.Pipeline()
	pipe.Incr(ctx, "containers:deleted:count")
	pipe.ZAdd(ctx, "containers:deleted:history", redis.Z{
		Score: float64(time.Now().UnixMilli()), Member: containerID,
	})
	_, err := pipe.Exec(ctx)
	return err
}

func (v *valkeyClient) incrEBPFEvents(ctx context.Context) error {
	return v.rdb.Incr(ctx, "ebpf:events:count").Err()
}

// Sonda EBPf
type ebpfMonitor struct {
	objs   ebpfkillObjects
	tp     link.Link
	reader *ringbuf.Reader
}

func newEBPFMonitor() (*ebpfMonitor, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("removiendo memlock: %w", err)
	}
	objs := ebpfkillObjects{}
	if err := loadEbpfkillObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("cargando objetos eBPF: %w", err)
	}
	tp, err := link.Tracepoint("syscalls", "sys_enter_kill", objs.HandleKill, nil)
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("adjuntando tracepoint: %w", err)
	}
	rd, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		tp.Close()
		objs.Close()
		return nil, fmt.Errorf("abriendo ring buffer: %w", err)
	}
	return &ebpfMonitor{objs: objs, tp: tp, reader: rd}, nil
}

func (m *ebpfMonitor) events() <-chan KillEvent {
	ch := make(chan KillEvent, 64)
	go func() {
		defer close(ch)
		for {
			record, err := m.reader.Read()
			if err != nil {
				if errors.Is(err, ringbuf.ErrClosed) {
					return
				}
				log.Printf("[ebpf] error leyendo ring buffer: %v", err)
				continue
			}
			if len(record.RawSample) < 12 {
				continue
			}
			ch <- KillEvent{
				SenderPID: binary.LittleEndian.Uint32(record.RawSample[0:4]),
				TargetPID: binary.LittleEndian.Uint32(record.RawSample[4:8]),
				Signal:    int32(binary.LittleEndian.Uint32(record.RawSample[8:12])),
			}
		}
	}()
	return ch
}

func (m *ebpfMonitor) Close() error {
	if err := m.reader.Close(); err != nil {
		return err
	}
	if err := m.tp.Close(); err != nil {
		return err
	}
	return m.objs.Close()
}

// Eliminación de contenedores
func decide(active []Container, usage map[string]ProcesoKernel) []Container {
	var bajo, alto, intruso []Container
	for _, c := range active {
		switch c.Profile {
		case "bajo":
			bajo = append(bajo, c)
		case "alto":
			alto = append(alto, c)
		case "intruso":
			intruso = append(intruso, c)
		}
	}

	sortByConsumption(bajo, usage)
	sortByConsumption(alto, usage)

	var toKill []Container
	toKill = append(toKill, intruso...) // los intrusos se eliminan siempre
	if len(bajo) > MinBajoConsumo {
		toKill = append(toKill, bajo[MinBajoConsumo:]...) // conserva los de menor consumo
	}
	if len(alto) > MinAltoConsumo {
		toKill = append(toKill, alto[MinAltoConsumo:]...)
	}
	return toKill
}

func sortByConsumption(cs []Container, usage map[string]ProcesoKernel) {
	sort.Slice(cs, func(i, j int) bool {
		ui, uj := usage[cs[i].ID], usage[cs[j].ID]
		if ui.RSSKB != uj.RSSKB {
			return ui.RSSKB < uj.RSSKB
		}
		return ui.CPUUsage < uj.CPUUsage
	})
}

// Daemon

type daemon struct {
	vk  *valkeyClient
	mon *ebpfMonitor

	mu      sync.Mutex
	pending map[uint32]string // targetPID -> containerID
}

func (d *daemon) watchEBPFEvents(ctx context.Context) {
	for ev := range d.mon.events() {
		if err := d.vk.incrEBPFEvents(ctx); err != nil {
			log.Printf("[daemon] error incrementando contador eBPF: %v", err)
		}

		d.mu.Lock()
		containerID, ok := d.pending[ev.TargetPID]
		if ok {
			delete(d.pending, ev.TargetPID)
		}
		d.mu.Unlock()

		if !ok {
			continue
		}

		log.Printf("[daemon] eBPF confirmo terminacion de %s (pid=%d, sig=%d)", containerID, ev.TargetPID, ev.Signal)
		if err := removeContainer(containerID); err != nil {
			log.Printf("[daemon] error removiendo %s: %v", containerID, err)
		}
		if err := d.vk.recordDeleted(ctx, containerID); err != nil {
			log.Printf("[daemon] error registrando eliminacion: %v", err)
		}
	}
}

// tick es una iteracion del loop principal: lee /proc, guarda metricas, decide y dispara eliminaciones.
func (d *daemon) tick(ctx context.Context) error {
	snap, err := readMemoriaKernel(ProcInfoPath)
	if err != nil {
		return err
	}
	if err := d.vk.saveMemoriaKernel(ctx, snap); err != nil {
		log.Printf("[daemon] error guardando MemoriaKernel: %v", err)
	}

	active, err := listLabeledContainers()
	if err != nil {
		return err
	}

	for _, c := range decide(active, snap.byContainer()) {
		d.startKill(c)
	}
	return nil
}

func (d *daemon) startKill(c Container) {
	pid, err := hostPID(c.ID)
	if err != nil {
		log.Printf("[daemon] no se pudo obtener PID de %s: %v", c.ID, err)
		return
	}

	d.mu.Lock()
	d.pending[uint32(pid)] = c.ID
	d.mu.Unlock()

	log.Printf("[daemon] eliminando %s (perfil=%s, host_pid=%d)", c.ID, c.Profile, pid)
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Printf("[daemon] error en kill(%d): %v", pid, err)
		d.mu.Lock()
		delete(d.pending, uint32(pid))
		d.mu.Unlock()
		return
	}

	// Solo para diagnostico: avisa si no llega confirmacion eBPF a tiempo.
	go func(pid uint32, id string) {
		time.Sleep(KillConfirmTimeout)
		d.mu.Lock()
		_, stillPending := d.pending[pid]
		d.mu.Unlock()
		if stillPending {
			log.Printf("[daemon] advertencia: sin confirmacion eBPF para %s", id)
		}
	}(uint32(pid), c.ID)
}

// Main, 5 procesos principales del proyecto.

func main() {
	log.Println("[daemon] iniciando...")

	// Inicio del servicio.
	if err := startGrafanaStack(); err != nil {
		log.Fatalf("[daemon] Grafana/Valkey: %v", err)
	}
	log.Println("[daemon] Grafana/Valkey arriba.")

	// Cronjob.
	if err := installCronjob(); err != nil {
		log.Fatalf("[daemon] cronjob: %v", err)
	}
	log.Println("[daemon] cronjob instalado.")

	// El modulo de kernel se carga con `make load` desde modulo/
	if out, err := exec.Command("make", "-C", "../modulo", "load").CombinedOutput(); err != nil {
	log.Printf("[daemon] no se pudo cargar el modulo de kernel: %v\n%s", err, out)
	} else {
		log.Println("[daemon] modulo de kernel cargado.")
	}

	mon, err := newEBPFMonitor()
	if err != nil {
		log.Fatalf("[daemon] eBPF: %v", err)
	}
	defer mon.Close()

	vk := newValkeyClient()
	defer vk.rdb.Close()

	d := &daemon{vk: vk, mon: mon, pending: make(map[uint32]string)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.watchEBPFEvents(ctx)

	// Loop principal.
	ticker := time.NewTicker(LoopInterval)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Printf("[daemon] listo, loop cada %s, leyendo %s", LoopInterval, ProcInfoPath)
	for {
		select {
		case <-ticker.C:
			if err := d.tick(ctx); err != nil {
				log.Printf("[daemon] error en tick: %v", err)
			}
		case <-stop:
			log.Println("[daemon] señal de apagado, finalizando...")
			// Finalizacion del servicio.
			if err := removeCronjob(); err != nil {
				log.Printf("[daemon] error removiendo cronjob: %v", err)
			} else {
				log.Println("[daemon] cronjob removido.")
			}
			return
		}
	}
}