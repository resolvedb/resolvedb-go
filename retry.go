package resolvedb

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"time"
)

// RetryConfig configures retry behavior with exponential backoff.
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retries (0 = no retries)
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	Multiplier     float64       // Backoff multiplier (e.g., 2.0 for doubling)
	JitterFactor   float64       // Jitter factor (0.0-1.0)
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
		JitterFactor:   0.2,
	}
}

// NoRetry returns a config that disables retries.
func NoRetry() RetryConfig {
	return RetryConfig{MaxRetries: 0}
}

// retryer handles retry logic with exponential backoff.
type retryer struct {
	config  RetryConfig
	attempt int
	rng     *rand.Rand
}

// newRetryer creates a new retryer.
func newRetryer(config RetryConfig) *retryer {
	// Use crypto/rand for secure seeding to prevent predictable backoff timing
	var seed int64
	if err := binary.Read(cryptorand.Reader, binary.BigEndian, &seed); err != nil {
		// Fallback to time-based seed if crypto/rand fails (should never happen)
		seed = time.Now().UnixNano()
	}
	return &retryer{
		config: config,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

// ShouldRetry returns true if the operation should be retried.
func (r *retryer) ShouldRetry(err error) bool {
	if r.attempt >= r.config.MaxRetries {
		return false
	}
	return IsRetryable(err)
}

// NextBackoff returns the duration to wait before the next retry.
func (r *retryer) NextBackoff() time.Duration {
	r.attempt++

	backoff := float64(r.config.InitialBackoff)
	for i := 1; i < r.attempt; i++ {
		backoff *= r.config.Multiplier
	}

	// Apply maximum limit
	if backoff > float64(r.config.MaxBackoff) {
		backoff = float64(r.config.MaxBackoff)
	}

	// Apply jitter: ±jitterFactor
	if r.config.JitterFactor > 0 {
		jitter := (r.rng.Float64()*2 - 1) * r.config.JitterFactor * backoff
		backoff += jitter
	}

	return time.Duration(backoff)
}

// Wait waits for the next backoff duration or until context is cancelled.
func (r *retryer) Wait(ctx context.Context) error {
	backoff := r.NextBackoff()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoff):
		return nil
	}
}

// Attempt returns the current attempt number (0-indexed).
func (r *retryer) Attempt() int {
	return r.attempt
}

// Reset resets the retry state.
func (r *retryer) Reset() {
	r.attempt = 0
}

// doWithRetry executes a function with retry logic.
func doWithRetry[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	r := newRetryer(config)
	var zero T

	for {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if !r.ShouldRetry(err) {
			return zero, err
		}

		if waitErr := r.Wait(ctx); waitErr != nil {
			return zero, waitErr
		}
	}
}
