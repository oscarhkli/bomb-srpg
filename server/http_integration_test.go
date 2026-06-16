package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRouting(t *testing.T) {
	mux := http.NewServeMux()
	serverState := NewServerStateManager()

	mux.HandleFunc("GET /api/archetypes", serverState.HandleGetAllArchetypes)

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("GET /api/archetypes returns 200", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/archetypes")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /api/archetypes: got %d want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("POST /api/archetypes returns 405", func(t *testing.T) {
		req, _ := http.NewRequest("POST", server.URL+"/api/archetypes", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("POST /api/archetypes: got %d want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})

	t.Run("unknown route returns 404", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/unknown")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET /unknown: got %d want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}
