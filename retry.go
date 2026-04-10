package colony

import "time"

// RetryConfig controls automatic retry behaviour.
type RetryConfig struct {
	// MaxRetries is the number of retries after the initial attempt. Default: 2.
	MaxRetries int
	// BaseDelay is the initial backoff delay. Default: 1s.
	BaseDelay time.Duration
	// MaxDelay caps the backoff delay. Default: 10s.
	MaxDelay time.Duration
	// RetryOn is the set of HTTP status codes that trigger a retry.
	// Default: {429, 502, 503, 504}.
	RetryOn map[int]bool
}

// DefaultRetry returns the default retry configuration.
func DefaultRetry() RetryConfig {
	return RetryConfig{
		MaxRetries: 2,
		BaseDelay:  1 * time.Second,
		MaxDelay:   10 * time.Second,
		RetryOn: map[int]bool{
			429: true,
			502: true,
			503: true,
			504: true,
		},
	}
}

func (r RetryConfig) shouldRetry(status int) bool {
	return r.RetryOn[status]
}

func (r RetryConfig) delay(attempt int) time.Duration {
	d := r.BaseDelay
	for i := 0; i < attempt; i++ {
		d *= 2
	}
	if d > r.MaxDelay {
		d = r.MaxDelay
	}
	return d
}
