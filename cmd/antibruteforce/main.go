package main

import (
	"antibf/internal/app"
	"antibf/internal/logger"
	http_internal "antibf/internal/server/http"
	RedisStorage "antibf/internal/storage/redis"
	SQLstorage "antibf/internal/storage/sqldb"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

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
	log.Info("Server Address: " + config.GetAddress())
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
	server := http_internal.NewServer(log, antibf, &config)
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
		os.Exit(1)
	}
}
