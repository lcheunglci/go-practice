package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Payment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type ClearingHouseClient struct {
	httpClient        *http.Client
	pinnedFingerprint string
	pinMismatches     int
}

func NewClearingHouseClient(pinnedFingerprint string) *ClearingHouseClient {
	caCertPEM, _ := os.ReadFile(filepath.Join("certs", "ca.crt"))
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCertPEM)

	cert, _ := tls.LoadX509KeyPair(
		filepath.Join("certs", "client.crt"),
		filepath.Join("certs", "client.key"),
	)

	client := &ClearingHouseClient{
		pinnedFingerprint: pinnedFingerprint,
	}

	tlsConfig := &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return client.verifyPinning(rawCerts)
		},
	}

	client.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 10 * time.Second,
	}

	return client
}

func (c *ClearingHouseClient) verifyPinning(rawCerts [][]byte) error {
	if len(rawCerts) == 0 {
		return nil
	}

	cert, _ := x509.ParseCertificate(rawCerts[0])
	fingerprint := sha256.Sum256(cert.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	if fingerprintHex != c.pinnedFingerprint {
		c.pinMismatches++
		log.Printf("Certificate pin mismatch! Expected: %s, Got: %s (mismatches: %d)",
			c.pinnedFingerprint, fingerprintHex, c.pinMismatches)
	}

	return nil
}

func (c *ClearingHouseClient) SubmitPayment(paymentID string) error {
	resp, err := c.httpClient.Get("https://localhost:9443/submit")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func startClearingHouseMock() {

	certFile := filepath.Join("certs", "server.crt")
	keyFile := filepath.Join("certs", "server.key")
	caCertFile := filepath.Join("certs", "ca.crt")

	caCert, _ := os.ReadFile(caCertFile)
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caPool,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Clearing house received submission from %s", r.TLS.PeerCertificates[0].Subject.CommonName)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "accepted",
			"message": "Payment queued for clearing",
		})
	})

	server := &http.Server{
		Addr:      ":9443",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	log.Println("Clearing house mock starting on :9443")
	server.ListenAndServeTLS(certFile, keyFile)
}

func main() {
	var wg sync.WaitGroup

	wg.Go(startClearingHouseMock)

	time.Sleep(500 * time.Millisecond)

	certFile := filepath.Join("certs", "server.crt")
	keyFile := filepath.Join("certs", "server.key")
	caCertFile := filepath.Join("certs", "ca.crt")

	caCert, _ := os.ReadFile(caCertFile)
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caPool,
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

	certBytes, _ := os.ReadFile(certFile)
	block, _ := pem.Decode(certBytes)
	serverCert, _ := x509.ParseCertificate(block.Bytes)
	fingerprint := sha256.Sum256(serverCert.Raw)
	pinnedFingerprint := hex.EncodeToString(fingerprint[:])

	clearingClient := NewClearingHouseClient(pinnedFingerprint)

	http.HandleFunc("GET /payments", handlePayments(clearingClient))
	http.HandleFunc("GET /health", handleHealth)

	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	log.Println("AtlasPay API starting on :8443 with certificate pinning")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func handlePayments(client *ClearingHouseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payments := []Payment{
			{ID: "PAY-001", Amount: 50000.00, Status: "completed"},
			{ID: "PAY-002", Amount: 125000.00, Status: "pending"},
		}

		if err := client.SubmitPayment("PAY-002"); err != nil {
			log.Printf("Failed to submit payment to clearing house: %v", err)
		} else {
			log.Println("Payment PAY-002 submitted to clearing house")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payments)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
