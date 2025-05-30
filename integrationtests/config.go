package integration

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Logger                  LoggerConf    `mapstructure:"Logger"`
	ServerShutdownTimeout   time.Duration `mapstructure:"SERVER_SHUTDOWN_TIMEOUT"`
	dbMaxConnectionLifetime time.Duration `mapstructure:"DB_MAX_CONN_LIFETIME"`
	dbTimeout               time.Duration `mapstructure:"DB_TIMEOUT"`
	limitTimeCheck          time.Duration `mapstructure:"LIMIT_TIMECHECK"`
	address                 string        `mapstructure:"ADDRESS"`
	port                    string        `mapstructure:"PORT"`
	redisAddress            string        `mapstructure:"REDIS_ADDRESS"`
	redisPort               string        `mapstructure:"REDIS_PORT"`
	dbAddress               string        `mapstructure:"DB_ADDRESS"`
	dbPort                  string        `mapstructure:"DB_PORT"`
	dbName                  string        `mapstructure:"POSTGRES_DB"`
	dbUser                  string        `mapstructure:"POSTGRES_USER"`
	dbPassword              string        `mapstructure:"POSTGRES_PASSWORD"`
	dbMaxOpenConnections    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	dbMaxIdleConnections    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	limitLogin              int           `mapstructure:"LIMIT_LOGIN"`
	limitPassword           int           `mapstructure:"LIMIT_PASSWORD"`
	limitIP                 int           `mapstructure:"LIMIT_IP"`
}
type LoggerConf struct {
	Level string `mapstructure:"LOG_LEVEL"`
}

func NewConfig() Config {
	return Config{}
}

func (config *Config) Init(path string) error {
	if path == "" {
		err := errors.New("no path provided")
		return err
	}
	viper.SetDefault("ADDRESS", "127.0.0.1")
	viper.SetDefault("PORT", "4000")
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second)
	viper.SetDefault("POSTGRES_DB", "OTUSFin")
	viper.SetDefault("POSTGRES_USER", "postgres")
	viper.SetDefault("POSTGRES_PASSWORD", "SecurePass")
	viper.SetDefault("DB_MAX_CONN_LIFETIME", 3*time.Minute)
	viper.SetDefault("DB_MAX_OPEN_CONNS", 10)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DB_TIMEOUT", 5*time.Second)
	viper.SetDefault("DB_ADDRESS", "127.0.0.1")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("REDIS_ADDRESS", "127.0.0.1")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("LIMIT_TIMECHECK", time.Minute)
	viper.SetDefault("LIMIT_LOGIN", 10)
	viper.SetDefault("LIMIT_PASSWORD", 100)
	viper.SetDefault("LIMIT_IP", 1000)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("env")
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
	config.ServerShutdownTimeout = viper.GetDuration("SERVER_SHUTDOWN_TIMEOUT")
	config.dbName = viper.GetString("POSTGRES_DB")
	config.dbUser = viper.GetString("POSTGRES_USER")
	config.dbPassword = viper.GetString("POSTGRES_PASSWORD")
	config.dbMaxOpenConnections = viper.GetInt("DB_MAX_OPEN_CONNS")
	config.dbMaxIdleConnections = viper.GetInt("DB_MAX_IDLE_CONNS")
	config.dbMaxConnectionLifetime = viper.GetDuration("DB_MAX_CONN_LIFETIME")
	config.dbTimeout = viper.GetDuration("DB_TIMEOUT")
	config.Logger.Level = viper.GetString("LOG_LEVEL")
	config.dbAddress = viper.GetString("DB_ADDRESS")
	config.dbPort = viper.GetString("DB_PORT")
	config.redisAddress = viper.GetString("REDIS_ADDRESS")
	config.redisPort = viper.GetString("REDIS_PORT")
	config.limitTimeCheck = viper.GetDuration("LIMIT_TIMECHECK")
	config.limitLogin = viper.GetInt("LIMIT_LOGIN")
	config.limitPassword = viper.GetInt("LIMIT_PASSWORD")
	config.limitIP = viper.GetInt("LIMIT_IP")
	return nil
}

func (config *Config) GetServerURL() string {
	return config.address + ":" + config.port
}

func (config *Config) GetAddress() string {
	return config.address
}

func (config *Config) GetPort() string {
	return config.port
}

func (config *Config) GetServerShutdownTimeout() time.Duration {
	return config.ServerShutdownTimeout
}

func (config *Config) GetDBName() string {
	return config.dbName
}

func (config *Config) GetDBUser() string {
	return config.dbUser
}

func (config *Config) GetDBPassword() string {
	return config.dbPassword
}

func (config *Config) GetDBMaxOpenConnections() int {
	return config.dbMaxOpenConnections
}

func (config *Config) GetDBMaxIdleConnections() int {
	return config.dbMaxIdleConnections
}

func (config *Config) GetDBMaxConnectionLifetime() time.Duration {
	return config.dbMaxConnectionLifetime
}

func (config *Config) GetDBTimeout() time.Duration {
	return config.dbTimeout
}

func (config *Config) GetDBAddress() string {
	return config.dbAddress
}

func (config *Config) GetDBPort() string {
	return config.dbPort
}

func (config *Config) GetRedisAddress() string {
	return config.redisAddress
}

func (config *Config) GetRedisPort() string {
	return config.redisPort
}

func (config *Config) GetLimitLogin() int {
	return config.limitLogin
}

func (config *Config) GetLimitPassword() int {
	return config.limitPassword
}

func (config *Config) GetLimitIP() int {
	return config.limitIP
}

func (config *Config) GetLimitTimeCheck() time.Duration {
	return config.limitTimeCheck
}
