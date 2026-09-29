package main

import "net/http"

func checkHealth(args []string) bool {
	return len(args) > 0 && args[0] == "healthcheck"
}

func runHealthCheck(url string) int {
	resp, err := http.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
