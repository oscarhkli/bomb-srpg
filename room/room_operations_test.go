package room

import (
	"bomb-srpg/engine"
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestRoom_Start(t *testing.T) {
	tests := []struct {
		name    string
		started bool
		cfg     engine.GameCfg
		wantErr error
	}{
		{"Success", false, validGameCfg(), nil},
		{"Match Already Exists", true, validGameCfg(), ErrMatchExists},
		{"Invalid Config", false, engine.GameCfg{StagePreset: "INVALID_STAGE", MaxTurns: 10}, ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			if tt.started {
				if _, err := f.r.Start(validGameCfg()); err != nil {
					t.Fatalf("Start() error = %v", err)
				}
			}
			tokens, err := f.r.Start(tt.cfg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Start() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if tokens[0] == "" || tokens[1] == "" || tokens[0] == tokens[1] {
				t.Errorf("Start() tokens = %v, want 2 unique non-empty tokens", tokens)
			}
			got, err := f.r.Config()
			if err != nil {
				t.Fatalf("Config() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.cfg) {
				t.Errorf("Config() = %+v, want the config passed to Start %+v", got, tt.cfg)
			}
			assertTokensAuthorize(t, f.r, tokens)
		})
	}
}

func TestRoom_Rematch(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*Room, [2]string)
		wantErr error
	}{
		{
			name: "Success - Match not exist but with previous gameCfg",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				r, tokens := startedRoom(t)
				endAndDelete(t, r, tokens[0])
				return r, tokens
			},
		},
		{
			name: "Success - Wipe existing Match",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				return startedRoom(t)
			},
		},
		{
			name: "Invalid token",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				r, _ := startedRoom(t)
				return r, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "Failure - No previous match",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				return newFixture().r, [2]string{}
			},
			wantErr: ErrMatchNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, tokens := tt.setup(t)
			newTokens, err := r.Rematch(tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Rematch() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if newTokens[0] == "" || newTokens[1] == "" || newTokens[0] == newTokens[1] {
				t.Errorf("Rematch() tokens = %v, want 2 unique non-empty tokens", newTokens)
			}
			got, err := r.Config()
			if err != nil {
				t.Fatalf("Config() error = %v", err)
			}
			if want := validGameCfg(); !reflect.DeepEqual(got, want) {
				t.Errorf("Config() = %+v, want %+v", got, want)
			}
			assertTokensAuthorize(t, r, newTokens)
		})
	}
}

func TestRoom_Delete(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (*Room, [2]string)
		wantErr error
	}{
		{
			name: "Success - Without existing Match",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				r, tokens := startedRoom(t)
				endAndDelete(t, r, tokens[0])
				return r, tokens
			},
		},
		{
			name: "Success - Existing Match",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				r, tokens := startedRoom(t)
				if _, err := r.Surrender(1, tokens[0]); err != nil {
					t.Fatalf("Surrender() error = %v", err)
				}
				return r, tokens
			},
		},
		{
			name: "Failure - Match still in progress",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				return startedRoom(t)
			},
			wantErr: ErrMatchInProgress,
		},
		{
			name: "Invalid token",
			setup: func(t *testing.T) (*Room, [2]string) {
				t.Helper()
				r, _ := startedRoom(t)
				return r, [2]string{"INVALID_TOKEN", ""}
			},
			wantErr: ErrInvalidToken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, tokens := tt.setup(t)
			err := r.Delete(tokens[0])
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				if _, err := r.State(); !errors.Is(err, ErrMatchNotFound) {
					t.Fatalf("State() error = %v, want %v", err, ErrMatchNotFound)
				}
			}
		})
	}
}

func TestRoom_State(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, _ := startedRoom(t)
		fresh, err := engine.InitGame(validGameCfg())
		if err != nil {
			t.Fatalf("InitGame() error = %v", err)
		}
		gs := mustState(t, r)
		if got, want := gs.Turn, fresh.WorkingState.Turn; got != want {
			t.Errorf("Expected Turn %d, got %d", want, got)
		}
		if got, want := gs.ActiveTeam, fresh.WorkingState.ActiveTeam; got != want {
			t.Errorf("Expected ActiveTeam %d, got %d", want, got)
		}
		if got, want := len(gs.Units), len(fresh.WorkingState.Units); got != want {
			t.Errorf("Expected %d units, got %d", want, got)
		}
	})

	t.Run("Mid-Turn Progress Survives The Copy", func(t *testing.T) {
		r, _ := startedRoom(t, suddenDeathMidTurn)
		gs := mustState(t, r)
		if !gs.InSuddenDeath {
			t.Error("Expected InSuddenDeath to survive the copy, got false")
		}
		for unitID, u := range gs.Units {
			if !u.HasMoved || !u.HasUsedSkill {
				t.Errorf("Expected unit %#x action economy to survive the copy, got HasMoved %v, HasUsedSkill %v", unitID, u.HasMoved, u.HasUsedSkill)
			}
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		if _, err := newFixture().r.State(); !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("State() error = %v, want %v", err, ErrMatchNotFound)
		}
	})
}

