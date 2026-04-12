package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
)

type Payment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

func main() {

	certFile := filepath.Join("certs", "server.crt")
	keyFile := filepath.Join("certs", "server.key")

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				log.Printf("Failed to load certificate: %v", err)
				return nil, err
			}
			return &cert, nil
		},
	}

	http.HandleFunc("GET /payments", handlePayments)
	http.HandleFunc("GET /health", handleHealth)

	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	log.Println("AtlasPay API starting on :8443 with TLS 1.3")
	log.Fatal(server.ListenAndServeTLS("", ""))
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
