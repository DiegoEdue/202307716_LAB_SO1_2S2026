package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pb "go-api/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	switch getenv("MODE", "rest") {
	case "rest":
		runREST()
	case "client":
		runClient()
	default:
		log.Fatal("MODE invalido: usa rest o client")
	}
}

// Contenedor 1: API REST. Recibe de Rust y reenvia al contenedor gRPC client.
func runREST() {
	target := getenv("CLIENT_URL", "http://127.0.0.1:9090/send")
	hc := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 100}}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
			return
		}
		resp, err := hc.Post(target, "application/json", r.Body)
		if err != nil {
			log.Printf("error contactando grpc-client: %v", err)
			http.Error(w, "grpc-client no disponible", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
	log.Println("api-rest escuchando en :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// Contenedor 2: cliente gRPC. Solo escucha en localhost del pod.
func runClient() {
	addr := getenv("GRPC_SERVER_ADDR", "go-server-svc:50051")
	conn, err := grpc.NewClient("dns:///"+addr,
	grpc.WithTransportCredentials(insecure.NewCredentials()),
	grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`))
	if err != nil {
		log.Fatalf("no se pudo crear el cliente gRPC: %v", err)
	}
	defer conn.Close()
	client := pb.NewWarReportServiceClient(conn)

	mux := http.NewServeMux()
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		var rep Report
		if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
			http.Error(w, "json invalido", http.StatusBadRequest)
			return
		}
		c := strings.ToLower(rep.Country)
		if c == "gtm" {
			c = "gmt"
		}
		code, ok := pb.Countries_value[c]
		if !ok || code == int32(pb.Countries_countries_unknown) {
			http.Error(w, "pais invalido", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		resp, err := client.SendReport(ctx, &pb.WarReportRequest{
			Country:         pb.Countries(code),
			WarplanesInAir:  rep.WarplanesInAir,
			WarshipsInWater: rep.WarshipsInWater,
			Timestamp:       rep.Timestamp,
		})
		if err != nil {
			log.Printf("error gRPC: %v", err)
			http.Error(w, "error al contactar el servidor gRPC", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": resp.Status})
	})
	log.Println("grpc-client escuchando en 127.0.0.1:9090, servidor gRPC en", addr)
	log.Fatal(http.ListenAndServe("127.0.0.1:9090", mux))
}