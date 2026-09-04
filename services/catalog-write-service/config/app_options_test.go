package config

import "testing"

func TestAppOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options AppOptions
		wantErr bool
	}{
		{name: "valid", options: AppOptions{GRPCPort: "6002", HealthPort: "7002"}},
		{name: "missing grpc port", options: AppOptions{HealthPort: "7002"}, wantErr: true},
		{name: "missing health port", options: AppOptions{GRPCPort: "6002"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.options.Validate()
			if test.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
