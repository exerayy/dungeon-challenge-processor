package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DungeonCfgPath string `env:"DUNGEON_CFG_PATH" env-default:"config.json"`
	EventsPath     string `env:"EVENTS_PATH" env-default:"events"`
}

func MustLoad() Config {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("error reading environment variables: %v", err)
	}
	return cfg
}
