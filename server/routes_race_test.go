package server

import (
	"net/http"
	"sync"
	"testing"
)

func TestRegisterRoutesConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			RegisterRoutes(http.NewServeMux(), NewHandler(NewServerStateManager()))
		})
	}
	wg.Wait()
}
