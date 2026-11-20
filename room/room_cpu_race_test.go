package room

import (
	"bomb-srpg/engine"
	"sync"
	"testing"
)

func TestRoom_ConsumeCPURacesTheCPUTurn(t *testing.T) {
	f := startedFixture(t, vsCPUCfg(), cpuTurnPending)
	f.r.deps.Spawn = func(run func()) { go run() }

	ready := make(chan struct{})
	var once sync.Once
	var consumers sync.WaitGroup
	for range 3 {
		consumers.Go(func() {
			for {
				select {
				case <-ready:
					return
				default:
				}
				status, err := f.r.ConsumeCPU(f.tokens[0])
				if err == nil && status.Phase == engine.TurnPhaseReady {
					once.Do(func() { close(ready) })
				}
			}
		})
	}
	consumers.Wait()
}
