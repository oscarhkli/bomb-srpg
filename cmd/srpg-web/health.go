package main

import (
	"context"
	"net/http"
)

func checkHealth(args []string) bool {
	return len(args) > 0 && args[0] == "healthcheck"
}

func runHealthCheck(ctx context.Context, url string) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
