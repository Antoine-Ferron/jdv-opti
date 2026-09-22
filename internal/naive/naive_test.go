package naive

import (
	"testing"
	"unsafe"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/firetest"
)

func TestConformite(t *testing.T) {
	firetest.Run(t, func(m fire.Map) fire.Engine { return New(m) })
}

// TestSimLayout documente l'empreinte mémoire (axe padding) : à citer dans le rapport.
func TestSimLayout(t *testing.T) {
	t.Logf("unsafe.Sizeof(Cell{}) = %d octets", unsafe.Sizeof(Cell{}))
	t.Logf("unsafe.Sizeof(Sim{})  = %d octets", unsafe.Sizeof(Sim{}))
}
