package room

import (
	"bomb-srpg/engine"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	cur time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{cur: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cur
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cur = c.cur.Add(d)
}

func newClockedRoom(clock *fakeClock) *Room {
	return New("room", Deps{
		Logger:   slog.New(slog.DiscardHandler),
		NewMatch: engine.InitGame,
		Now:      clock.now,
	})
}

func TestRoom_Expired(t *testing.T) {
	const idle = time.Hour
	tests := []struct {
		name    string
		advance time.Duration
		want    bool
	}{
		{"fresh room", 0, false},
		{"exactly at the limit", idle, false},
		{"idle exceeded", idle + time.Nanosecond, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := newFakeClock()
			r := newClockedRoom(clock)

			clock.advance(tt.advance)

			if got := r.Expired(clock.now(), idle); got != tt.want {
				t.Errorf("Expired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoom_Expired_OnlyTheAgedRoom(t *testing.T) {
	clock := newFakeClock()
	aged := newClockedRoom(clock)
	clock.advance(2 * time.Hour)
	fresh := newClockedRoom(clock)

	if got := aged.Expired(clock.now(), time.Hour); !got {
		t.Errorf("aged room Expired() = %v, want true", got)
	}
	if got := fresh.Expired(clock.now(), time.Hour); got {
		t.Errorf("fresh room Expired() = %v, want false", got)
	}
}

func TestRoom_ExpiredRacesWithReads(t *testing.T) {
	cfg := validGameCfg()
	cfg.AllowResetTurn = true
	r := New("room", Deps{Logger: slog.New(slog.DiscardHandler), NewMatch: engine.InitGame})
	tokens, err := r.Start(cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	move := engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 4, Y: 7})

	const iterations = 300
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range iterations {
			if err := r.Reset(tokens[0]); err != nil {
				t.Errorf("Reset() error = %v", err)
				return
			}
			if _, err := r.Submit(move, tokens[0]); err != nil {
				t.Errorf("Submit() error = %v", err)
				return
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
			r.Expired(time.Now(), time.Hour)
		}
	}
}
