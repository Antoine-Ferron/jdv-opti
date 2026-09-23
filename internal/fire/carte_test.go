package fire

import (
	"math"
	"testing"
)

func cfgTest() Config {
	return Config{
		Width: 128, Height: 128, Seed: 42, Scale: 4,
		Lakes: 0.08, Rivers: 0.04, Forest: 0.45, Wind: 0.01, Fires: 3,
	}
}

func part(m Map, pred func(i int) bool, sur func(i int) bool) float64 {
	n, total := 0, 0
	for i := range m.Terrain {
		if !sur(i) {
			continue
		}
		total++
		if pred(i) {
			n++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total)
}

func proche(t *testing.T, got, want float64, quoi string) {
	t.Helper()
	if math.Abs(got-want) > 0.02 {
		t.Errorf("%s : %.3f, attendu %.3f ± 0.02", quoi, got, want)
	}
}

func TestGenerationProportions(t *testing.T) {
	cfg := cfgTest()
	m := Generate(cfg)
	tout := func(int) bool { return true }

	eau := part(m, func(i int) bool { return m.Terrain[i] == Water }, tout)
	proche(t, eau, cfg.Lakes+cfg.Rivers, "proportion d'eau")

	foret := part(m, func(i int) bool { return m.Terrain[i] == Forest },
		func(i int) bool { return m.Terrain[i] != Water })
	proche(t, foret, cfg.Forest, "proportion de forêt hors de l'eau")

	vent := part(m, func(i int) bool { return m.Wind[i] > 0 },
		func(i int) bool { return m.Terrain[i].Combustion() > 0 })
	proche(t, vent, cfg.Wind, "proportion de cases vent parmi les combustibles")
}

func TestGenerationVentSeulementSurCombustible(t *testing.T) {
	m := Generate(cfgTest())
	for i := range m.Wind {
		if m.Wind[i] > 0 && m.Terrain[i].Combustion() == 0 {
			t.Fatalf("case %d : du vent posé sur de l'eau", i)
		}
	}
}

func TestGenerationFoyersCombustiblesEtDistincts(t *testing.T) {
	cfg := cfgTest()
	m := Generate(cfg)
	if len(m.Fires) != cfg.Fires {
		t.Fatalf("%d foyers, attendu %d", len(m.Fires), cfg.Fires)
	}
	vus := map[int]bool{}
	for _, i := range m.Fires {
		if m.Terrain[i].Combustion() == 0 {
			t.Errorf("foyer %d sur une case incombustible", i)
		}
		if vus[i] {
			t.Errorf("foyer %d tiré deux fois", i)
		}
		vus[i] = true
	}
}

func TestGenerationMemeGraineMemeCarte(t *testing.T) {
	a, b := Generate(cfgTest()), Generate(cfgTest())
	for i := range a.Terrain {
		if a.Terrain[i] != b.Terrain[i] || a.Wind[i] != b.Wind[i] {
			t.Fatalf("case %d : deux cartes de même graine diffèrent", i)
		}
	}
	for k := range a.Fires {
		if a.Fires[k] != b.Fires[k] {
			t.Fatalf("foyer %d : deux tirages de même graine diffèrent", k)
		}
	}

	cfg := cfgTest()
	cfg.Seed = 43
	if c := Generate(cfg); c.Terrain[0] == a.Terrain[0] && c.Fires[0] == a.Fires[0] {
		t.Log("graines différentes, cartes identiques en (0,0) : coïncidence possible, à surveiller")
	}
}
