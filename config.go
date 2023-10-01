package main

import (
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog/log"
)

type RootConfig struct {
	Routers      map[string]RouterConfig        `koanf:"routers"`
	StaticRoutes map[string][]StaticRouteConfig `koanf:"static_routes"`
}

type RouterConfig struct {
	Host                string                  `koanf:"host"`
	Port                int                     `koanf:"port"`
	User                string                  `koanf:"user"`
	Password            string                  `koanf:"password"`
	Interfaces          []RouterInterfaceConfig `koanf:"interfaces"`
	MaxParallelCommands int                     `koanf:"parallel_commands"`
}

type RouterInterfaceConfig struct {
	Name    string                        `koanf:"name"`
	Cleanup bool                          `koanf:"cleanup"`
	Routes  []RouterInterfaceRoutesConfig `koanf:"routes"`
}

type RouterInterfaceRoutesConfig struct {
	Name    string `koanf:"name"`
	Auto    bool   `koanf:"auto"`
	Gateway string `koanf:"gateway"`
}

type StaticRouteConfig struct {
	Target   string   `koanf:"target"`
	Resolver []string `koanf:"resolver"`
}

func loadConfig(args map[string]interface{}) RootConfig {
	path := args["--config"].(string)
	k := koanf.New(".")

	if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}

	var out RootConfig

	if err := k.Unmarshal("", &out); err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}

	return out
}
