package app

import (
	"testing"

	"go.uber.org/fx"
)

func TestModuleDependencyGraph(t *testing.T) {
	if err := fx.ValidateApp(Module()); err != nil {
		t.Fatalf("validate Fx module: %v", err)
	}
}
