package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Payment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

func main() {
	http.HandleFunc("GET /payments", handlePayments)
	http.HandleFunc("GET /health", handleHealth)

	log.Println("AtlasPay API starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handlePayments(w http.ResponseWriter, r *http.Request) {
	payments := []Payment{
		{ID: "PAY-001", Amount: 50000.00, Status: "completed"},
		{ID: "PAY-002", Amount: 125000.00, Status: "pending"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
