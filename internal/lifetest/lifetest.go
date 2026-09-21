// Package lifetest fournit une suite de conformité commune : toute implémentation
// doit produire exactement les mêmes grilles qu'une référence indépendante.
// Une optimisation qui casse la correction n'est pas une optimisation.
package lifetest

import (
	"testing"

	"gol/internal/life"
)

// reference est une implémentation minimale, lente mais évidente, sur grille plate.
// Elle ne doit JAMAIS être optimisée.
func reference(w, h int, cells []bool, gens int) []bool {
	cur := append([]bool(nil), cells...)
	next := make([]bool, len(cur))
	for g := 0; g < gens; g++ {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				n := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if (dx != 0 || dy != 0) && cur[((y+dy+h)%h)*w+(x+dx+w)%w] {
							n++
						}
					}
				}
				a := cur[y*w+x]
				next[y*w+x] = n == 3 || (a && n == 2)
			}
		}
		cur, next = next, cur
	}
	return cur
}

func fromPattern(w, h int, pts [][2]int) []bool {
	c := make([]bool, w*h)
	for _, p := range pts {
		c[p[1]*w+p[0]] = true
	}
	return c
}

func assertGrid(t *testing.T, e life.Engine, want []bool, w, h int, label string) {
	t.Helper()
	pop := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if want[y*w+x] {
				pop++
			}
			if got := e.Alive(x, y); got != want[y*w+x] {
				t.Fatalf("%s : cellule (%d,%d) = %v, attendu %v", label, x, y, got, want[y*w+x])
			}
		}
	}
	if got := e.Population(); got != pop {
		t.Fatalf("%s : population = %d, attendu %d", label, got, pop)
	}
}

// Run exécute la suite de conformité sur une factory.
func Run(t *testing.T, f life.Factory) {
	cases := []struct {
		name string
		w, h int
		cell []bool
		gens int
	}{
		{"block_stable", 8, 8, fromPattern(8, 8, [][2]int{{3, 3}, {4, 3}, {3, 4}, {4, 4}}), 5},
		{"blinker", 7, 7, fromPattern(7, 7, [][2]int{{2, 3}, {3, 3}, {4, 3}}), 3},
		// Le planeur traverse les bords : vérifie l'enroulement torique.
		{"glider_wrap", 10, 10, fromPattern(10, 10, [][2]int{{1, 0}, {2, 1}, {0, 2}, {1, 2}, {2, 2}}), 40},
		// Dimensions non multiples de 64 : piège classique du bit-packing.
		{"random_odd_size", 67, 45, life.RandomCells(67, 45, 7, 0.35), 30},
		{"random_1x_row", 130, 3, life.RandomCells(130, 3, 11, 0.5), 10},
		{"random_large", 256, 200, life.RandomCells(256, 200, 42, 0.3), 25},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := f(c.w, c.h, c.cell)
			assertGrid(t, e, c.cell, c.w, c.h, "état initial")
			for g := 1; g <= c.gens; g++ {
				e.Step()
			}
			assertGrid(t, e, reference(c.w, c.h, c.cell, c.gens), c.w, c.h, "après simulation")
		})
	}

	t.Run("fingerprint", func(t *testing.T) {
		cells := life.RandomCells(64, 64, 3, 0.3)
		a, b := f(64, 64, cells), f(64, 64, cells)
		if a.Fingerprint() != b.Fingerprint() {
			t.Fatal("deux états identiques ont des empreintes différentes")
		}
		b.Step()
		if a.Fingerprint() == b.Fingerprint() {
			t.Fatal("deux états différents ont la même empreinte")
		}
	})

	t.Run("blinker_cycle_detected", func(t *testing.T) {
		e := f(7, 7, fromPattern(7, 7, [][2]int{{2, 3}, {3, 3}, {4, 3}}))
		res, err := life.Run(e, life.Options{Generations: 100, DetectCycles: true})
		if err != nil {
			t.Fatal(err)
		}
		if res.Period != 2 || res.Generations != 2 {
			t.Fatalf("cycle attendu de période 2 détecté à la génération 2, obtenu %+v", res)
		}
	})
}
