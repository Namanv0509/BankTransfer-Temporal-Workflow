package config

import (
	"github.com/spf13/viper"
	"fmt"
)

type Config struct {
	TemporalHost string `mapstructure:"temporal_host"`
	PostgresDSN  string `mapstructure:"postgres_dsn"`
	NATSHost     string `mapstructure:"nats_host"`
	HTTPPort     string `mapstructure:"http_port"`
}

func LoadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return cfg, nil
}
