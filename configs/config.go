package configs

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	PORT          int    `mapstructure:"PORT"`
	ENV           string `mapstructure:"ENV"`
	REACT_APP_URL string `mapstructure:"REACT_APP_API_URL"`

	DB_DSN            string `mapstructure:"DB_DSN"`
	DB_AUTOMIGRATE    bool   `mapstructure:"DB_AUTOMIGRATE"`
	DB_MAX_OPEN_CONNS int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	DB_MAX_IDLE_CONNS int    `mapstructure:"DB_MAX_IDLE_CONNS"`
	DB_MAX_IDLE_TIME  string `mapstructure:"DB_MAX_IDLE_TIME"`

	ACCESS_TOKEN_SECRET  string `mapstructure:"ACCESS_TOKEN_SECRET"`
	ACCESS_TOKEN_TTL     string `mapsctructure:"ACCESS_TOKEN_TTL"`
	REFRESH_TOKEN_SECRET string `mapsctructure:"REFRESH_TOKEN_SECRET"`
	REFRESH_TOKEN_TTL    string `mapsctructure:"REFRESH_TOKEN_TTL"`

	SMTP_HOST     string `mapstructure:"SMTP_HOST"`
	SMTP_PORT     int    `mapstructure:"SMTP_PORT"`
	SMTP_USERNAME string `mapstructure:"SMTP_USERNAME"`
	SMTP_PASSWORD string `mapstructure:"SMTP_PASSWORD"`
	SMTP_FROM     string `mapstructure:"SMTP_FROM"`

	GOOGLE_CLIENT_ID string `mapstructure:"GOOGLE_CLIENT_ID"`
}

func LoadConfig(path string) (config *Config, err error) {
	if os.Getenv("ENV") == "dev" || os.Getenv("ENV") == "" {
		viper.AddConfigPath(path)
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		if err = viper.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	viper.SetDefault("PORT", 8080)
	viper.SetDefault("ENV", "dev")
	viper.SetDefault("REACT_APP_URL", "http://localhost:3000")

	viper.SetDefault("DB_DSN", "")
	viper.SetDefault("DB_AUTOMIGRATE", true)
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_TIME", "15m")

	viper.SetDefault("ACCESS_TOKEN_SECRET", "secret")
	viper.SetDefault("ACCESS_TOKEN_TTL", "15m")
	viper.SetDefault("REFRESH_TOKEN_SECRET", "secret")
	viper.SetDefault("REFRESH_TOKEN_TTL", "168h")

	viper.SetDefault("SMTP_HOST", "smtp.gmail.com")
	viper.SetDefault("SMTP_PORT", 587)
	viper.SetDefault("SMTP_USERNAME", "")
	viper.SetDefault("SMTP_PASSWORD", "")
	viper.SetDefault("SMTP_FROM", "")

	viper.SetDefault("GOOGLE_CLIENT_ID", "")

	viper.AutomaticEnv()

	if err = viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	fmt.Println("Config loaded", config)
	return config, nil
}
