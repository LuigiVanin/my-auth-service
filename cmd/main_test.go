package main

import (
	"testing"

	"go.uber.org/fx"
)

// TestDependencyGraphIsWireable type checks every provider and every consumer
// without constructing anything, so a constructor signature change that breaks
// the container fails here instead of at boot.
func TestDependencyGraphIsWireable(t *testing.T) {
	if err := fx.ValidateApp(AppBootstrap()); err != nil {
		t.Fatalf("fx dependency graph is broken: %v", err)
	}
}
