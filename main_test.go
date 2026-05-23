package main

import (
	"fmt"
	"runtime"
	"testing"
)

func TestVersionDefault(t *testing.T) {
	if Version == "" {
		t.Fatal("expected non-empty default version")
	}
}

func TestBuildVersionOutputAddsVPrefixAndMetadata(t *testing.T) {
	got := buildVersionOutput("datacur8", "1.2.3")
	want := fmt.Sprintf("datacur8 version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestBuildVersionOutputPreservesExistingVPrefix(t *testing.T) {
	got := buildVersionOutput("datacur8", "v1.2.3")
	want := fmt.Sprintf("datacur8 version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestBuildVersionOutputLeavesDevVersionUntouched(t *testing.T) {
	got := buildVersionOutput("datacur8", "dev")
	want := fmt.Sprintf("datacur8 version dev (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}
