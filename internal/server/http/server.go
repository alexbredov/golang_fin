package httpinternal

import (
	"context"
	"errors"
	"net/http"
	"time"

	storageData "github.com/abredov/golang_fin/internal/storage/storageData"
	"go.uber.org/zap"
)

var ErrBadBucketTypeTag = errors.New("bad bucket type tag")

type Server struct {
	server *http.Server
	logger Logger
	app    Application
	Config Config
}
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
type Application interface {
	InitBucketStorageAndLimits(ctx context.Context, config storageData.Config) error
	CloseBucketStorage(ctx context.Context) error
	CheckRequest(ctx context.Context, request storageData.RequestAuth) (bool, string, error)
	RateLimitTicker(ctx context.Context)
	ClearBucketForLogin(ctx context.Context, login string) error
	ClearBucketForIP(ctx context.Context, IP string) error
	InitStorage(ctx context.Context, config storageData.Config) error
	CloseStorage(ctx context.Context) error
	IPAddToList(ctx context.Context, listname string, IPData storageData.StorageIPData) (int, error)
	IPRemoveFromList(ctx context.Context, listname string, IPData storageData.StorageIPData) error
	IPIsInList(ctx context.Context, listname string, IPData storageData.StorageIPData) (bool, error)
	IPGetAllFromList(ctx context.Context, listname string) ([]storageData.StorageIPData, error)
}

func NewServer(logger Logger, app Application, config Config) *Server {
	server := Server{}
	server.logger = logger
	server.app = app
	server.Config = config
	server.server = &http.Server{
		Addr:              config.GetServerURL(),
		Handler:           server.routes(),
		ReadHeaderTimeout: 2 * time.Second,
	}
	return &server
}

func (server *Server) Start(ctx context.Context) error {
	server.logger.Info("AntiBF is running")
	server.app.RateLimitTicker(ctx)
	err := server.server.ListenAndServe()
	if err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			server.logger.Error("Server failed to start: " + err.Error())
			return err
		}
	}
	<-ctx.Done()
	return err
}

func (server *Server) Shutdown(ctx context.Context) error {
	err := server.server.Shutdown(ctx)
	if err != nil {
		server.logger.Error("Server failed to shutdown: " + err.Error())
		return err
	}
	err = server.app.CloseStorage(ctx)
	if err != nil {
		server.logger.Error("Server failed to close storage: " + err.Error())
		return err
	}
	err = server.app.CloseBucketStorage(ctx)
	if err != nil {
		server.logger.Error("Server failed to close bucket storage: " + err.Error())
		return err
	}
	server.logger.Info("AntiBF is stopped")
	return nil
}
