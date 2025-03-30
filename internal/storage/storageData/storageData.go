package storagedata

import (
	"errors"
	"github.com/alexbredov/golang_fin/helpers"
	"go.uber.org/zap"
	"strconv"
	"time"
)

const (
	WhiteListName string = "whitelist"
	BlackListName string = "blacklist"
)

var (
	ErrNoRecord       = errors.New("no record found")
	ErrStorageTimeout = errors.New("storage timeout")
	ErrBadListType    = errors.New("bad list type")
)

type Config interface {
	Init(path string) error
	GetServerURL() string
	GetAddress() string
	GetPort() string
	GetServerShutdownTimeout() time.Duration
	GetDBName() string
	GetDBUser() string
	GetDBPassword() string
	GetDBAddress() string
	GetDBPort() string
	GetDBMaxIdleConnections() int
	GetDBMaxOpenConnections() int
	GetDBMaxConnectionLifetime() time.Duration
	GetDBTimeout() time.Duration
	GetRedisAddress() string
	GetRedisPort() string
	GetLimitLogin() int
	GetLimitPassword() int
	GetLimitIP() int
	GetLimitTimeCheck() time.Duration
}
type Logger interface {
	Info(msg string)
	Warning(msg string)
	Error(msg string)
	Fatal(msg string)
	GetZapLogger() *zap.SugaredLogger
}
type StorageIPData struct {
	IP   string
	Mask int
	ID   int
}

func (ip *StorageIPData) String() string {
	result := helpers.StringBuild("[ID: ", strconv.Itoa(ip.ID), ", IP: ", ip.IP, "]")
	return result
}

type RequestAuth struct {
	Login    string
	Password string
	IP       string
}

func (request *RequestAuth) String() string {
	result := helpers.StringBuild("[Login: ", request.Login, ", Password: ", request.Password, ", IP: ", request.IP, "]")
	return result
}
