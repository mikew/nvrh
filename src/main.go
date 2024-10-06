package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"nvrh/src/client"
)

func main() {
	app := &cli.App{
		Name:  "nvrh",
		Usage: "Helps work with a remote nvim instance",

		Commands: []*cli.Command{
			&client.CliClientCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
