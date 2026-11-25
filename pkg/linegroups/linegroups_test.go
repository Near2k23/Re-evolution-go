package linegroups

import (
	"errors"
	"testing"
)

func TestClean(t *testing.T) {
	if n, c, err := clean("  Ventas  ", ""); err != nil || n != "Ventas" || c != Colors[0] {
		t.Fatalf("clean = %q %q %v", n, c, err)
	}
	if _, _, err := clean("", "green"); !errors.Is(err, ErrInvalid) {
		t.Fatal("nombre vacío debe fallar")
	}
	if _, _, err := clean("X", "fucsia"); !errors.Is(err, ErrInvalid) {
		t.Fatal("color desconocido debe fallar")
	}
}
