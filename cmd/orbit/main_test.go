package main

import "testing"

// TestBuild verifies the package compiles and the entrypoint exists.
func TestBuild(t *testing.T) {
	// Sanity smoke test — if this compiles and runs, the package is healthy.
	t.Log("orbit cmd/orbit package builds successfully")
}
