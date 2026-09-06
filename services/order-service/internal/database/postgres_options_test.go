package database

import (
	"testing"
	"time"
)

func TestPostgresOptionsValidateRequiresConnectionTimeout(t *testing.T) {
	options := PostgresOptions{
		Host: "localhost", Port: "5432", User: "user", Password: "password",
		Database: "orders", SSLMode: "disable", ConnectionTimeout: 5 * time.Second,
	}
	if err := options.Validate(); err != nil {
		t.Fatalf("valid options returned an error: %v", err)
	}

	options.ConnectionTimeout = 0
	if err := options.Validate(); err == nil {
		t.Fatal("expected missing connection timeout to fail validation")
	}
}
