package flat

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

// Les deux buffers doivent appartenir au moteur, sans modifier les données d'entrée.
func TestOwnsBuffers(t *testing.T) {
	cells := life.RandomCells(9, 7, 42, 0.3)
	original := append([]bool(nil), cells...)
	sim := New(9, 7, cells)
	for i := 0; i < 3; i++ {
		sim.Step()
	}
	for i := range cells {
		if cells[i] != original[i] {
			t.Fatal("Step modifie les données d'entrée")
		}
	}
	sim = New(9, 7, cells)
	cells[0] = !cells[0]
	if sim.Alive(0, 0) != original[0] {
		t.Fatal("New conserve un alias sur les données d'entrée")
	}
}
