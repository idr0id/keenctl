package app

import (
	"fmt"
	"github.com/idr0id/keenctl/internal/keenetic"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	SSH        keenetic.ConnConfig `koanf:"ssh"`
	Interfaces []InterfaceConfig   `koanf:"interfaces"`
}

func ParseConfig(path string, dryRun bool) (Config, error) {
	var config Config

	k := koanf.New(".")
	if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
		return config, fmt.Errorf("error loading config: %w", err)
	}

	if err := k.Unmarshal("", &config); err != nil {
		return config, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if dryRun {
		config.SSH.DryRun = true
	}

	return config, nil
}

type InterfaceConfig struct {
	Name     string             `koanf:"name"`
	Cleanup  bool               `koanf:"cleanup"`
	Defaults routeOptionsConfig `koanf:"defaults"`
	Routes   []RouteConfig      `koanf:"routes"`
}

type RouteConfig struct {
	routeOptionsConfig
	Target   string `koanf:"target"`
	Resolver string `koanf:"resolver"`
}

type routeOptionsConfig struct {
	Auto    *bool     `koanf:"auto"`
	Gateway *string   `koanf:"gateway"`
	Filters *[]string `koanf:"filters"`
}

func (r routeOptionsConfig) GetFilters(defaults routeOptionsConfig) []string {
	if r.Filters != nil {
		return *r.Filters
	}
	return *defaults.Filters
}

func (r routeOptionsConfig) GetAuto(defaults routeOptionsConfig) bool {
	if r.Auto != nil {
		return *r.Auto
	}
	if defaults.Auto != nil {
		return *defaults.Auto
	}
	return false
}

func (r routeOptionsConfig) GetGateway(defaults routeOptionsConfig) string {
	if r.Gateway != nil {
		return *r.Gateway
	}
	if defaults.Gateway != nil {
		return *defaults.Gateway
	}
	return ""
}
