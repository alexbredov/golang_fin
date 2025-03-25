package main

import (
	"antibf/helpers"
	"antibf/internal/storage/storageData"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const correctAnswerText string = "Everything is OK"

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
type CommandController struct {
	address string
	logger  *loggercli.LogWrap
}

var (
	ErrUnsupported = errors.New("unsupported command")
	ErrBadArgCount = errors.New("bad argument count")
	ErrBadArgument = errors.New("bad argument")
)

func NewCommandController() *CommandController {
	return &CommandController{}
}
func (comcont *CommandController) Init(address string, logger *loggercli.LogWrap) {
	comcont.address = address
	comcont.logger = logger
}
func (comcont *CommandController) processCommand(rawCommand string) string {
	comcont.logger.Info("Command: " + rawCommand)
	commandData := strings.Split(rawCommand, " ")
	for i := range commandData {
		commandData[i] = strings.ToLower(strings.TrimSpace(commandData[i]))
	}
	switch commandData[0] {
	case "help":
		return comcont.help()
	case "whitelistadd", "wladd":
		return comcont.addToList(commandData, whitelist)
	case "whitelistremove", "wlrm":
		return comcont.removeFromList(commandData, whitelist)
	case "whitelistisin", "wlisin":
		return comcont.isInList(commandData, whitelist)
	case "whitelistallin", "wlallin":
		return comcont.allInList(commandData, whitelist)
	case "blacklistadd", "bladd":
		return comcont.addToList(commandData, blacklist)
	case "blacklistremove", "blrm":
		return comcont.removeFromList(commandData, blacklist)
	case "blacklistisin", "blisin":
		return comcont.isInList(commandData, blacklist)
	case "blacklistallin", "blallin":
		return comcont.allInList(commandData, blacklist)
	case "clearbucketforlogin" "logincl":
		return comcont.clearBucketByTag(commandData, "login")
	case "clearbucketforip", "ipcl":
		return comcont.clearBucketByTag(commandData, "ip")
	case "request":
		return comcont.request(commandData)
	default:
	}
	msg := "Error: " + ErrUnsupported.Error()
	comcont.logger.Info(msg)
	return msg
}
func (comcont *CommandController) help() string {
	return `
help - show this message
long: WhitelistAdd [subnet], short: wladd [subnet] - add subnet to whitelist
long: Remove`
}
func (comcont *CommandController) addToList(arg []string, listname string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if len(arg) != 2 {
		errStr := "Error: " + ErrBadArgCount.Error()
		comcont.logger.Error(errStr)
		return errStr
	}
	subArgs := strings.Split(arg[1], "/")
	if len(subArgs) != 2 {
		errStr := "Error: " + ErrBadArgCount.Error()
		comcont.logger.Error(errStr)
		return errStr
	}
	url := helpers.StringBuild("http://", comcont.address, "/", listname, "/")
	jsonStr := []byte(`{"IP":"` + subArgs[0] + `", "Mask":` + subArgs[1] + `}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		errStr := "Error: " + err.Error()
		comcont.logger.Error(errStr)
		return errStr
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		errStr := "Error: " + err.Error()
		comcont.logger.Error(errStr)
		return errStr
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		errStr := "Error: " + err.Error()
		comcont.logger.Error(errStr)
		return errStr
	}
	answer := outputJSON{}
	err = json.Unmarshal(respBody, &answer)
	if err != nil {
		errStr := "Error: " + err.Error()
		comcont.logger.Error(errStr)
		return errStr
	}
	if answer.Text != correctAnswerText {
		errStr := "Error: " + answer.Text
		comcont.logger.Error(errStr)
		return errStr
	}
	msg := "Subnet successfully added to " + listname + "."
	comcont.logger.Info(msg)
	return msg
}
