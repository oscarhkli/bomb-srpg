package room

import (
	"bomb-srpg/engine"
	"log/slog"
	"sync"
	"testing"
)

func TestRoom_WriteMethodsHoldExclusiveLock(t *testing.T) {
	r := New("room", Deps{Logger: slog.New(slog.DiscardHandler), NewMatch: engine.InitGame})
	tokens, err := r.Start(humanTurnCfg())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for range 3 {
		readers.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = r.State()
					_, _ = r.Config()
					_, _ = r.AllowedTiles(engine.NewUnitID(1, 0), engine.TurnCmdMove)
				}
			}
		})
	}

	const cycles = 300
	for range cycles {
		_, _ = r.Submit(humanMove, tokens[0])
		_ = r.Reset(tokens[0])
		_, _, _ = r.Resolve(tokens[0])
		_, _ = r.Surrender(1, tokens[0])
		_ = r.Delete(tokens[0])
		_, _ = r.Rematch(tokens[0])
	}
	close(stop)
	readers.Wait()
}
