package main

import (
	"errors"

	"github.com/spf13/viper"
)

type Config struct {
	Logger  LoggerConf
	address string `mapstructure:"ADDRESS"`
	port    string `mapstructure:"PORT"`
}
type LoggerConf struct {
	level string `mapstructure:"LOG_LEVEL"`
}

func NewCLIConfig() *Config {
	return &Config{}
}

func (config *Config) Init(path string) error {
	if path == "" {
		err := errors.New("path can't be empty")
		return err
	}
	viper.SetDefault("ADDRESS", "127.0.0.1")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetConfigName("config_cli")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}
	config.address = viper.GetString("ADDRESS")
	config.port = viper.GetString("PORT")
	config.Logger.level = viper.GetString("LOG_LEVEL")
	return nil
}

func (config *Config) GetAddress() string {
	return config.address
}

func (config *Config) GetPort() string {
	return config.port
}
