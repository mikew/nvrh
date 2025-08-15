package bridge_files

import (
	"embed"
	"log/slog"
)

//go:embed lua/*.lua shell/*
var luaFolder embed.FS

func ReadFileWithoutError(filename string) string {
	data, err := luaFolder.ReadFile(filename)

	if err != nil {
		slog.Error("Error reading lua file", "filename", filename, "err", err)
		return ""
	}

	return string(data)
}
