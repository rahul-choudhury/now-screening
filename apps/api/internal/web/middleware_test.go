package web

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddlewareLogsRecoveredPanics(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)

	handler := Chain(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("boom")
		}),
		CORSMiddleware(),
		LoggingMiddleware(logger),
		RecoverMiddleware(logger),
	)

	req := httptest.NewRequest(http.MethodGet, "/movies?city=cuttack", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, "panic while serving GET /movies?city=cuttack: boom") {
		t.Fatalf("logs = %q, want panic log", logOutput)
	}

	if !strings.Contains(logOutput, "GET /movies?city=cuttack 500 ") {
		t.Fatalf("logs = %q, want access log with 500", logOutput)
	}
}
