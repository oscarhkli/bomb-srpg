package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRouting(t *testing.T) {
	mux := http.NewServeMux()
	serverState := NewServerStateManager()

	mux.HandleFunc("GET /api/archetypes", serverState.HandleGetAllArchetypes)
	mux.HandleFunc("POST /api/match-rooms", serverState.HandleCreateNewMatchRoom)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET /api/archetypes",
			method:     "GET",
			path:       "/api/archetypes",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /api/match-rooms",
			method:     "POST",
			path:       "/api/match-rooms",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "POST /api/archetypes (405)",
			method:     "POST",
			path:       "/api/archetypes",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "GET /api/match-rooms (405)",
			method:     "GET",
			path:       "/api/match-rooms",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /api/archetypes (405)",
			method:     "PUT",
			path:       "/api/archetypes",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /api/match-rooms (405)",
			method:     "DELETE",
			path:       "/api/match-rooms",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Unknown route",
			method:     "GET",
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Status code: got %d want %d", resp.StatusCode, tt.wantStatus)
			}

			// Consume body to allow connection reuse
			_, _ = io.Copy(io.Discard, resp.Body)
		})
	}
}
