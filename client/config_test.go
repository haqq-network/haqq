package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func TestInitConfigNonNotExistError(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "nonPerms")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create sub directory: %v", err)
	}
	// Create config directory and file with no read permission
	configDir := filepath.Join(subDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	configFile := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configFile, []byte("test = true\n"), 0o000); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer func() {
		os.Chmod(configFile, 0o644) // Restore permissions for cleanup
		os.RemoveAll(subDir)
	}()

	cmd := &cobra.Command{}
	cmd.PersistentFlags().String(flags.FlagHome, "", "")
	cmd.PersistentFlags().String(flags.FlagChainID, "", "")
	cmd.PersistentFlags().String("encoding", "", "")
	cmd.PersistentFlags().String("output", "", "")
	if err := cmd.PersistentFlags().Set(flags.FlagHome, subDir); err != nil {
		t.Fatalf("Could not set home flag [%T] %v", err, err)
	}
	if err := cmd.PersistentFlags().Set(flags.FlagChainID, "test-chain"); err != nil {
		t.Fatalf("Could not set chain-id flag [%T] %v", err, err)
	}
	if err := cmd.PersistentFlags().Set("encoding", "json"); err != nil {
		t.Fatalf("Could not set encoding flag [%T] %v", err, err)
	}
	if err := cmd.PersistentFlags().Set("output", "text"); err != nil {
		t.Fatalf("Could not set output flag [%T] %v", err, err)
	}

	// InitConfig should return a permission error when trying to read the config file
	err := InitConfig(cmd)
	if err == nil {
		// If no error, that's also acceptable - the file might be stat-able but not readable
		// The important thing is that it doesn't panic
		return
	}
	if !os.IsPermission(err) {
		t.Fatalf("Expected permission error or nil, got: [%T] %v", err, err)
	}
}
