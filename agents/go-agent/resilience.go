package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	JitterRange float64
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  5,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		JitterRange: 0.1,
	}
}

// RetryWithBackoff executes a function with exponential backoff
func RetryWithBackoff(ctx context.Context, config *RetryConfig, operation string, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateBackoff(attempt, config)
			log.Printf("Retry %d/%d for %s after %v (last error: %v)", 
				attempt, config.MaxRetries, operation, delay, lastErr)
			
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		lastErr = fn()
		if lastErr == nil {
			if attempt > 0 {
				log.Printf("✓ %s succeeded after %d retries", operation, attempt)
			}
			return nil
		}
		
		// Check if error is retryable
		if !isRetryable(lastErr) {
			log.Printf("Non-retryable error for %s: %v", operation, lastErr)
			return lastErr
		}
	}
	
	return fmt.Errorf("%s failed after %d retries: %w", operation, config.MaxRetries, lastErr)
}

// calculateBackoff returns the delay for a given attempt with jitter
func calculateBackoff(attempt int, config *RetryConfig) time.Duration {
	// Exponential backoff: baseDelay * (multiplier ^ attempt)
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt-1))
	
	// Apply maximum delay cap
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	// Add jitter (±10% by default)
	jitter := delay * config.JitterRange * (2*float64(time.Now().UnixNano()%1000)/1000 - 1)
	delay += jitter
	
	return time.Duration(delay)
}

// isRetryable determines if an error should be retried
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	// Check gRPC status codes
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded:
			return true
		case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.PermissionDenied:
			return false
		default:
			return false
		}
	}
	
	// Check for common transient errors
	errStr := err.Error()
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"deadline exceeded",
		"temporary failure",
		"try again",
	}
	
	for _, pattern := range transientPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)*2 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name          string
	maxFailures   int
	resetTimeout  time.Duration
	halfOpenMax   int
	
	mu            sync.RWMutex
	state         CircuitState
	failures      int
	lastFailTime  time.Time
	halfOpenCount int
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		halfOpenMax:  3,
		state:        StateClosed,
	}
}

// Execute runs a function through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}
	
	err := fn()
	cb.afterRequest(err)
	return err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			log.Printf("Circuit breaker '%s': Transitioning to HALF_OPEN", cb.name)
			cb.state = StateHalfOpen
			cb.halfOpenCount = 0
			return nil
		}
		return fmt.Errorf("circuit breaker '%s' is OPEN", cb.name)
		
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.halfOpenMax {
			return fmt.Errorf("circuit breaker '%s' HALF_OPEN limit reached", cb.name)
		}
		cb.halfOpenCount++
		return nil
		
	case StateClosed:
		return nil
	}
	
	return nil
}

func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()
		
		switch cb.state {
		case StateClosed:
			if cb.failures >= cb.maxFailures {
				log.Printf("Circuit breaker '%s': Too many failures (%d), opening circuit", 
					cb.name, cb.failures)
				cb.state = StateOpen
			}
			
		case StateHalfOpen:
			log.Printf("Circuit breaker '%s': Failure in HALF_OPEN, reopening", cb.name)
			cb.state = StateOpen
			cb.halfOpenCount = 0
		}
	} else {
		// Success
		switch cb.state {
		case StateClosed:
			// Reset failure count on success
			if cb.failures > 0 {
				cb.failures = 0
			}
			
		case StateHalfOpen:
			// After successful requests in half-open, close the circuit
			if cb.halfOpenCount >= cb.halfOpenMax {
				log.Printf("Circuit breaker '%s': Requests successful in HALF_OPEN, closing circuit", 
					cb.name)
				cb.state = StateClosed
				cb.failures = 0
				cb.halfOpenCount = 0
			}
		}
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// DialWithRetry creates a gRPC connection with retry logic
func DialWithRetry(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	config := DefaultRetryConfig()
	config.MaxRetries = 10
	config.MaxDelay = 60 * time.Second
	
	var conn *grpc.ClientConn
	err := RetryWithBackoff(ctx, config, fmt.Sprintf("connect to %s", target), func() error {
		var dialErr error
		conn, dialErr = grpc.Dial(target, opts...)
		return dialErr
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
	}
	
	log.Printf("✓ Connected to %s", target)
	return conn, nil
}

// Fallback provides a fallback mechanism
type Fallback struct {
	primary   func() error
	fallbacks []func() error
}

// NewFallback creates a new fallback handler
func NewFallback(primary func() error, fallbacks ...func() error) *Fallback {
	return &Fallback{
		primary:   primary,
		fallbacks: fallbacks,
	}
}

// Execute tries primary first, then fallbacks
func (f *Fallback) Execute() error {
	err := f.primary()
	if err == nil {
		return nil
	}
	
	log.Printf("Primary operation failed: %v, trying fallbacks...", err)
	
	for i, fallback := range f.fallbacks {
		fallbackErr := fallback()
		if fallbackErr == nil {
			log.Printf("✓ Fallback %d succeeded", i+1)
			return nil
		}
		log.Printf("Fallback %d failed: %v", i+1, fallbackErr)
	}
	
	return fmt.Errorf("all fallbacks failed, last error: %w", err)
}

var ErrCircuitOpen = errors.New("circuit breaker is open")

