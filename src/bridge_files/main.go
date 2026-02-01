package bridge_files

import (
	"bytes"
	"embed"
	"log/slog"
	"text/template"
)

//go:embed lua/* shell/*
var luaFolder embed.FS

func ReadFileWithoutError(filename string) string {
	data, err := luaFolder.ReadFile(filename)

	if err != nil {
		slog.Error("Error reading lua file", "filename", filename, "err", err)
		return ""
	}

	return string(data)
}

func ReadFileWithTemplate(filename string, data any) string {
	fileContent := ReadFileWithoutError(filename)

	tmpl, err := template.New("file").Parse(fileContent)
	if err != nil {
		slog.Error("Error parsing template", "filename", filename, "err", err)
		return ""
	}

	var renderedContentBytes bytes.Buffer
	err = tmpl.Execute(&renderedContentBytes, data)
	if err != nil {
		slog.Error("Error executing template", "filename", filename, "err", err)
		return ""
	}

	return renderedContentBytes.String()
}
