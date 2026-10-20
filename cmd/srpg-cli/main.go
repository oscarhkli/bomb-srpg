package main

import (
	"bomb-srpg/cli"
	"bomb-srpg/engine"
	"log"
	"os"
)

const (
	defaultMaxTurns  = 30
	archetypeFighter = "Fighter"
)

func main() {
	gameCfg := engine.GameCfg{
		StagePreset:    "Plain",
		MaxTurns:       defaultMaxTurns,
		AllowResetTurn: true,
		P1Slots: []engine.TeamSlot{
			{
				Archetype: "King",
				Role:      engine.RoleKing,
			},
			{
				Archetype: "Witch",
				Role:      engine.RoleNormal,
			},
			{
				Archetype: "Bandit",
				Role:      engine.RoleNormal,
			},
			{
				Archetype: archetypeFighter,
				Role:      engine.RoleNormal,
			},
			{
				Archetype: archetypeFighter,
				Role:      engine.RoleNormal,
			},
		},
		P2Slots: []engine.TeamSlot{
			{
				Archetype: "King",
				Role:      engine.RoleKing,
			},
			{
				Archetype: "Witch",
				Role:      engine.RoleNormal,
			},
			{
				Archetype: "Bandit",
				Role:      engine.RoleNormal,
			},
			{
				Archetype: archetypeFighter,
				Role:      engine.RoleNormal,
			},
			{
				Archetype: archetypeFighter,
				Role:      engine.RoleNormal,
			},
		},
	}

	match, err := engine.InitGame(gameCfg)
	if err != nil {
		log.Fatalf("Game setup error: %v", err)
	}

	terminalView := cli.NewTerminalView(os.Stdout)
	controller := cli.NewMatchController(match, terminalView, os.Stdin)

	controller.StartInputLoop()
}
