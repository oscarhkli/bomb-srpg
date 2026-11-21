package room

import (
	"bomb-srpg/engine"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestRoom_WriteMethodsHoldExclusiveLockAgainstReadsAndExpiry(t *testing.T) {
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
					_ = r.Expired(time.Now(), time.Hour)
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
		tokens, err = r.Start(humanTurnCfg())
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		tokens, err = r.Rematch(tokens[0])
		if err != nil {
			t.Fatalf("Rematch() error = %v", err)
		}
	}
	close(stop)
	readers.Wait()
}
