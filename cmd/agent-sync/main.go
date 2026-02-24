package main

import (
	"os"

	"github.com/bianoble/agent-sync/cmd/agent-sync/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
