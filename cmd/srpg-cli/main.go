package main

import (
	"bomb-srpg/cli"
	"bomb-srpg/engine"
	"log"
	"os"
)

const defaultMaxTurns = 30

func defaultSlots() []engine.TeamSlot {
	return []engine.TeamSlot{
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
			Archetype: "Fighter",
			Role:      engine.RoleNormal,
		},
		{
			Archetype: "Fighter",
			Role:      engine.RoleNormal,
		},
	}
}

func main() {
	gameCfg := engine.GameCfg{
		StagePreset:    "Plain",
		MaxTurns:       defaultMaxTurns,
		AllowResetTurn: true,
		P1Slots:        defaultSlots(),
		P2Slots:        defaultSlots(),
	}

	match, err := engine.InitGame(gameCfg)
	if err != nil {
		log.Fatalf("Game setup error: %v", err)
	}

	terminalView := cli.NewTerminalView(os.Stdout)
	controller := cli.NewMatchController(match, terminalView, os.Stdin)

	controller.StartInputLoop()
}
