package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type Report struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

type config struct {
	assigned string
	ttl      int
	maxLen   int
	keys     []string
}

// Todo se actualiza en una sola operacion atomica (seguro con varias replicas)
// KEYS: 1 max:planes 2 min:planes 3 max:ships 4 min:ships 5 top:planes 6 top:ships
//
//	7 freq:planes 8 freq:ships 9 reports:count 10 ts:<pais asignado>
//
// ARGV: 1 aviones 2 barcos 3 pais 4 ttl 5 "1" si es el pais asignado 6 maxlen del stream
var script = redis.NewScript(`
if #KEYS ~= 12 then
  return redis.error_reply('se esperaban 12 llaves, llegaron ' .. #KEYS)
end
local planes = tonumber(ARGV[1])
local ships = tonumber(ARGV[2])
local country = ARGV[3]

local function maxset(key, v)
  local cur = redis.call('GET', key)
  if (not cur) or v > tonumber(cur) then redis.call('SET', key, v) end
end
local function minset(key, v)
  local cur = redis.call('GET', key)
  if (not cur) or v < tonumber(cur) then redis.call('SET', key, v) end
end

maxset(KEYS[1], planes)
minset(KEYS[2], planes)
maxset(KEYS[3], ships)
minset(KEYS[4], ships)
redis.call('HINCRBY', KEYS[5], country, planes)
redis.call('HINCRBY', KEYS[6], country, ships)
redis.call('ZINCRBY', KEYS[7], 1, tostring(planes))
redis.call('ZINCRBY', KEYS[8], 1, tostring(ships))
redis.call('SET', KEYS[11], redis.call('ZRANGE', KEYS[7], 0, 0, 'REV')[1])
redis.call('SET', KEYS[12], redis.call('ZRANGE', KEYS[8], 0, 0, 'REV')[1])
redis.call('HINCRBY', KEYS[9], country, 1)
if ARGV[5] == '1' then
  redis.call('XADD', KEYS[10], 'MAXLEN', '~', ARGV[6], '*', 'planes', planes, 'ships', ships)
end
for i = 1, #KEYS do
  redis.call('EXPIRE', KEYS[i], tonumber(ARGV[4]))
end
return 1
`)

var processed atomic.Int64

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(k)); err == nil && n > 0 {
		return n
	}
	return def
}

func waitValkey(rdb *redis.Client) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := rdb.Ping(ctx).Err()
		cancel()
		if err == nil {
			return
		}
		log.Printf("Valkey no disponible, reintentando: %v", err)
		time.Sleep(3 * time.Second)
	}
}

func dialRabbit(url string) *amqp.Connection {
	for {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn
		}
		log.Printf("RabbitMQ no disponible, reintentando: %v", err)
		time.Sleep(3 * time.Second)
	}
}

func process(rdb *redis.Client, cfg config, d amqp.Delivery) {
	var r Report
	if err := json.Unmarshal(d.Body, &r); err != nil || r.Country == "" {
		log.Printf("mensaje invalido, descartado: %v", err)
		d.Reject(false) // no se reencola un mensaje que nunca sera valido
		return
	}
	isAssigned := "0"
	if strings.EqualFold(r.Country, cfg.assigned) {
		isAssigned = "1"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := script.Run(ctx, rdb, cfg.keys,
		r.WarplanesInAir, r.WarshipsInWater, strings.ToUpper(r.Country),
		cfg.ttl, isAssigned, cfg.maxLen).Err()
	if err != nil {
		log.Printf("error escribiendo en Valkey, se reencola: %v", err)
		time.Sleep(time.Second)
		d.Nack(false, true)
		return
	}
	d.Ack(false)
	processed.Add(1)
}

func main() {
	rabbitURL := getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	queue := getenv("QUEUE", "reports")
	valkeyAddr := getenv("VALKEY_ADDR", "valkey-svc:6379")
	workers := getenvInt("WORKERS", 8)
	prefetch := getenvInt("PREFETCH", 50)

	assigned := strings.ToLower(getenv("ASSIGNED_COUNTRY", "esp"))
	cfg := config{
		assigned: assigned,
		ttl:      getenvInt("TTL_SECONDS", 86400),
		maxLen:   getenvInt("STREAM_MAXLEN", 20000),
		keys: []string{
			"max:planes", "min:planes", "max:ships", "min:ships",
			"top:planes", "top:ships", "freq:planes", "freq:ships",
			"reports:count", "ts:" + assigned,
			"mode:planes", "mode:ships",
		},
	}

	rdb := redis.NewClient(&redis.Options{Addr: valkeyAddr, PoolSize: workers * 2})
	waitValkey(rdb)
	conn := dialRabbit(rabbitURL)
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("no se pudo abrir canal: %v", err)
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		log.Fatalf("no se pudo declarar la cola: %v", err)
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		log.Fatalf("no se pudo configurar prefetch: %v", err)
	}
	msgs, err := ch.Consume(queue, "go-consumer", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("no se pudo consumir: %v", err)
	}
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))

	// Pool de workers: cada goroutine procesa mensajes en paralelo
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range msgs {
				process(rdb, cfg, d)
			}
		}()
	}

	go func() {
		var last int64
		for range time.Tick(10 * time.Second) {
			if n := processed.Load(); n != last {
				log.Printf("procesados: %d (+%d en 10s)", n, n-last)
				last = n
			}
		}
	}()

	log.Printf("consumer listo: cola=%s valkey=%s workers=%d pais asignado=%s", queue, valkeyAddr, workers, assigned)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case e := <-closed:
		log.Fatalf("conexion con RabbitMQ perdida: %v", e) // Kubernetes lo reinicia
	case <-sig:
		log.Println("cerrando...")
		ch.Cancel("go-consumer", false)
		wg.Wait()
		conn.Close()
	}
}
