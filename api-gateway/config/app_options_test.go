package config

import (
	"testing"
	"time"
)

func TestLoadAppOptionsParsesTimeouts(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CONFIG_PATH", ".")

	options, err := LoadAppOptions()
	if err != nil {
		t.Fatalf("load app options: %v", err)
	}
	if options.RequestTimeout != 10*time.Second {
		t.Fatalf("expected 10s request timeout, got %v", options.RequestTimeout)
	}
	if options.ManagementPort != "7070" {
		t.Fatalf("expected management port 7070, got %q", options.ManagementPort)
	}
	if options.IdleTimeout != 60*time.Second {
		t.Fatalf("expected 60s idle timeout, got %v", options.IdleTimeout)
	}
	if options.GRPCReadTimeout != 5*time.Second {
		t.Fatalf("expected 5s gRPC read timeout, got %v", options.GRPCReadTimeout)
	}
	if options.GRPCWriteTimeout != 8*time.Second {
		t.Fatalf("expected 8s gRPC write timeout, got %v", options.GRPCWriteTimeout)
	}
}

func TestAppOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*AppOptions)
		wantErr bool
	}{
		{name: "valid"},
		{name: "missing request timeout", change: func(options *AppOptions) { options.RequestTimeout = 0 }, wantErr: true},
		{name: "read call exceeds request timeout", change: func(options *AppOptions) { options.GRPCReadTimeout = options.RequestTimeout }, wantErr: true},
		{name: "write call exceeds request timeout", change: func(options *AppOptions) { options.GRPCWriteTimeout = options.RequestTimeout }, wantErr: true},
		{name: "write timeout does not allow response", change: func(options *AppOptions) { options.WriteTimeout = options.RequestTimeout }, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := validOptions()
			if test.change != nil {
				test.change(&options)
			}
			err := options.Validate()
			if test.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func validOptions() AppOptions {
	return AppOptions{
		AppPort:              "8080",
		ManagementPort:       "7070",
		CatalogReadGrpcAddr:  "catalog-read-service:6001",
		CatalogWriteGrpcAddr: "catalog-write-service:6002",
		OrderGrpcUrl:         "order-service:6005",
		RequestTimeout:       10 * time.Second,
		GRPCReadTimeout:      5 * time.Second,
		GRPCWriteTimeout:     8 * time.Second,
		ReadHeaderTimeout:    5 * time.Second,
		ReadTimeout:          15 * time.Second,
		WriteTimeout:         15 * time.Second,
		IdleTimeout:          60 * time.Second,
	}
}
