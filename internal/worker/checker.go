package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"pingme-golang/internal/models"
)

type Checker interface {
	Check(ctx context.Context, monitor models.Monitor) CheckResult
}

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (c *HTTPChecker) Check(ctx context.Context, monitor models.Monitor) CheckResult {
	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now().UTC()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, monitor.URL, nil)
	if err != nil {
		return CheckResult{
			Success:      false,
			ErrorMessage: err.Error(),
			CheckedAt:    startedAt,
		}
	}

	req.Header.Set("User-Agent", "PingMe-Worker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return CheckResult{
			Success:        false,
			ErrorMessage:   err.Error(),
			ResponseTimeMs: int(time.Since(startedAt).Milliseconds()),
			CheckedAt:      time.Now().UTC(),
		}
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	errorMessage := ""
	if !success {
		errorMessage = fmt.Sprintf("http status %d", resp.StatusCode)
	}

	return CheckResult{
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: int(time.Since(startedAt).Milliseconds()),
		Success:        success,
		ErrorMessage:   errorMessage,
		CheckedAt:      time.Now().UTC(),
	}
}
