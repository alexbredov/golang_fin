//go:build integration

package integrationtests

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/abredov/golang_fin/helpers"
	"github.com/abredov/golang_fin/internal/logger"
	storageData "github.com/abredov/golang_fin/internal/storage/storageData"
	"github.com/redis/go-redis/v9"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
)

var (
	configFilePath string
	pgSQL_DB       *sql.DB
	reddb          *redis.Client
	config         Config
	log            *logger.LogWrapper
)

type AuthorizationRequestAnswer struct {
	Message string
	OK      bool
}
type outputJSON struct {
	Text string
	Code int
}
type IPListResult struct {
	IPList  []storageData.StorageIPData
	Message outputJSON
}
type InputTag struct {
	Tag string
}

func init() {
	flag.StringVar(&configFilePath, "config", "./configs/docker/", "Path to config file")
}

func TestMain(m *testing.M) {
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	config = NewConfig()
	err := config.Init(configFilePath)
	if err != nil {
		fmt.Println(err)
	}
	log, err = logger.New(config.Logger.Level)
	if err != nil {
		fmt.Println(err)
	}
	for {
		select {
		case <-ctx.Done():
			log.Info("Integration tests failed")
			os.Exit(1)
		default:
			pgSQL_DB, err = InitAndConnectDB(ctx, log, &config)
			if err != nil {
				log.Error("PGSQL InitAndConnectDB err: " + err.Error())
				cancel()
			}
			reddb, err = InitAndConnectRedis(ctx, log, &config)
			log.Info("Integration tests are up and running")
			exitCode := m.Run()
			log.Info("Exit code:" + strconv.Itoa(exitCode))
			err = cleanDBandRedis(ctx)
			if err != nil {
				cancel()
			}
			err = closeDBandRedis(ctx)
			if err != nil {
				cancel()
			}
			log.Info("Integration tests complete")
			os.Exit(exitCode)
		}
	}
}

func InitAndConnectDB(ctx context.Context, logger storageData.Logger, config storageData.Config) (*sql.DB, error) {
	select {
	case <-ctx.Done():
		return nil, storageData.ErrStorageTimeout
	default:
		defer recover()
		var err error
		dsn := helpers.StringBuild(config.GetDBUser(), ":", config.GetDBPassword(), "@tcp(", config.GetDBAddress(), ":", config.GetDBPort(), ")/", config.GetDBName())
		pgSQL_DBint, err := sql.Open("pgx", dsn)
		if err != nil {
			logger.Error("SQL Open connection failed:" + err.Error())
			return nil, err
		}
		pgSQL_DBint.SetConnMaxLifetime(config.GetDBMaxConnectionLifetime())
		pgSQL_DBint.SetMaxOpenConns(config.GetDBMaxOpenConnections())
		pgSQL_DBint.SetMaxIdleConns(config.GetDBMaxIdleConnections())
		err = pgSQL_DBint.PingContext(ctx)
		if err != nil {
			logger.Error("SQL DB ping failed:" + err.Error())
			return nil, err
		}
		return pgSQL_DBint, nil
	}
}
func InitAndConnectRedis(ctx context.Context, logger storageData.Logger, config storageData.Config) (*redis.Client, error) {
	select {
	case <-ctx.Done():
		return nil, storageData.ErrStorageTimeout
	default:
		defer recover()
		var err error
		reddb = redis.NewClient(&redis.Options{
			Addr:     config.GetRedisAddress() + ":" + config.GetRedisPort(),
			Password: "",
			DB:       0,
		})
		_, err = reddb.Ping(ctx).Result()
		if err != nil {
			logger.Error("Redis Ping err:" + err.Error())
			return nil, err
		}
		reddb.FlushDB(ctx)
		return reddb, nil
	}
}
func cleanDBandRedis(ctx context.Context) error {
	reddb.FlushDB(ctx)
	script := "TRUNCATE TABLE OTUSAntibf.whitelist"
	_, err := pgSQL_DB.ExecContext(ctx, script)
	if err != nil {
		return err
	}
	script = "TRUNCATE TABLE OTUSAntibf.blacklist"
	_, err = pgSQL_DB.ExecContext(ctx, script)
	if err != nil {
		return err
	}
}

func closeDBandRedis(ctx context.Context) error {
	err := reddb.Close()
	if err != nil {
		return err
	}
	err = pgSQL_DB.Close()
	return err
}
