package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"VM"`
	Carnet    string `json:"carnet"`
}

type CallResponse struct {
	APIName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

const Carnet = "202307716"
const Ip1 = "192.168.122.11"
const Ip2 = "192.168.122.12"

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func checkHealth(url string) bool {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var health HealthResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	if err := json.Unmarshal(body, &health); err != nil {
		return false
	}

	return health.Status == "UP"
}

func main() {
	// Endpoint: Health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{
			Status:    "UP",
			Message:   "API3 is Ready",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			VM:        "VM2",
			Carnet:    Carnet,
		}
		writeJSON(w, resp)
	})

	// Endpoint: Llamar a API1
	http.HandleFunc("/api3/"+Carnet+"/call-api1", func(w http.ResponseWriter, r *http.Request) {
		isWorking := checkHealth("http://" + Ip1 + ":8081/health")

		msg := "The API1 located on the VM1 is working"
		if !isWorking {
			msg = "ERROR: The API1 located on the VM1 is not working"
		}

		writeJSON(w, CallResponse{
			APIName:    "API1",
			Message:    msg,
			Connection: isWorking,
			Carnet:     Carnet,
		})
	})

	// Endpoint: Llamar a API2
	http.HandleFunc("/api3/"+Carnet+"/call-api2", func(w http.ResponseWriter, r *http.Request) {
		isWorking := checkHealth("http://" + Ip1 + ":8082/health")

		msg := "The API2 located on the VM1 is working"
		if !isWorking {
			msg = "ERROR: The API2 located on the VM1 is not working"
		}

		writeJSON(w, CallResponse{
			APIName:    "API2",
			Message:    msg,
			Connection: isWorking,
			Carnet:     Carnet,
		})
	})

	fmt.Println("API3 corriendo en puerto 8083 (VM2)")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