func TestRoom_Submit(t *testing.T) {
	unitID := engine.NewUnitID(1, 0)
	newPos := engine.Coordinate{X: 4, Y: 7}

	t.Run("Success", func(t *testing.T) {
		r, tokens := startedRoom(t)
		gameEvents, err := r.Submit(engine.NewMoveCommand(unitID, newPos), tokens[0])
		if err != nil {
			t.Fatalf("Submit() error = %v", err)
		}
		if gotPos := mustState(t, r).Units[unitID].Position; gotPos != newPos {
			t.Errorf("Expected Unit %#X new position %#v, got %#v", unitID, newPos, gotPos)
		}
		if len(gameEvents) != 1 {
			t.Fatalf("expected 1 GameEvent returned, got %d", len(gameEvents))
		}
		resEvt := gameEvents[0]
		validFrom := engine.Coordinate{X: 4, Y: 8}
		if resEvt != (engine.UnitMoved{UnitID: unitID, From: validFrom, To: newPos}) {
			t.Errorf("malformed UnitMoveEvent returned: %+v", resEvt)
		}
	})

	t.Run("Invalid TurnCommand (out of range)", func(t *testing.T) {
		r, tokens := startedRoom(t)
		gameEvents, err := r.Submit(engine.NewMoveCommand(unitID, engine.Coordinate{X: 4, Y: 7777}), tokens[0])
		if !errors.Is(err, ErrInvalidTurnCmd) {
			t.Fatalf("Submit() error = %v, want %v", err, ErrInvalidTurnCmd)
		}
		if gotPos := mustState(t, r).Units[unitID].Position; gotPos.X == 4 && gotPos.Y == 7777 {
			t.Errorf("Expected Unit %#X didn't move", unitID)
		}
		if len(gameEvents) > 0 {
			t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
		}
	})

	t.Run("Invalid TeamID (out of range)", func(t *testing.T) {
		r, _ := startedRoom(t)
		_, err := r.Submit(engine.NewMoveCommand(engine.NewUnitID(3, 0), newPos), "dummy-token")
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("Submit() error = %v, want %v", err, ErrInvalidConfig)
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		_, err := newFixture().r.Submit(engine.NewMoveCommand(unitID, newPos), "dummy-token")
		if !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("Submit() error = %v, want %v", err, ErrMatchNotFound)
		}
	})
}

func TestRoom_SubmitTeamFromUnitID(t *testing.T) {
	tests := []struct {
		name      string
		team      int
		tokenTeam int
		wantErr   error
	}{
		{"team-1 unit with team 1's token", 1, 1, nil},
		{"team-2 unit with team 2's token", 2, 2, ErrInvalidTurnCmd},
		{"team-2 unit with team 1's token", 2, 1, ErrInvalidToken},
		{"team-1 unit with team 2's token", 1, 2, ErrInvalidToken},
		{"team nibble 0", 0, 1, ErrInvalidConfig},
		{"team nibble 3", 3, 1, ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, tokens := startedRoom(t)
			cmd := engine.NewMoveCommand(engine.NewUnitID(tt.team, 0), engine.Coordinate{X: 4, Y: 7})
			_, err := r.Submit(cmd, tokens[tt.tokenTeam-1])
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Submit() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoom_Reset(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, tokens := startedRoom(t, unitAlreadyMoved(16))
		if !mustState(t, r).Units[16].HasMoved {
			t.Fatal("arranged unit has not moved")
		}
		if err := r.Reset(tokens[0]); err != nil {
			t.Fatalf("Reset() error = %v", err)
		}
		if mustState(t, r).Units[16].HasMoved {
			t.Errorf("Expected Unit %#X HasMoved reset to false, got true", 16)
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		if err := newFixture().r.Reset("dummy-token"); !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("Reset() error = %v, want %v", err, ErrMatchNotFound)
		}
	})
}

func TestRoom_Resolve(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, tokens := startedRoom(t, bombAboutToExplode(16))
		planGameEvents, resolveTurnGameEvents, err := r.Resolve(tokens[0])
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if got, want := planGameEvents, 1; len(got) != want {
			t.Errorf("Expected %d planGameEvents returned, got %#v", want, got)
		}
		if !hasBombPlaced(planGameEvents, 16) {
			t.Errorf("Expected the placed bomb in planGameEvents, got %#v", planGameEvents)
		}
		if got, want := resolveTurnGameEvents, 5; len(got) != want {
			t.Errorf("Expected %d resolveTurnGameEvents returned, got %#v", want, got)
		}
		if hasBombPlaced(resolveTurnGameEvents, 16) {
			t.Errorf("Expected no planning event in resolveTurnGameEvents, got %#v", resolveTurnGameEvents)
		}
		if got, want := mustState(t, r).Units[16].HP, 0; got != want {
			t.Errorf("Expected Unit %#X HP %v, got %v", 16, want, got)
		}
		if !slices.ContainsFunc(resolveTurnGameEvents, func(evt engine.GameEvent) bool {
			ended, ok := evt.(engine.MatchEnded)
			return ok && ended.WinnerTeamID == 2
		}) {
			t.Errorf("Expected a matchEnded event with winner 2, got %+v", resolveTurnGameEvents)
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		planGameEvents, resolveTurnGameEvents, err := newFixture().r.Resolve("dummy-token")
		if !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("Resolve() error = %v, want %v", err, ErrMatchNotFound)
		}
		if len(planGameEvents) != 0 || len(resolveTurnGameEvents) != 0 {
			t.Errorf("Expected gameEvents to be empty, got %#v and %#v", planGameEvents, resolveTurnGameEvents)
		}
	})
}

