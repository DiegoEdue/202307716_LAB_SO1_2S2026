package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	pb "go-grpc-server/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type Report struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	switch getenv("MODE", "server") {
	case "server":
		runServer()
	case "writer":
		runWriter()
	default:
		log.Fatal("MODE invalido: usa server o writer")
	}
}

// ---------- Contenedor 1: servidor gRPC ----------

type server struct {
	pb.UnimplementedWarReportServiceServer
	hc        *http.Client
	writerURL string
}

func (s *server) SendReport(ctx context.Context, req *pb.WarReportRequest) (*pb.WarReportResponse, error) {
	rep := Report{
		Country:         req.GetCountry().String(),
		WarplanesInAir:  req.GetWarplanesInAir(),
		WarshipsInWater: req.GetWarshipsInWater(),
		Timestamp:       req.GetTimestamp(),
	}
	body, err := json.Marshal(rep)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "no se pudo serializar: %v", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.writerURL, bytes.NewReader(body))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.hc.Do(httpReq)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "writer no disponible: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Internal, "writer respondio %d", resp.StatusCode)
	}
	return &pb.WarReportResponse{Status: "Reporte de " + rep.Country + " procesado"}, nil
}

func runServer() {
	writerURL := getenv("WRITER_URL", "http://127.0.0.1:9091/publish")
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("no se pudo escuchar en :50051: %v", err)
	}
	// MaxConnectionAge obliga a los clientes a reconectar y volver a resolver DNS,
	// asi detectan las replicas nuevas cuando escalas de 1 a 2
	srv := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 10 * time.Second,
	}))
	hc := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 100}}
	pb.RegisterWarReportServiceServer(srv, &server{hc: hc, writerURL: writerURL})
	log.Println("grpc-server escuchando en :50051, writer en", writerURL)
	log.Fatal(srv.Serve(lis))
}

// ---------- Contenedor 2: writer hacia RabbitMQ ----------

type job struct {
	body []byte
	res  chan error
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

func runWriter() {
	url := getenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	queue := getenv("QUEUE", "reports")
	workers, err := strconv.Atoi(getenv("WORKERS", "4"))
	if err != nil || workers < 1 {
		workers = 4
	}

	conn := dialRabbit(url)
	// Si se pierde la conexion, el proceso termina y Kubernetes/Docker lo reinicia
	go func() {
		e := <-conn.NotifyClose(make(chan *amqp.Error, 1))
		log.Fatalf("conexion con RabbitMQ perdida: %v", e)
	}()

	// Pool de workers: cada goroutine tiene su propio canal AMQP (no son seguros entre hilos)
	jobs := make(chan job, 1024)
	for i := 0; i < workers; i++ {
		ch, err := conn.Channel()
		if err != nil {
			log.Fatalf("no se pudo abrir canal: %v", err)
		}
		if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			log.Fatalf("no se pudo declarar la cola: %v", err)
		}
		if err := ch.Confirm(false); err != nil {
			log.Fatalf("no se pudo activar confirmaciones: %v", err)
		}
		go func(ch *amqp.Channel) {
			confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
			for j := range jobs {
				err := ch.PublishWithContext(context.Background(), "", queue, false, false, amqp.Publishing{
					ContentType:  "application/json",
					DeliveryMode: amqp.Persistent,
					Body:         j.body,
				})
				if err == nil {
					if c, ok := <-confirms; !ok || !c.Ack {
						err = errors.New("RabbitMQ no confirmo el mensaje")
					}
				}
				j.res <- err
			}
		}(ch)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cuerpo invalido", http.StatusBadRequest)
			return
		}
		j := job{body: body, res: make(chan error, 1)}
		select {
		case jobs <- j:
		case <-time.After(5 * time.Second):
			http.Error(w, "cola interna llena", http.StatusServiceUnavailable)
			return
		}
		select {
		case err := <-j.res:
			if err != nil {
				log.Printf("error publicando: %v", err)
				http.Error(w, "error publicando en RabbitMQ", http.StatusBadGateway)
				return
			}
			w.Write([]byte("ok"))
		case <-time.After(5 * time.Second):
			http.Error(w, "timeout publicando", http.StatusGatewayTimeout)
		}
	})
	log.Printf("writer escuchando en 127.0.0.1:9091 (%d workers, cola %s)", workers, queue)
	log.Fatal(http.ListenAndServe("127.0.0.1:9091", mux))
}