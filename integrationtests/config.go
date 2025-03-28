package integrationtests

import (
	"errors"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	Logger                  LoggerConf    `mapstructure:"Logger"`
	ServerShutdownTimeout   time.Duration `mapstructure:"server_shutdown_timeout"`
	dbMaxConnectionLifetime time.Duration `mapstructure:"db_max_conn_lifetime"`
	dbTimeout               time.Duration `mapstructure:"db_timeout"`
	limitTimeCheck          time.Duration `mapstructure:"limit_timecheck"`
	address                 string        `mapstructure:"address"`
	port                    string        `mapstructure:"port"`
	redisAddress            string        `mapstructure:"redis_address"`
	redisPort               string        `mapstructure:"redis_port"`
	dbAddress               string        `mapstructure:"db_address"`
	dbPort                  string        `mapstructure:"db_port"`
	dbName                  string        `mapstructure:"POSTGRES_DB"`
	dbUser                  string        `mapstructure:"POSTGRES_USER"`
	dbPassword              string        `mapstructure:"POSTGRES_PASSWORD"`
	dbMaxOpenConnections    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	dbMaxIdleConnections    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	limitLogin              int           `mapstructure:"limit_login"`
	limitPassword           int           `mapstructure:"limit_password"`
	limitIP                 int           `mapstructure:"limit_ip"`
}
type LoggerConf struct {
	Level string `mapstructure:"log_level"`
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
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	config.address = viper.GetString("ADDRESS")
	config.port = viper.GetString("PORT")
	config.ServerShutdownTimeout = viper.GetDuration("server_shutdown_timeout")
	config.dbName = viper.GetString("POSTGRES_DB")
	config.dbUser = viper.GetString("POSTGRES_USER")
	config.dbPassword = viper.GetString("POSTGRES_PASSWORD")
	config.dbMaxOpenConnections = viper.GetInt("DB_MAX_OPEN_CONNS")
	config.dbMaxIdleConnections = viper.GetInt("DB_MAX_IDLE_CONNS")
	config.dbMaxConnectionLifetime = viper.GetDuration("db_max_conn_lifetime")
	config.dbTimeout = viper.GetDuration("db_timeout")
	config.Logger.Level = viper.GetString("LOG_LEVEL")
	config.dbAddress = viper.GetString("DB_ADDRESS")
	config.dbPort = viper.GetString("DB_PORT")
	config.redisAddress = viper.GetString("REDIS_ADDRESS")
	config.redisPort = viper.GetString("REDIS_PORT")
	config.limitTimeCheck = viper.GetDuration("limit_timecheck")
	config.limitLogin = viper.GetInt("limit_login")
	config.limitPassword = viper.GetInt("limit_password")
	config.limitIP = viper.GetInt("limit_ip")
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
