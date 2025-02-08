package utils

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration settings
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Scraper  ScraperConfig  `mapstructure:"scraper"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	DSN      string // This will be constructed from the other fields
}

// ScraperConfig holds scraper-specific configuration
type ScraperConfig struct {
	MaxPages int `mapstructure:"max_pages"`
	Timeout  int `mapstructure:"timeout"`
	Delay    int `mapstructure:"delay"`
}

// LoadConfig reads configuration from a config file
func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Build DSN after loading config
	config.Database.BuildDSN()

	return &config, nil
}

// BuildDSN constructs the database connection string
func (dc *DatabaseConfig) BuildDSN() {
	dc.DSN = fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dc.Host,
		dc.Port,
		dc.User,
		dc.Password,
		dc.DBName,
		dc.SSLMode,
	)
}
