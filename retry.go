package dagcuter

import (
	"context"
	"fmt"
	"math"
	"time"
)

type RetryPolicy struct {
	Interval    time.Duration `json:"interval" yaml:"interval"`
	MaxInterval time.Duration `json:"maxInterval" yaml:"maxInterval"`
	MaxAttempts int           `json:"maxAttempts" yaml:"maxAttempts"`
	Multiplier  float64       `json:"multiplier" yaml:"multiplier"`
}

type RetryExecutor struct {
	policy *RetryPolicy
}

func (d *Dagcuter) newRetryExecutor(policy *RetryPolicy) *RetryExecutor {
	if policy == nil {
		// Default strategy: no retry, execute only once
		policy = &RetryPolicy{
			MaxAttempts: -1,
		}
	}
	if policy.Interval <= 0 {
		policy.Interval = 1 * time.Second // Default interval of 1 second
	}
	if policy.MaxInterval <= 0 {
		policy.MaxInterval = 30 * time.Second // Default maximum interval of 30 seconds
	}
	if policy.Multiplier <= 0 {
		policy.Multiplier = 2.0 // Default multiplier of 2
	}
	return &RetryExecutor{policy: policy}
}

// ExecuteWithRetry executes a function with retry logic based on the provided policy.
func (r *RetryExecutor) ExecuteWithRetry(ctx context.Context, taskName string, fn func(n int) error) error {
	if r.policy.MaxAttempts <= 0 {
		// No retry policy, execute directly
		return fn(0)
	}

	if r.policy.MaxInterval > 150*time.Second {
		r.policy.MaxInterval = 150 * time.Second // Default maximum interval of 150 seconds
	}

	var lastErr error
	for attempt := 1; attempt <= r.policy.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry attempt %d: %w", attempt, ctx.Err())
		default:
		}

		// Execute the task
		if err := fn(attempt); err == nil {
			// Task executed successfully
			return nil
		} else {
			lastErr = err
		}

		// If this is the last attempt, no need to wait
		if attempt == r.policy.MaxAttempts {
			break
		}

		// Calculate the backoff time using the retry policy
		waitTime := r.calculateBackoff(attempt, r.policy.MaxInterval)

		// Wait before the next retry
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		case <-time.After(waitTime):
			// Continue to retry
		}
	}

	return fmt.Errorf("task %s failed after %d attempts, last error: %w",
		taskName, r.policy.MaxAttempts, lastErr)
}

// calculateBackoff calculates the backoff time based on the retry policy.
func (r *RetryExecutor) calculateBackoff(attempt int, maxInterval time.Duration) time.Duration {
	// Calculate the backoff time using exponential backoff formula
	// Formula: baseInterval * (multiplier ^ (attempt - 1))
	// Use math.Pow for accurate floating point exponentiation
	backoff := float64(r.policy.Interval) * math.Pow(r.policy.Multiplier, float64(attempt-1))

	result := time.Duration(backoff)

	// Ensure the result does not exceed the maximum interval
	if result > maxInterval {
		return maxInterval
	}

	return result
}
