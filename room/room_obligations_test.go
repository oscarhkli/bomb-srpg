package room

import (
	"bomb-srpg/engine"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"
)

var humanMove = engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 4, Y: 7})

func (f fixture) idle() bool {
	return f.r.Expired(f.clock.now(), time.Minute)
}

func (f fixture) ageAnHour() { f.clock.advance(time.Hour) }

type commandCase struct {
	name       string
	call       func(r *Room, token string) error
	humanIdx   int
	planIdx    int
	allowedCPU bool
}

func commandCases() []commandCase {
	return []commandCase{
		{
			name:     "Submit",
			call:     func(r *Room, token string) error { _, err := r.Submit(humanMove, token); return err },
			humanIdx: 0, planIdx: 0,
		},
		{
			name:     "Reset",
			call:     func(r *Room, token string) error { return r.Reset(token) },
			humanIdx: 0, planIdx: 1,
		},
		{
			name:     "Resolve",
			call:     func(r *Room, token string) error { _, _, err := r.Resolve(token); return err },
			humanIdx: 0, planIdx: 1,
		},
		{
			name:     "Surrender",
			call:     func(r *Room, token string) error { _, err := r.Surrender(1, token); return err },
			humanIdx: 0, planIdx: 0, allowedCPU: true,
		},
	}
}

