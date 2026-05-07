package server

import (
	"log/slog"
	"testing"
)

func TestServer_RegistersWorkspaceMemoryTools(t *testing.T) {
	t.Parallel()

	server := New(nil, nil, slog.Default(), "test")
	if got, want := server.RegisteredToolCount(), 17; got != want {
		t.Fatalf("RegisteredToolCount() = %d, want %d", got, want)
	}
}
