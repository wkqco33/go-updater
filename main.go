package main

import (
	"fmt"
	"os"

	"github.com/wkqco33/go-updater/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
