package lua_files

import (
	"embed"
	"log/slog"
)

//go:embed lua/*.lua
var luaFolder embed.FS

func ReadLuaFile(filename string) string {
	data, err := luaFolder.ReadFile(filename)

	if err != nil {
		slog.Error("Error reading lua file", "filename", filename, "err", err)
		return ""
	}

	return string(data)
}
