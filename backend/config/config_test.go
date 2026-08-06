package config

import "testing"

func TestGinModeUsesFixedReleaseDefault(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")

	cfg := Load()
	if cfg.GinMode != "release" {
		t.Fatalf("GinMode = %q, want release", cfg.GinMode)
	}
}