func TestRoom_Obligations(t *testing.T) {
	covered := map[string]bool{}

	t.Run("commands", func(t *testing.T) {
		for _, c := range commandCases() {
			covered[c.name] = true
			t.Run(c.name, func(t *testing.T) {
				t.Run("wrong token is rejected", func(t *testing.T) {
					f := humanFixture(t)
					if err := c.call(f.r, "x"); !errors.Is(err, ErrInvalidToken) {
						t.Errorf("error = %v, want %v", err, ErrInvalidToken)
					}
				})
				t.Run("match missing is reported before the token check", func(t *testing.T) {
					f := newFixture()
					if err := c.call(f.r, "x"); !errors.Is(err, ErrMatchNotFound) {
						t.Errorf("error = %v, want %v", err, ErrMatchNotFound)
					}
				})
				if c.allowedCPU {
					t.Run("allowed in Planning", func(t *testing.T) {
						f := planningFixture(t)
						if err := c.call(f.r, f.tokens[c.planIdx]); err != nil {
							t.Errorf("error = %v, want nil", err)
						}
					})
				} else {
					t.Run("rejected in Planning", func(t *testing.T) {
						f := planningFixture(t)
						if err := c.call(f.r, f.tokens[c.planIdx]); !errors.Is(err, ErrCPUTurnInProgress) {
							t.Errorf("error = %v, want %v", err, ErrCPUTurnInProgress)
						}
					})
					t.Run("a Planning rejection leaves activity alone", func(t *testing.T) {
						f := planningFixture(t)
						f.ageAnHour()
						_ = c.call(f.r, f.tokens[c.planIdx])
						if !f.idle() {
							t.Error("activity was stamped by a rejected call")
						}
					})
				}
				t.Run("success stamps activity", func(t *testing.T) {
					f := humanFixture(t)
					f.ageAnHour()
					if err := c.call(f.r, f.tokens[c.humanIdx]); err != nil {
						t.Fatalf("error = %v, want nil", err)
					}
					if f.idle() {
						t.Error("activity was not stamped")
					}
				})
				t.Run("a wrong token leaves activity alone", func(t *testing.T) {
					f := humanFixture(t)
					f.ageAnHour()
					_ = c.call(f.r, "x")
					if !f.idle() {
						t.Error("activity was stamped by a rejected call")
					}
				})
			})
		}

		t.Run("Submit a command the engine rejects leaves activity alone", func(t *testing.T) {
			f := humanFixture(t)
			f.ageAnHour()
			farAway := engine.NewMoveCommand(engine.NewUnitID(1, 0), engine.Coordinate{X: 99, Y: 99})
			if _, err := f.r.Submit(farAway, f.tokens[0]); !errors.Is(err, ErrInvalidTurnCmd) {
				t.Fatalf("error = %v, want %v", err, ErrInvalidTurnCmd)
			}
			if !f.idle() {
				t.Error("activity was stamped by a rejected command")
			}
		})
		t.Run("Surrender team 3 on a never-started room is an invalid config", func(t *testing.T) {
			f := newFixture()
			if _, err := f.r.Surrender(3, "x"); !errors.Is(err, ErrInvalidConfig) {
				t.Errorf("error = %v, want %v", err, ErrInvalidConfig)
			}
		})
	})

	t.Run("reads", func(t *testing.T) {
		reads := []struct {
			name string
			call func(r *Room) error
		}{
			{"State", func(r *Room) error { _, err := r.State(); return err }},
			{"Config", func(r *Room) error { _, err := r.Config(); return err }},
			{"AllowedTiles", func(r *Room) error {
				_, err := r.AllowedTiles(engine.NewUnitID(1, 0), engine.TurnCmdMove)
				return err
			}},
		}
		for _, c := range reads {
			covered[c.name] = true
			t.Run(c.name, func(t *testing.T) {
				t.Run("match missing", func(t *testing.T) {
					f := newFixture()
					if err := c.call(f.r); !errors.Is(err, ErrMatchNotFound) {
						t.Errorf("error = %v, want %v", err, ErrMatchNotFound)
					}
				})
				t.Run("success leaves activity alone", func(t *testing.T) {
					f := humanFixture(t)
					f.ageAnHour()
					if err := c.call(f.r); err != nil {
						t.Fatalf("error = %v, want nil", err)
					}
					if !f.idle() {
						t.Error("a read stamped activity")
					}
				})
			})
		}
	})

	t.Run("ConsumeCPU", func(t *testing.T) {
		covered["ConsumeCPU"] = true
		t.Run("wrong token is rejected", func(t *testing.T) {
			f := humanFixture(t)
			if _, err := f.r.ConsumeCPU("x"); !errors.Is(err, ErrInvalidToken) {
				t.Errorf("error = %v, want %v", err, ErrInvalidToken)
			}
		})
		t.Run("either team's token is accepted", func(t *testing.T) {
			f := humanFixture(t)
			for i, token := range f.tokens {
				if _, err := f.r.ConsumeCPU(token); err != nil {
					t.Errorf("team %d token: error = %v, want nil", i+1, err)
				}
			}
		})
		t.Run("match missing is reported before the token check", func(t *testing.T) {
			f := newFixture()
			if _, err := f.r.ConsumeCPU("x"); !errors.Is(err, ErrMatchNotFound) {
				t.Errorf("error = %v, want %v", err, ErrMatchNotFound)
			}
		})
	})

	lifecycle := []struct {
		name string
		call func(r *Room, token string) error
	}{
		{"Rematch", func(r *Room, token string) error { _, err := r.Rematch(token); return err }},
		{"Delete", func(r *Room, token string) error { return r.Delete(token) }},
	}
	for _, c := range lifecycle {
		covered[c.name] = true
		t.Run(c.name, func(t *testing.T) {
			t.Run("wrong token is rejected", func(t *testing.T) {
				f := humanFixture(t)
				if err := c.call(f.r, "x"); !errors.Is(err, ErrInvalidToken) {
					t.Errorf("error = %v, want %v", err, ErrInvalidToken)
				}
			})
			t.Run("either team's token passes the token check", func(t *testing.T) {
				f := humanFixture(t)
				for i, token := range f.tokens {
					if err := c.call(f.r, token); errors.Is(err, ErrInvalidToken) {
						t.Errorf("team %d token: error = %v", i+1, err)
					}
				}
			})
			t.Run("token first on a never-started room", func(t *testing.T) {
				f := newFixture()
				if err := c.call(f.r, "x"); !errors.Is(err, ErrInvalidToken) {
					t.Errorf("error = %v, want %v", err, ErrInvalidToken)
				}
			})
			t.Run("a wrong token leaves activity alone", func(t *testing.T) {
				f := humanFixture(t)
				f.ageAnHour()
				_ = c.call(f.r, "x")
				if !f.idle() {
					t.Error("activity was stamped by a rejected call")
				}
			})
		})
	}
	t.Run("Rematch success stamps activity", func(t *testing.T) {
		f := humanFixture(t)
		f.ageAnHour()
		if _, err := f.r.Rematch(f.tokens[0]); err != nil {
			t.Fatalf("Rematch() error = %v", err)
		}
		if f.idle() {
			t.Error("activity was not stamped")
		}
	})

	t.Run("every exported method is covered", func(t *testing.T) {
		exceptions := map[string]string{
			"Start":   "takes no token and builds the match",
			"Expired": "needs no match and no token",
		}
		var uncovered []string
		typ := reflect.TypeFor[*Room]()
		for method := range typ.Methods() {
			name := method.Name
			if _, ok := exceptions[name]; !ok && !covered[name] {
				uncovered = append(uncovered, name)
			}
		}
		sort.Strings(uncovered)
		if len(uncovered) > 0 {
			t.Errorf("exported methods with no row in the obligations table: %v", uncovered)
		}
	})
}

func TestRoom_StampsActivity(t *testing.T) {
	t.Run("Start", func(t *testing.T) {
		f := newFixture()
		f.ageAnHour()
		if _, err := f.r.Start(validGameCfg()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if f.idle() {
			t.Error("activity was not stamped")
		}
	})
	t.Run("the CPU turn", func(t *testing.T) {
		f := planningFixture(t)
		f.ageAnHour()
		f.held.runAll()
		if f.idle() {
			t.Error("activity was not stamped")
		}
	})
}