func TestRoom_Surrender(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, tokens := startedRoom(t)
		gameEvents, err := r.Surrender(1, tokens[0])
		if err != nil {
			t.Fatalf("Surrender() error = %v", err)
		}
		if got, want := len(gameEvents), 1; got != want {
			t.Fatalf("Expected %d gameEvents returned, got %#v", want, gameEvents)
		}
		if got, want := gameEvents[0], (engine.MatchEnded{WinnerTeamID: 2}); got != want {
			t.Errorf("Expected gameEvent %v, got %v", want, got)
		}
	})

	t.Run("Invalid team", func(t *testing.T) {
		r, tokens := startedRoom(t)
		gameEvents, err := r.Surrender(3, "dummy-token")
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Surrender() error = %v, want %v", err, ErrInvalidConfig)
		}
		if len(gameEvents) != 0 {
			t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
		}
		if err := r.Delete(tokens[0]); !errors.Is(err, ErrMatchInProgress) {
			t.Errorf("Delete() error = %v, want %v", err, ErrMatchInProgress)
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		gameEvents, err := newFixture().r.Surrender(1, "dummy")
		if !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("Surrender() error = %v, want %v", err, ErrMatchNotFound)
		}
		if len(gameEvents) != 0 {
			t.Errorf("Expected gameEvents to be empty, got %#v", gameEvents)
		}
	})
}

func TestRoom_Config(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, _ := startedRoom(t)
		got, err := r.Config()
		if err != nil {
			t.Fatalf("Config() error = %v", err)
		}
		if want := validGameCfg(); !reflect.DeepEqual(got, want) {
			t.Errorf("Expected GameCfg %+v, got %+v", want, got)
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		if _, err := newFixture().r.Config(); !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("Config() error = %v, want %v", err, ErrMatchNotFound)
		}
	})
}

func TestRoom_AllowedTiles(t *testing.T) {
	unitID := engine.NewUnitID(1, 0)

	t.Run("Success", func(t *testing.T) {
		r, _ := startedRoom(t)
		allowed, err := r.AllowedTiles(unitID, engine.TurnCmdPlaceBomb)
		if err != nil {
			t.Fatalf("AllowedTiles() error = %v", err)
		}
		want := []engine.Coordinate{{X: 2, Y: 8}, {X: 5, Y: 8}, {X: 6, Y: 8}, {X: 4, Y: 7}, {X: 4, Y: 6}}
		byXThenY := func(a, b engine.Coordinate) int {
			if a.X != b.X {
				return a.X - b.X
			}
			return a.Y - b.Y
		}
		slices.SortFunc(want, byXThenY)
		slices.SortFunc(allowed, byXThenY)
		if !slices.Equal(want, allowed) {
			t.Errorf("Expected coordinates %#v, got %#v", want, allowed)
		}
	})

	t.Run("Unsupported command", func(t *testing.T) {
		r, _ := startedRoom(t)
		if _, err := r.AllowedTiles(unitID, "invalid"); !errors.Is(err, engine.ErrUnsupportedCommand) {
			t.Errorf("AllowedTiles() error = %v, want %v", err, engine.ErrUnsupportedCommand)
		}
	})

	t.Run("Match Not Found", func(t *testing.T) {
		if _, err := newFixture().r.AllowedTiles(unitID, engine.TurnCmdPlaceBomb); !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("AllowedTiles() error = %v, want %v", err, ErrMatchNotFound)
		}
	})
}
