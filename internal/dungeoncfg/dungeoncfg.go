package dungeoncfg

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const layout = "15:04:05"

type Config struct {
	Floors   int64  `json:"Floors"`
	Monsters int64  `json:"Monsters"`
	OpenAt   string `json:"OpenAt"`
	Duration int64  `json:"Duration"`
}

type ParsedCfg struct {
	Floors    int64
	Monsters  int64
	OpenTime  time.Time
	CloseTime time.Time
}

func ParseConfig(filename string) (ParsedCfg, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ParsedCfg{}, fmt.Errorf("failed to read dungeon cfg file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ParsedCfg{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if cfg.Floors <= 0 {
		return ParsedCfg{}, fmt.Errorf("floors cannot be <= 0: %d", cfg.Floors)
	}

	if cfg.Monsters < 0 {
		return ParsedCfg{}, fmt.Errorf("monsters cannot be < 0: %d", cfg.Monsters)
	}

	if cfg.Duration < 0 {
		return ParsedCfg{}, fmt.Errorf("duration cannot be < 0: %d", cfg.Duration)
	}

	openTime, err := time.Parse(layout, cfg.OpenAt)
	if err != nil {
		return ParsedCfg{}, fmt.Errorf("invalid time format OpenAt: %w", err)
	}

	timeDuration := time.Duration(cfg.Duration) * time.Hour

	closeTime := openTime.Add(timeDuration)

	return ParsedCfg{
		Floors:    cfg.Floors,
		Monsters:  cfg.Monsters,
		OpenTime:  openTime,
		CloseTime: closeTime,
	}, nil
}
