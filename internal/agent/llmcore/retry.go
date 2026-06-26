package llmcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultProviderRetryMaxAttempts = 3
	defaultProviderRetryBaseDelay   = 50 * time.Millisecond
	defaultProviderRetryMaxDelay    = 200 * time.Millisecond
)

type ProviderStatusError struct {
	MessagePrefix string
	StatusCode    int
	Status        string
	Body          string
}

func (e *ProviderStatusError) Error() string {
	return fmt.Sprintf("%s API error: %s - %s", e.MessagePrefix, e.Status, strings.TrimSpace(e.Body))
}

type ProviderAPIError struct {
	MessagePrefix string
	Code          string
	Message       string
}

func (e *ProviderAPIError) Error() string {
	return fmt.Sprintf("%s API error: %s - %s", e.MessagePrefix, strings.TrimSpace(e.Code), strings.TrimSpace(e.Message))
}

func providerRetryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return defaultProviderRetryBaseDelay
	}
	delay := defaultProviderRetryBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= defaultProviderRetryMaxDelay {
			return defaultProviderRetryMaxDelay
		}
	}
	if delay > defaultProviderRetryMaxDelay {
		return defaultProviderRetryMaxDelay
	}
	return delay
}

func waitProviderRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type retryNotifierKey struct{}

type RetryNotifyFunc func(attempt, maxAttempts int, providerName, operation string)

func WithRetryNotifier(ctx context.Context, fn RetryNotifyFunc) context.Context {
	return context.WithValue(ctx, retryNotifierKey{}, fn)
}

func RetryNotifierFromContext(ctx context.Context) RetryNotifyFunc {
	if fn, ok := ctx.Value(retryNotifierKey{}).(RetryNotifyFunc); ok {
		return fn
	}
	return nil
}

func RetryProviderCall[T any](ctx context.Context, providerName, operation string, fn func() (T, error), onRetry ...RetryNotifyFunc) (T, error) {
	var zero T
	var lastErr error

	for attempt := 1; attempt <= defaultProviderRetryMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		result, err := fn()
		if err == nil {
			return result, nil
		}
		classified := classifyProviderError(providerName, operation, err)
		if classified == nil {
			return zero, err
		}
		if !classified.IsRetryable() {
			return zero, classified
		}
		lastErr = classified
		if attempt == defaultProviderRetryMaxAttempts {
			break
		}
		for _, notify := range onRetry {
			if notify != nil {
				notify(attempt, defaultProviderRetryMaxAttempts, providerName, operation)
			}
		}
		if err := waitProviderRetry(ctx, providerRetryDelay(attempt)); err != nil {
			return zero, err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unknown provider retry failure")
	}
	return zero, NewRetryableError(fmt.Sprintf("%s %s failed after %d attempts", providerName, operation, defaultProviderRetryMaxAttempts), lastErr)
}

func classifyProviderError(providerName, operation string, err error) *AgentError {
	if err == nil {
		return nil
	}
	if ae := AsAgentError(err); ae != nil {
		return ae
	}

	var statusErr *ProviderStatusError
	if errors.As(err, &statusErr) {
		if isTransientStatusCode(statusErr.StatusCode) {
			return NewRetryableError(fmt.Sprintf("%s %s hit transient status %d", providerName, operation, statusErr.StatusCode), statusErr)
		}
		return NewFatalError(fmt.Sprintf("%s %s rejected with status %d", providerName, operation, statusErr.StatusCode), statusErr)
	}

	var apiErr *ProviderAPIError
	if errors.As(err, &apiErr) {
		if isTransientProviderCode(apiErr.Code) {
			return NewRetryableError(fmt.Sprintf("%s %s hit transient provider error %s", providerName, operation, strings.TrimSpace(apiErr.Code)), apiErr)
		}
		return NewFatalError(fmt.Sprintf("%s %s returned a non-retriable provider error", providerName, operation), apiErr)
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return NewFatalError(fmt.Sprintf("%s %s canceled", providerName, operation), err)
	}
	if isTransientNetworkError(err) {
		return NewRetryableError(fmt.Sprintf("%s %s encountered a transient network error", providerName, operation), err)
	}
	return NewFatalError(fmt.Sprintf("%s %s failed", providerName, operation), err)
}

func isTransientStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isTransientProviderCode(code string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(code))
	if trimmed == "" {
		return false
	}
	if n, err := strconv.Atoi(trimmed); err == nil {
		return isTransientStatusCode(n)
	}
	for _, marker := range []string{"429", "500", "502", "503", "504", "rate_limit", "too_many_requests", "temporarily_unavailable"} {
		if strings.Contains(trimmed, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		if urlErr.Err != nil {
			return isTransientNetworkError(urlErr.Err)
		}
	}

	var statusErr *ProviderStatusError
	if errors.As(err, &statusErr) {
		return false
	}
	var apiErr *ProviderAPIError
	if errors.As(err, &apiErr) {
		return false
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	for _, marker := range []string{"connection reset", "connection refused", "broken pipe", "unexpected eof", "timeout", "temporarily unavailable", "temporary failure", "no such host", "dial tcp"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
