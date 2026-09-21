package naive

import (
	"testing"
	"unsafe"

	"gol/internal/life"
	"gol/internal/lifetest"
)

func TestConformance(t *testing.T) {
	lifetest.Run(t, func(w, h int, c []bool) life.Engine { return New(w, h, c) })
}

// TestSimLayout documente la taille de la struct (axe padding) : à citer dans le rapport.
func TestSimLayout(t *testing.T) {
	t.Logf("unsafe.Sizeof(Sim{}) = %d octets", unsafe.Sizeof(Sim{}))
}
