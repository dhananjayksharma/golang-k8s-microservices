package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCircuitBreakerOpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Hour)
	downstreamErr := errors.New("downstream failed")
	downstreamCalls := 0

	fail := func() error {
		downstreamCalls++
		return downstreamErr
	}

	if err := cb.Execute(fail); !errors.Is(err, downstreamErr) {
		t.Fatalf("first call error = %v, want %v", err, downstreamErr)
	}
	if err := cb.Execute(fail); !errors.Is(err, downstreamErr) {
		t.Fatalf("second call error = %v, want %v", err, downstreamErr)
	}
	if got := cb.Status(); got != Open {
		t.Fatalf("status = %s, want %s", got, Open)
	}

	blockedCallReachedDownstream := false
	err := cb.Execute(func() error {
		blockedCallReachedDownstream = true
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("third call error = %v, want %v", err, ErrCircuitOpen)
	}
	if blockedCallReachedDownstream {
		t.Fatal("third call reached downstream, want circuit breaker to fail fast")
	}
	if downstreamCalls != 2 {
		t.Fatalf("downstream calls = %d, want 2", downstreamCalls)
	}
}

func TestCircuitBreakerHalfOpenSuccessClosesCircuit(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Minute)

	if err := cb.Execute(func() error {
		return errors.New("temporary failure")
	}); err == nil {
		t.Fatal("first call error = nil, want failure")
	}
	if got := cb.Status(); got != Open {
		t.Fatalf("status = %s, want %s", got, Open)
	}

	cb.mu.Lock()
	cb.lastFailureTime = time.Now().Add(-time.Minute)
	cb.mu.Unlock()

	if err := cb.Execute(func() error {
		return nil
	}); err != nil {
		t.Fatalf("half-open call error = %v, want nil", err)
	}
	if got := cb.Status(); got != Closed {
		t.Fatalf("status = %s, want %s", got, Closed)
	}
}

func TestCircuitBreakerHalfOpenFailureReopensCircuit(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Minute)

	if err := cb.Execute(func() error {
		return errors.New("temporary failure")
	}); err == nil {
		t.Fatal("first call error = nil, want failure")
	}

	cb.mu.Lock()
	cb.lastFailureTime = time.Now().Add(-time.Minute)
	cb.mu.Unlock()

	if err := cb.Execute(func() error {
		return errors.New("still failing")
	}); err == nil {
		t.Fatal("half-open call error = nil, want failure")
	}
	if got := cb.Status(); got != Open {
		t.Fatalf("status = %s, want %s", got, Open)
	}
}

func TestCallExternalAPI(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		err := callExternalAPI(server.Client(), server.URL)
		if err == nil {
			t.Fatal("error = nil, want server error")
		}
	})

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		if err := callExternalAPI(server.Client(), server.URL); err != nil {
			t.Fatalf("error = %v, want nil", err)
		}
	})
}
