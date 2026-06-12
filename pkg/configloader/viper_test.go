package configloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBindKeyEnvironmentOverridesConfigFile(t *testing.T) {
	configPath := t.TempDir()
	configFile := `{
		"postgresOptions": {
			"host": "json-host",
			"password": ""
		}
	}`

	if err := os.WriteFile(
		filepath.Join(configPath, "config.development.json"),
		[]byte(configFile),
		0o600,
	); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("DB_PASSWORD", "environment-password")

	options, err := BindKey[struct {
		Host     string `mapstructure:"host"`
		Password string `mapstructure:"password"`
	}](
		"postgresOptions",
		map[string]string{
			"password": "DB_PASSWORD",
		},
	)
	if err != nil {
		t.Fatalf("bind config: %v", err)
	}

	if options.Host != "json-host" {
		t.Fatalf("expected JSON host, got %q", options.Host)
	}

	if options.Password != "environment-password" {
		t.Fatalf("expected environment password, got %q", options.Password)
	}
}
