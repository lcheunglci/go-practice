package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type EmailChangeState string

const (
	StateIdle                EmailChangeState = "idle"
	StateRequestedChange     EmailChangeState = "requested"
	StatePendingVerification EmailChangeState = "pending_verification"
	StateConfirmed           EmailChangeState = "confirmed"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type EmailChangeRequest struct {
	ID          string           `json:"id"`
	UserID      string           `json:"user_id"`
	NewEmail    string           `json:"new_email"`
	State       EmailChangeState `json:"state"`
	RequestedAt time.Time        `json:"requested_at"`
}

type ChangeEmailRequestPayload struct {
	NewEmail string `json:"new_email"`
}

type IdempotencyRecord struct {
	RequestID string
	Response  any
	ExpiresAt time.Time
}

type ProfileService struct {
	users             map[string]*User
	emailRequests     map[string]*EmailChangeRequest
	requestIDGen      int
	mu                sync.RWMutex
	idempotencyStore  map[string]*IdempotencyRecord
	idempotencyMu     sync.RWMutex
	confirmationGroup singleflight.Group
}

func NewProfileService() *ProfileService {
	return &ProfileService{
		users:            make(map[string]*User),
		emailRequests:    make(map[string]*EmailChangeRequest),
		requestIDGen:     0,
		idempotencyStore: make(map[string]*IdempotencyRecord),
	}
}

func (ps *ProfileService) GetUser(userID string) *User {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.users[userID]
}

func (ps *ProfileService) GetEmailChangeRequest(requestID string) *EmailChangeRequest {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.emailRequests[requestID]
}

// Idempotency key generation and checking
func (ps *ProfileService) generateIdempotencyKey(userID, newEmail string) string {
	data := fmt.Sprintf("%s:%s", userID, newEmail)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (ps *ProfileService) checkIdempotency(key string) (*IdempotencyRecord, bool) {
	ps.idempotencyMu.RLock()
	defer ps.idempotencyMu.RUnlock()

	record, exists := ps.idempotencyStore[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(record.ExpiresAt) {
		// Expired, will be cleaned up later
		return nil, false
	}

	return record, true
}

func (ps *ProfileService) storeIdempotency(key, requestID string, response any) {
	ps.idempotencyMu.Lock()
	defer ps.idempotencyMu.Unlock()

	ps.idempotencyStore[key] = &IdempotencyRecord{
		RequestID: requestID,
		Response:  response,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour TTL
	}
}

func (ps *ProfileService) cleanupExpiredIdempotencyRecords() {
	ps.idempotencyMu.Lock()
	defer ps.idempotencyMu.Unlock()

	now := time.Now()
	for key, record := range ps.idempotencyStore {
		if now.After(record.ExpiresAt) {
			delete(ps.idempotencyStore, key)
		}
	}
}

// State machine transition guards (now thread-safe)
func (ps *ProfileService) CanRequestEmailChange(userID string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Check if user exists
	if ps.users[userID] == nil {
		return fmt.Errorf("user not found")
	}

	// Check if there's already an active request
	for _, req := range ps.emailRequests {
		if req.UserID == userID && req.State != StateConfirmed {
			return fmt.Errorf("email change already in progress")
		}
	}

	return nil
}

func (ps *ProfileService) CanMarkPendingVerification(requestID string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	req := ps.emailRequests[requestID]
	if req == nil {
		return fmt.Errorf("request not found")
	}

	if req.State != StateRequestedChange {
		return fmt.Errorf("invalid state transition: expected %s, got %s", StateRequestedChange, req.State)
	}

	return nil
}

func (ps *ProfileService) CanConfirmEmailChange(requestID string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	req := ps.emailRequests[requestID]
	if req == nil {
		return fmt.Errorf("request not found")
	}

	if req.State != StatePendingVerification {
		return fmt.Errorf("invalid state transition: expected %s, got %s", StatePendingVerification, req.State)
	}

	return nil
}

// State machine operations (now thread-safe)
func (ps *ProfileService) RequestEmailChange(userID, newEmail string) (*EmailChangeRequest, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if err := ps.CanRequestEmailChange(userID); err != nil {
		return nil, err
	}

	ps.requestIDGen++
	requestID := fmt.Sprintf("req_%d", ps.requestIDGen)

	req := &EmailChangeRequest{
		ID:          requestID,
		UserID:      userID,
		NewEmail:    newEmail,
		State:       StateRequestedChange,
		RequestedAt: time.Now(),
	}

	ps.emailRequests[requestID] = req
	return req, nil
}

func (ps *ProfileService) MarkPendingVerification(requestID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if err := ps.CanMarkPendingVerification(requestID); err != nil {
		return err
	}

	ps.emailRequests[requestID].State = StatePendingVerification
	return nil
}

func (ps *ProfileService) ConfirmEmailChange(requestID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if err := ps.CanConfirmEmailChange(requestID); err != nil {
		return err
	}

	req := ps.emailRequests[requestID]
	user := ps.users[req.UserID]

	// Apply the change
	user.Email = req.NewEmail
	req.State = StateConfirmed

	return nil
}

// HTTP handlers with idempotency support
func (ps *ProfileService) requestEmailChangeHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	var payload ChangeEmailRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check for idempotency key in header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		// Generate one based on request content
		idempotencyKey = ps.generateIdempotencyKey(userID, payload.NewEmail)
	}

	// Check if we've seen this request before
	if record, exists := ps.checkIdempotency(idempotencyKey); exists {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(record.Response)
		return
	}

	// Process the request
	req, err := ps.RequestEmailChange(userID, payload.NewEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Store for idempotency
	ps.storeIdempotency(idempotencyKey, req.ID, req)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (ps *ProfileService) markPendingHandler(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Missing request ID", http.StatusBadRequest)
		return
	}

	if err := ps.MarkPendingVerification(requestID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "marked pending verification"})
}

func (ps *ProfileService) confirmEmailChangeHandler(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "Missing request ID", http.StatusBadRequest)
		return
	}

	// Use singleflight to prevent concurrent confirmations of the same request
	result, err, _ := ps.confirmationGroup.Do(requestID, func() (any, error) {
		return nil, ps.ConfirmEmailChange(requestID)
	})

	_ = result // result is nil in this case, we only care about the error

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "email change confirmed"})
}

func main() {
	service := NewProfileService()

	// Create a test user
	service.users["user1"] = &User{
		ID:    "user1",
		Email: "old@example.com",
	}

	// Start cleanup goroutine for expired idempotency records
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			service.cleanupExpiredIdempotencyRecords()
		}
	}()

	http.HandleFunc("/request-email-change", service.requestEmailChangeHandler)
	http.HandleFunc("/mark-pending", service.markPendingHandler)
	http.HandleFunc("/confirm-email-change", service.confirmEmailChangeHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
