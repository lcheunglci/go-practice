package main

import (
	"sync"
	"testing"
	"time"
)

func TestConcurrencyAndIdempotency(t *testing.T) {
	service := NewProfileService()
	service.users["user1"] = &User{ID: "user1", Email: "old@example.com"}

	t.Run("concurrent requests with same idempotency key", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan *EmailChangeRequest, 5)
		errors := make(chan error, 5)

		// Launch multiple concurrent requests
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := service.RequestEmailChange("user1", "concurrent@example.com")
				if err != nil {
					errors <- err
				} else {
					results <- req
				}
			}()
		}

		wg.Wait()
		close(results)
		close(errors)

		// Should only get one successful result due to state machine guards
		successCount := 0
		var successReq *EmailChangeRequest
		for req := range results {
			successCount++
			successReq = req
		}

		errorCount := 0
		for range errors {
			errorCount++
		}

		if successCount != 1 {
			t.Errorf("Expected exactly 1 successful request, got %d", successCount)
		}

		if errorCount != 4 {
			t.Errorf("Expected exactly 4 failed requests, got %d", errorCount)
		}

		// Clean up
		if successReq != nil {
			service.MarkPendingVerification(successReq.ID)
			service.ConfirmEmailChange(successReq.ID)
		}
	})

	t.Run("idempotency key prevents duplicate processing", func(t *testing.T) {
		// Reset service state
		service.emailRequests = make(map[string]*EmailChangeRequest)
		service.requestIDGen = 0
		service.users["user1"].Email = "old@example.com"

		idempotencyKey := service.generateIdempotencyKey("user1", "idempotent@example.com")

		// First request
		req1, err1 := service.RequestEmailChange("user1", "idempotent@example.com")
		if err1 != nil {
			t.Fatalf("First request failed: %v", err1)
		}

		// Store idempotency record
		service.storeIdempotency(idempotencyKey, req1.ID, req1)

		// Complete the workflow
		service.MarkPendingVerification(req1.ID)
		service.ConfirmEmailChange(req1.ID)

		// Check idempotency
		record, exists := service.checkIdempotency(idempotencyKey)
		if !exists {
			t.Error("Idempotency record should exist")
		}

		if record.RequestID != req1.ID {
			t.Errorf("Expected request ID %s, got %s", req1.ID, record.RequestID)
		}
	})

	t.Run("singleflight prevents concurrent confirmations", func(t *testing.T) {
		// Reset service state
		service.emailRequests = make(map[string]*EmailChangeRequest)
		service.requestIDGen = 0
		service.users["user1"].Email = "old@example.com"

		// Create a request and mark it pending
		req, err := service.RequestEmailChange("user1", "singleflight@example.com")
		if err != nil {
			t.Fatalf("Request creation failed: %v", err)
		}

		err = service.MarkPendingVerification(req.ID)
		if err != nil {
			t.Fatalf("Mark pending failed: %v", err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, 5)

		// Launch multiple concurrent confirmations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := service.ConfirmEmailChange(req.ID)
				errors <- err
			}()
		}

		wg.Wait()
		close(errors)

		// Should only get one success (nil error), rest should fail
		successCount := 0
		errorCount := 0
		for err := range errors {
			if err == nil {
				successCount++
			} else {
				errorCount++
			}
		}

		if successCount != 1 {
			t.Errorf("Expected exactly 1 successful confirmation, got %d", successCount)
		}

		if errorCount != 4 {
			t.Errorf("Expected exactly 4 failed confirmations, got %d", errorCount)
		}
	})

	t.Run("expired idempotency records are cleaned up", func(t *testing.T) {
		// Create an expired record
		key := "expired_key"
		service.idempotencyStore[key] = &IdempotencyRecord{
			RequestID: "req_expired",
			Response:  "expired_response",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		}

		// Should not find expired record
		_, exists := service.checkIdempotency(key)
		if exists {
			t.Error("Should not find expired idempotency record")
		}

		// Cleanup should remove it
		service.cleanupExpiredIdempotencyRecords()

		// Verify it's removed from the store
		service.idempotencyMu.RLock()
		_, stillExists := service.idempotencyStore[key]
		service.idempotencyMu.RUnlock()

		if stillExists {
			t.Error("Expired record should be cleaned up")
		}
	})
}
