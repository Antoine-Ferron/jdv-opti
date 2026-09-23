package flat

import (
	"gol-wildfire/internal/naive"
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

func TestBuffersAndFingerprint(t *testing.T) {
	cfg := fire.DefaultConfig(67, 45, 42)
	cfg.Fires, cfg.Wind = 8, 0.2
	m := fire.Generate(cfg)
	a, b := New(m), naive.New(m)
	for turn := 0; turn < 100; turn++ {
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				if a.Fire(x, y) != b.Fire(x, y) || a.Rest(x, y) != b.Rest(x, y) {
					t.Fatalf("tour %d, case (%d,%d) différente", turn, x, y)
				}
			}
		}
		if a.Burning() != b.Burning() || a.Fingerprint() != b.Fingerprint() {
			t.Fatalf("compteur ou empreinte différents au tour %d", turn)
		}
		a.Step()
		b.Step()
	}
}

func TestNoResidualIgnition(t *testing.T) {
	m := fire.NewMap(5, 5)
	m.Terrain[12], m.Terrain[13] = fire.Plain, fire.Plain
	m.Ignite(2, 2)
	e := New(m)
	for i := 0; i < 12; i++ {
		e.Step()
	}
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			if e.Fire(x, y) != 0 || e.Rest(x, y) != 0 {
				t.Fatal("état résiduel après extinction")
			}
		}
	}
}

func TestStepNoAllocations(t *testing.T) {
	e := New(fire.Generate(fire.DefaultConfig(67, 45, 42)))
	if n := testing.AllocsPerRun(20, e.Step); n != 0 {
		t.Fatalf("%g allocations/tour", n)
	}
}
