package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerAddr     string        `mapstructure:"SERVER_ADDR"`
	DatabaseURL    string        `mapstructure:"DATABASE_URL"`
	JWTSecret      string        `mapstructure:"JWT_SECRET"`
	AccessTokenTTL time.Duration `mapstructure:"ACCESS_TOKEN_TTL"`
	RefreshTokenTTL time.Duration `mapstructure:"REFRESH_TOKEN_TTL"`
	InternalAPIKey  string        `mapstructure:"INTERNAL_API_KEY"`
	BcryptCost      int           `mapstructure:"BCRYPT_COST"`
}

func Load() *Config {
	viper.SetDefault("SERVER_ADDR", ":3002")
	viper.SetDefault("ACCESS_TOKEN_TTL", "15m")
	viper.SetDefault("REFRESH_TOKEN_TTL", "168h") // 7 days
	viper.SetDefault("BCRYPT_COST", 12)

	viper.AutomaticEnv()

	cfg := &Config{
		ServerAddr:      viper.GetString("SERVER_ADDR"),
		DatabaseURL:     viper.GetString("DATABASE_URL"),
		JWTSecret:       viper.GetString("JWT_SECRET"),
		AccessTokenTTL:  viper.GetDuration("ACCESS_TOKEN_TTL"),
		RefreshTokenTTL: viper.GetDuration("REFRESH_TOKEN_TTL"),
		InternalAPIKey:  viper.GetString("INTERNAL_API_KEY"),
		BcryptCost:      viper.GetInt("BCRYPT_COST"),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	return cfg
}
