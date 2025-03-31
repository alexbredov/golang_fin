package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	helpers "github.com/alexbredov/golang_fin/helpers"
	loggercli "github.com/alexbredov/golang_fin/internal/logger-cli"
)

var ErrAntiBFNotAvailable = errors.New("antibf is not available")

type inputData struct {
	scanner *bufio.Scanner
}

func (input *inputData) Init() {
	input.scanner = bufio.NewScanner(os.Stdin)
}

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "./configs/", "Path to config file")
}

func main() {
	flag.Parse()
	if flag.Arg(0) == "version" {
		printCLIVersion()
		return
	}
	config := NewCLIConfig()
	err := config.Init(configPath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	log, err := loggercli.New(config.Logger.level)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	err = pingAntiBF(config.GetAddress() + ":" + config.GetPort())
	if err != nil {
		log.Error(err.Error())
		panic(err)
	}
	inpData := inputData{}
	inpData.Init()
	comCon := NewCommandController()
	comCon.Init(config.GetAddress()+":"+config.GetPort(), log)
	log.Info("antibfAddress:" + config.GetAddress() + ":" + config.GetPort())
	log.Info("antibf-cli is up and running")
	fmt.Println("Welcome to Antibf-cli!")
	fmt.Println(`Use "help" command to see all available commands, use "exit" to exit. :-)`)
	for {
		select {
		case <-ctx.Done():
			log.Info("antibf-cli is shutting down")
			os.Exit(1) //nolint:gocritic
		default:
			inpData.scanner.Scan()
			rawCommand := inpData.scanner.Text()
			if rawCommand == "" {
				continue
			}
			if rawCommand == "exit" {
				fmt.Println("Shutting down by user request")
				log.Info("antibf-cli is shutting down")
				os.Exit(1)
			}
			output := comCon.processCommand(rawCommand)
			fmt.Println(output)
		}
	}
}

func pingAntiBF(address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url := helpers.StringBuild("http://", address, "/")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(respBody) != "pong" {
		return ErrAntiBFNotAvailable
	}
	return nil
}
