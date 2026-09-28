package bitpack

import (
	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/firetest"
	"gol-wildfire/internal/ghost"
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
	a, b := New(m), ghost.New(m)
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

// Toutes les directions, coins, petites dimensions et largeurs non alignées.
func TestWordBoundaries(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {1, 2}, {2, 1}, {2, 2}, {2, 7}, {7, 2}, {3, 3}, {5, 7}, {63, 9}, {64, 9}, {65, 9}, {127, 5}, {128, 5}, {129, 5}} {
		for d := 0; d < 8; d++ {
			m := fire.NewMap(size[0], size[1])
			for i := range m.Terrain {
				m.Terrain[i] = fire.Terrain(1 + i%2)
				m.Wind[i] = uint8(d + 1)
			}
			m.Ignite(0, 0)
			m.Ignite(size[0]-1, size[1]-1)
			compare(t, m, 80)
		}
	}
}

func TestPackedPadding(t *testing.T) {
	for _, width := range []int{1, 2, 63, 64, 65, 127, 128, 129} {
		cfg := fire.DefaultConfig(width, 7, 123)
		cfg.Wind = 0
		m := fire.Generate(cfg)
		compare(t, m, 100)
		e := New(m)
		for turn := 0; turn < 100; turn++ {
			for y := 0; y < e.Height(); y++ {
				i := (y+1)*e.words - 1
				if (e.f1[i]|e.f2[i]|e.r1[i]|e.r2[i]|e.ignite[i])&^e.lastMask != 0 {
					t.Fatalf("bits hors grille largeur %d tour %d", width, turn)
				}
			}
			e.Step()
		}
	}
}
