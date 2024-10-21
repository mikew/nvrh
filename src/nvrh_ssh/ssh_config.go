package nvrh_ssh

import (
	"log/slog"
	"os"
	"strings"
)

func CleanupSshConfigValue(value string) string {
	replaced := strings.Trim(value, "\"")

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("Error getting user home dir", "err", err)
		return replaced
	}

	replaced = strings.ReplaceAll(replaced, "$HOME", userHomeDir)
	if strings.HasPrefix(replaced, "~/") {
		replaced = strings.Replace(replaced, "~", userHomeDir, 1)
	}

	return replaced
}
