package counters

import (
	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/firetest"
	"gol-wildfire/internal/flat"
	"testing"
)

func TestConformite(t *testing.T) {
	firetest.Run(t, func(m fire.Map) fire.Engine { return New(m) })
}

func TestCounterMatchesGrid(t *testing.T) {
	for _, fires := range []int{1, 64} {
		cfg := fire.DefaultConfig(67, 45, 42)
		cfg.Fires, cfg.Wind = fires, 0.2
		compare(t, fire.Generate(cfg), 500)
	}
	// Petits tores : plusieurs chemins de propagation aboutissent à la même case.
	for _, size := range [][2]int{{1, 1}, {1, 7}, {7, 1}, {3, 5}} {
		m := fire.NewMap(size[0], size[1])
		for i := range m.Terrain {
			m.Terrain[i] = fire.Forest
		}
		m.Ignite(0, 0)
		m.Ignite(0, 0)
		m.SetWind(0, 0, fire.West)
		compare(t, m, 30)
	}
}

func compare(t *testing.T, m fire.Map, turns int) {
	t.Helper()
	a, b := New(m), flat.New(m)
	for turn := 0; turn <= turns; turn++ {
		count := 0
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				if a.Fire(x, y) != b.Fire(x, y) || a.Rest(x, y) != b.Rest(x, y) {
					t.Fatalf("tour %d, case (%d,%d) différente", turn, x, y)
				}
				if a.Fire(x, y) > 0 {
					count++
				}
			}
		}
		if a.Burning() != count || a.Burning() != b.Burning() {
			t.Fatalf("tour %d : compteur=%d, recomptage=%d", turn, a.Burning(), count)
		}
		if turn == 0 || turn == turns {
			if a.Fingerprint() != b.Fingerprint() {
				t.Fatalf("empreinte différente au tour %d", turn)
			}
		}
		if turn < turns {
			a.Step()
			b.Step()
		}
	}
}

func TestInitialCountAndExtinction(t *testing.T) {
	m := fire.NewMap(5, 5)
	m.Terrain[12] = fire.Plain
	m.Ignite(2, 2)
	m.Ignite(2, 2)
	m.Ignite(0, 0) // Eau.
	e := New(m)
	if e.Burning() != 1 {
		t.Fatalf("foyer dupliqué/eau : %d", e.Burning())
	}
	result := fire.Run(e, fire.Options{Turns: 50})
	if !result.Extinct || result.Turns != 1 || e.Burning() != 0 {
		t.Fatalf("extinction incorrecte : %+v", result)
	}
	compare(t, m, 20) // Continuer après extinction ne doit pas rendre le compteur négatif.
	compare(t, fire.NewMap(3, 5), 10)
}

func TestStepNoAllocations(t *testing.T) {
	e := New(fire.Generate(fire.DefaultConfig(67, 45, 42)))
	if n := testing.AllocsPerRun(20, e.Step); n != 0 {
		t.Fatalf("%g allocations/tour", n)
	}
}
