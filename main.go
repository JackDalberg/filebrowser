package main

import (
	"os"

	"github.com/JackDalberg/filebrowser/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
