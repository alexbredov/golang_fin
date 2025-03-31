package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexbredov/golang_fin/internal/app"
	"github.com/alexbredov/golang_fin/internal/logger"
	httpinternal "github.com/alexbredov/golang_fin/internal/server/http"
	RedisStorage "github.com/alexbredov/golang_fin/internal/storage/redis"
	SQLstorage "github.com/alexbredov/golang_fin/internal/storage/sqldb"
	_ "github.com/jackc/pgx/stdlib" // db driver
)

//nolint:gofmt,gofumt,gci,gosec,nolintlint
var configFilePath string

func init() {
	flag.StringVar(&configFilePath, "config", "./configs/", "Path to config")
}

func main() {
	flag.Parse()
	if flag.Arg(0) == "version" {
		printVersion()
		return
	}
	config := NewConfig()
	err := config.Init(configFilePath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	log, err := logger.New(config.Logger.Level)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	log.Info("Config path: " + configFilePath)
	log.Info("Server Address: " + config.GetAddress())
	log.Info("SQL Address:" + config.GetDBAddress() + ":" + config.GetDBPort())
	log.Info("Redis Address:" + config.GetRedisAddress() + ":" + config.GetRedisPort())
	files, err := os.ReadDir("/etc/antibf")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, file := range files {
		log.Info(file.Name())
	}
	var storage app.Storage
	ctxStorage, cancelStorage := context.WithTimeout(context.Background(), config.GetDBTimeout())
	storage = SQLstorage.New()
	err = storage.Init(ctxStorage, log, &config)
	if err != nil {
		cancelStorage()
		log.Fatal("SQL storage Init fatal failure:" + err.Error())
	}
	redis := RedisStorage.New()
	err = redis.Init(ctxStorage, log, &config)
	if err != nil {
		cancelStorage()
		log.Fatal("RedisDB Init fatal failure:" + err.Error())
	}
	antibf := app.New(log, storage, redis, &config)
	server := httpinternal.NewServer(log, antibf, &config)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), config.GetServerShutdownTimeout())
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatal("Failed to stop http server:" + err.Error())
		}
	}()
	if err := server.Start(ctx); err != nil {
		log.Error("Failed to start http server:" + err.Error())
		cancel()
		os.Exit(1) //nolint:gocritic
	}
}
