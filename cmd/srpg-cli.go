package main

import (
	"bomb-srpg/cmd/cli"
	"bomb-srpg/engine"
	"log"
	"os"
)

func main() {
	gameCfg := engine.GameCfg{
		StagePreset: "Plain",
		P1Teams:     []string{"King", "Fighter"},
		P2Teams:     []string{"King", "Thief"},
	}

	match, err := engine.InitGame(gameCfg)
	if err != nil {
		log.Fatalf("Game setup error: %v", err)
	}

	terminalView := cli.NewTerminalView(os.Stdout)
	controller := cli.NewMatchController(match, terminalView)

	controller.StartInputLoop()
}
