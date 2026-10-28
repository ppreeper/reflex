package main

import (
	"os"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	t.Chdir(t.TempDir())

	if got := resolveConfigPath("other.conf", true, true); got != "other.conf" {
		t.Errorf("explicit config: got %q; want %q", got, "other.conf")
	}
	if got := resolveConfigPath("", true, false); got != "" {
		t.Errorf("command without config: got %q; want no config", got)
	}
	if got := resolveConfigPath("", false, true); got != "" {
		t.Errorf("per-command flags without config: got %q; want no config", got)
	}
	if got := resolveConfigPath("", false, false); got != "" {
		t.Errorf("no reflex.conf: got %q; want no config", got)
	}

	if err := os.WriteFile(defaultConfigFile, []byte("echo hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveConfigPath("", false, false); got != defaultConfigFile {
		t.Errorf("reflex.conf present: got %q; want %q", got, defaultConfigFile)
	}

	if err := os.Remove(defaultConfigFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(defaultConfigFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := resolveConfigPath("", false, false); got != "" {
		t.Errorf("reflex.conf is a directory: got %q; want no config", got)
	}
}
