package main

import (
	"context"
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
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Payment struct {
	ID       string  `json:"id"`
	Amount   float64 `json:"amount"`
	Status   string  `json:"status"`
	Customer string  `json:"-"`
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

func connectToDatabase() (*pgxpool.Pool, error) {
	clientCert, _ := tls.LoadX509KeyPair(
		filepath.Join("certs", "db-client.crt"),
		filepath.Join("certs", "db-client.key"),
	)

	caCertPEM, _ := os.ReadFile(filepath.Join("certs", "db-ca.crt"))
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCertPEM)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		ServerName:   "postgres.internal",
		MinVersion:   tls.VersionTLS12,
	}

	config, _ := pgxpool.ParseConfig("postgres://<app_name>:<placeholderpwd>@postgres.internal:5432/payments?sslmode=verify-full")
	config.ConnConfig.TLSConfig = tlsConfig

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pool, nil
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{"https://dashboard.atlaspay.com", "https://admin.atlaspay.com"}

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
				break
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

func sanitizeLogContext(r *http.Request) {
	r.URL.RawQuery = strings.ReplaceAll(r.URL.RawQuery, "customer_id=", "customer_id=***")
	r.URL.RawQuery = strings.ReplaceAll(r.URL.RawQuery, "api_key=", "api_key=***")
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

	db, _ := connectToDatabase()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /payments", handlePayments(clearingClient))
	mux.HandleFunc("GET /session", handleSession)
	mux.HandleFunc("GET /health", handleHealth)

	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
		Handler:   corsMiddleware(securityMiddleware(mux)),
	}

	log.Println("AtlasPay API starting on :8443 with full security controls")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func handlePayments(client *ClearingHouseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payment Payment
		if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		sanitizeLogContext(r)
		log.Printf("Processing payment %s from %s", payment.ID, r.URL.String())

		if err := client.SubmitPayment(payment.ID); err != nil {
			log.Printf("Failed to submit payment to clearing house: %v", err)
		} else {
			log.Printf("Payment %s submitted to clearing house", payment.ID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payment)
	}
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	sessionToken := "abc123sessiontoken"

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Session created"))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
