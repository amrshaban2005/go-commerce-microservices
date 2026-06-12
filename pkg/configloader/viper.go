package configloader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func BindKey[T any](key string, envBindings map[string]string) (*T, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join(".", "config")
	}

	v := viper.New()
	v.SetConfigName("config." + env)
	v.SetConfigType("json")
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	section := v.Sub(key)
	if section == nil {
		return nil, fmt.Errorf("config key %s not found", key)
	}

	for field, environmentVariable := range envBindings {
		if err := section.BindEnv(field, environmentVariable); err != nil {
			return nil, fmt.Errorf("bind %s to %s: %w", field, environmentVariable, err)
		}
	}

	var options T
	if err := section.Unmarshal(&options); err != nil {
		return nil, fmt.Errorf("unmarshal config key %s: %w", key, err)
	}

	return &options, nil
}
