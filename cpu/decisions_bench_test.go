package cpu

import (
	"bomb-srpg/engine"
	"fmt"
	"testing"
)

const (
	benchBombRange     = 2
	benchBombCountdown = 1
)

type benchRoster struct {
	name  string
	slots []engine.TeamSlot
}

func benchRosters() []benchRoster {
	king := engine.TeamSlot{Archetype: "King", Role: engine.RoleKing}
	fighter := engine.TeamSlot{Archetype: "Fighter", Role: engine.RoleNormal}
	boss := engine.TeamSlot{Archetype: "Bandit", Role: engine.RoleBoss}
	return []benchRoster{
		{"boss1", []engine.TeamSlot{boss}},
		{"king2", []engine.TeamSlot{king, fighter}},
		{"king5", []engine.TeamSlot{king, fighter, fighter, fighter, fighter}},
		{"boss5", []engine.TeamSlot{boss, boss, boss, boss, boss}},
	}
}

func benchState(b *testing.B, preset string, roster benchRoster, bombs int) *engine.GameState {
	b.Helper()
	match, err := engine.InitGame(engine.GameCfg{
		StagePreset: preset,
		P1Slots:     []engine.TeamSlot{{Archetype: "King", Role: engine.RoleKing}, {Archetype: "Fighter", Role: engine.RoleNormal}},
		P2Slots:     roster.slots,
		MaxTurns:    100,
	})
	if err != nil {
		b.Fatalf("InitGame() error = %v", err)
	}
	gs := match.WorkingState
	gs.ActiveTeam = engine.Team2

	var empty []engine.Coordinate
	for y, row := range gs.Grid {
		for x, tile := range row {
			if tile.Type == engine.TerrainPlain && tile.OccupantType == engine.OccupantNone {
				empty = append(empty, engine.Coordinate{X: x, Y: y})
			}
		}
	}
	if len(empty) < bombs {
		b.Fatalf("%s has %d free tiles, want at least %d for the bombs", preset, len(empty), bombs)
	}
	for i := range bombs {
		pos := empty[i*len(empty)/bombs]
		id := engine.NewBombID(1, i+1, engine.SystemUnitID)
		gs.Bombs[id] = &engine.Bomb{ID: id, OwnerID: engine.SystemUnitID, Position: pos, Range: benchBombRange, Countdown: benchBombCountdown}
		gs.UpdateStageOccupant(pos, engine.OccupantBomb, int64(id))
	}
	return gs
}

func BenchmarkDecide(b *testing.B) {
	for _, preset := range []string{"Plain", "Standard", "Divided"} {
		for _, roster := range benchRosters() {
			for _, bombs := range []int{0, 4, 10} {
				b.Run(fmt.Sprintf("%s/%s/bombs%d", preset, roster.name, bombs), func(b *testing.B) {
					gs := benchState(b, preset, roster, bombs)
					b.ReportAllocs()
					for b.Loop() {
						if _, err := Decide(gs); err != nil {
							b.Fatalf("Decide() error = %v", err)
						}
					}
				})
			}
		}
	}
}
