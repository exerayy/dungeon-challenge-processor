package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/exerayy/dungeon-challenge-processor/internal/config"
	"github.com/exerayy/dungeon-challenge-processor/internal/dungeoncfg"
	"github.com/exerayy/dungeon-challenge-processor/internal/processor"
)

const (
	layout = "15:04:05"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v. Shutting down gracefully...\n", sig)
		cancel()
	}()

	cfg := config.MustLoad()

	dungeonCfg, err := dungeoncfg.ParseConfig(cfg.DungeonCfgPath)
	if err != nil {
		fmt.Printf("failed to parse dungeoncfg: %v\n", err)
		return
	}

	fmt.Printf("Floors: %d\n", dungeonCfg.Floors)
	fmt.Printf("Monsters: %d\n", dungeonCfg.Monsters)
	fmt.Printf("OpenAt: %s\n", dungeonCfg.OpenTime.Format(layout))
	fmt.Printf("CloseAt: %s\n\n", dungeonCfg.CloseTime.Format(layout))

	game := processor.NewGameState(dungeonCfg)

	err = game.ProcessEvents(ctx, cfg.EventsPath, func(log string) {
		fmt.Println(log)
	})
	if err != nil {
		fmt.Printf("failed to process events: %v\n", err)
		return
	}

	reports, err := game.GetPlayersReportsSorted(ctx)
	if err != nil {
		fmt.Printf("failed to get players reports: %v\n", err)
		return
	}
	fmt.Println("\nFinal report:")
	for _, report := range reports {
		fmt.Println(report)
	}
}
