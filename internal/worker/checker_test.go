package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pingme-golang/internal/models"
)

func TestHTTPChecker_CheckUsesOnly2xxAsSuccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		wantSuccess      bool
		wantErrorMessage string
	}{
		{
			name:        "2xx is success",
			statusCode:  http.StatusNoContent,
			wantSuccess: true,
		},
		{
			name:             "3xx is failure",
			statusCode:       http.StatusFound,
			wantSuccess:      false,
			wantErrorMessage: "http status 302",
		},
		{
			name:             "5xx is failure",
			statusCode:       http.StatusServiceUnavailable,
			wantSuccess:      false,
			wantErrorMessage: "http status 503",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.statusCode)
			}))
			t.Cleanup(server.Close)

			checker := NewHTTPChecker()
			checker.client = server.Client()

			result := checker.Check(context.Background(), models.Monitor{
				URL:     server.URL,
				Timeout: 1,
			})

			if result.StatusCode != testCase.statusCode {
				t.Fatalf("statusCode = %d, want %d", result.StatusCode, testCase.statusCode)
			}
			if result.Success != testCase.wantSuccess {
				t.Fatalf("success = %t, want %t", result.Success, testCase.wantSuccess)
			}
			if result.ErrorMessage != testCase.wantErrorMessage {
				t.Fatalf("errorMessage = %q, want %q", result.ErrorMessage, testCase.wantErrorMessage)
			}
			if result.CheckedAt.IsZero() {
				t.Fatal("checkedAt is zero")
			}
		})
	}
}
