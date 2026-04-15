package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
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

type ProfileService struct {
	users         map[string]*User
	emailRequests map[string]*EmailChangeRequest
	requestIDGen  int
}

func NewProfileService() *ProfileService {
	return &ProfileService{
		users:         make(map[string]*User),
		emailRequests: make(map[string]*EmailChangeRequest),
		requestIDGen:  0,
	}
}

func (ps *ProfileService) GetUser(userID string) *User {
	return ps.users[userID]
}

func (ps *ProfileService) GetEmailChangeRequest(requestID string) *EmailChangeRequest {
	return ps.emailRequests[requestID]
}

// State machine transition guards
func (ps *ProfileService) CanRequestEmailChange(userID string) error {
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
	req := ps.emailRequests[requestID]
	if req == nil {
		return fmt.Errorf("request not found")
	}

	if req.State != StatePendingVerification {
		return fmt.Errorf("invalid state transition: expected %s, got %s", StatePendingVerification, req.State)
	}

	return nil
}

// State machine operations
func (ps *ProfileService) RequestEmailChange(userID, newEmail string) (*EmailChangeRequest, error) {
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
	if err := ps.CanMarkPendingVerification(requestID); err != nil {
		return err
	}

	ps.emailRequests[requestID].State = StatePendingVerification
	return nil
}

func (ps *ProfileService) ConfirmEmailChange(requestID string) error {
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

// HTTP handlers
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

	req, err := ps.RequestEmailChange(userID, payload.NewEmail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

	if err := ps.ConfirmEmailChange(requestID); err != nil {
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

	http.HandleFunc("/request-email-change", service.requestEmailChangeHandler)
	http.HandleFunc("/mark-pending", service.markPendingHandler)
	http.HandleFunc("/confirm-email-change", service.confirmEmailChangeHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
