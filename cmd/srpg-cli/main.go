package main

import (
	"bomb-srpg/cli"
	"bomb-srpg/engine"
	"log"
	"os"
)

func main() {
	gameCfg := engine.GameCfg{
		StagePreset:    "MAP01",
		MaxTurns:       30,
		AllowResetTurn: true,
		SuddenDeath:    true,
		P1Teams:        []string{"King", "Witch", "Bandit", "Fighter", "Fighter"},
		P2Teams:        []string{"King", "Witch", "Bandit", "Fighter", "Fighter"},
	}

	match, err := engine.InitGame(gameCfg)
	if err != nil {
		log.Fatalf("Game setup error: %v", err)
	}

	terminalView := cli.NewTerminalView(os.Stdout)
	controller := cli.NewMatchController(match, terminalView, os.Stdin)

	controller.StartInputLoop()
}
