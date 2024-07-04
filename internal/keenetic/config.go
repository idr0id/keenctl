package keenetic

import "time"

// ConnConfig holds the configuration settings for an SSH connection.
type ConnConfig struct {
	Host                string        `koanf:"host"`
	Port                int           `koanf:"port"`
	User                string        `koanf:"user"`
	Password            string        `koanf:"password"`
	MaxParallelCommands uint          `koanf:"max_parallel_commands"`
	Timeout             time.Duration `koanf:"timeout"`
	DryRun              bool          `koanf:"dry_run"`
}
