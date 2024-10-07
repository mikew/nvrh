package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"nvrh/src/client"
)

//go:embed manifest.json
var manifestData []byte

type Manifest struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	ShortDescription string `json:"shortDescription"`
}

func main() {
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		log.Fatalf("Error reading manifest: %v", err)
	}

	app := &cli.App{
		Name: manifest.Name,
		// These fields are named kind of strange. The `Usage` field is paired with
		// the `Name` when running `--help`.
		Usage:   manifest.ShortDescription,
		Version: manifest.Version,

		Commands: []*cli.Command{
			&client.CliClientCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
