package utils

import (
	"fmt"

	"github.com/spf13/viper"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	DSN      string
}

// ScraperConfig holds scraper configuration
type ScraperConfig struct {
	MaxPages int `mapstructure:"max_pages"`
	Timeout  int `mapstructure:"timeout"`
	Delay    int `mapstructure:"delay"`
	// ... any other scraper config fields
}

// Config holds all configuration
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Scraper  ScraperConfig  `mapstructure:"scraper"`
}

// BuildDSN builds the database connection string
func (c *Config) BuildDSN() {
	c.Database.DSN = fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Build the DSN string
	config.BuildDSN()

	return &config, nil
}
