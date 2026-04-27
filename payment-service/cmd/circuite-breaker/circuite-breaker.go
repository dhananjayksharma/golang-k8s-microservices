package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type State string

const (
	Closed   State = "CLOSED"
	Open     State = "OPEN"
	HalfOpen State = "HALF_OPEN"
)

var ErrCircuitOpen = errors.New("circuit open")

type CircuitBreaker struct {
	mu              sync.Mutex
	state           State
	failures        int
	maxFailures     int
	resetTimeout    time.Duration
	lastFailureTime time.Time
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        Closed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	if cb.state == Open {
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = HalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}

	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = Open
		}

		return err
	}

	cb.failures = 0
	cb.state = Closed
	return nil
}

func (cb *CircuitBreaker) Status() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.state
}

func callExternalAPI(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	cb := NewCircuitBreaker(3, 10*time.Second)
	client := &http.Client{Timeout: 2 * time.Second}
	url := os.Getenv("PAYMENT_API_URL")
	if url == "" {
		url = "http://localhost:8112/v1/payments"
	}

	for i := 1; i <= 6; i++ {
		start := time.Now()

		err := cb.Execute(func() error {
			return callExternalAPI(client, url)
		})

		elapsed := time.Since(start)

		if err != nil {
			fmt.Println("call", i, "failed:", err, "| state:", cb.Status(), "| took:", elapsed)
		} else {
			fmt.Println("call", i, "success | state:", cb.Status(), "| took:", elapsed)
		}

		time.Sleep(500 * time.Millisecond)
	}
}
