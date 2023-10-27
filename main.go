package main

import (
	"log"

	"github.com/zvonler/espy/cli"
)

func main() {
	espyCmd := cli.NewCommand()
	if err := espyCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
