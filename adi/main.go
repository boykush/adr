package main

import (
	"os"

	"github.com/boykush/adr/adi/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
