package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	release   = "NONE"
	buildDate = "NONE"
	gitHash   = "NONE"
)

func printCLIVersion() {
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Release   string
		BuildDate string
		GitHash   string
	}{
		Release:   release,
		BuildDate: buildDate,
		GitHash:   gitHash,
	}); err != nil {
		fmt.Printf("error while decoding version info: %v\n", err)
	}
}
