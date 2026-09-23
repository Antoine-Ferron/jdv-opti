package firetest

import (
	"testing"

	"gol-wildfire/internal/fire"
)

// reference applique REGLES.md de la façon la plus directe possible, sans se
// soucier ni de la performance ni de ce que fait l'implémentation testée.
// Elle ne doit JAMAIS être optimisée : c'est elle qui dit le vrai.
func reference(m fire.Map, tours int) (feu, repos []uint8) {
	n := m.Width * m.Height
	feu, repos = make([]uint8, n), make([]uint8, n)
	for _, i := range m.Fires {
		feu[i] = m.Terrain[i].Combustion()
	}

	for t := 0; t < tours; t++ {
		// §1 : la propagation est calculée sur l'état du tour précédent.
		enflammee := make([]bool, n)
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				if feu[y*m.Width+x] == 0 {
					continue
				}
				for _, cible := range cibles(m, x, y) {
					enflammee[cible] = true
				}
			}
		}
		// §4 : les trois branches, dans cet ordre.
		for i := 0; i < n; i++ {
			switch {
			case feu[i] > 0:
				feu[i]--
				if feu[i] == 0 {
					repos[i] = m.Terrain[i].Repos()
				}
			case repos[i] > 0:
				repos[i]--
			case enflammee[i] && m.Terrain[i].Combustion() > 0:
				feu[i] = m.Terrain[i].Combustion()
			}
		}
	}
	return feu, repos
}

// cibles énumère les cases qu'une case en feu enflamme (§5).
func cibles(m fire.Map, x, y int) []int {
	vent, vente := m.WindAt(x, y)
	var out []int
	for j := 0; j < 8; j++ {
		if vente {
			// Le secteur amont — l'opposé du vent et ses deux diagonales —
			// n'est pas enflammé.
			switch fire.Mod(j-int(vent), 8) {
			case 3, 4, 5:
				continue
			}
		}
		d := fire.Offsets[j]
		out = append(out, m.At(x+d[0], y+d[1]))
	}
	if vente {
		// Le saut : deux cases sous le vent, par-dessus ce qui s'y trouve.
		d := fire.Offsets[vent]
		out = append(out, m.At(x+2*d[0], y+2*d[1]))
	}
	return out
}

// runConformite compare l'implémentation à la référence sur des cartes variées.
func runConformite(t *testing.T, f fire.Factory) {
	cas := []struct {
		nom   string
		cfg   fire.Config
		tours int
	}{
		{"taille_impaire", carte(67, 45, 7, 6), 30},
		{"carte_plate", carte(130, 3, 11, 4), 20},
		{"grande_carte", carte(256, 200, 42, 12), 25},
		{"beaucoup_de_vent", venteuse(64, 64, 3, 8), 30},
		{"sans_eau", seche(48, 48, 5, 2), 20},
	}
	for _, c := range cas {
		t.Run("Conformite_"+c.nom, func(t *testing.T) {
			m := fire.Generate(c.cfg)
			e := f(m)
			for i := 0; i < c.tours; i++ {
				e.Step()
			}
			feu, repos := reference(m, c.tours)
			for y := 0; y < m.Height; y++ {
				for x := 0; x < m.Width; x++ {
					i := y*m.Width + x
					if e.Fire(x, y) != feu[i] || e.Rest(x, y) != repos[i] {
						t.Fatalf("après %d tours, case (%d,%d) : (feu=%d, repos=%d), la référence dit (feu=%d, repos=%d)",
							c.tours, x, y, e.Fire(x, y), e.Rest(x, y), feu[i], repos[i])
					}
				}
			}
		})
	}
}

func carte(w, h int, seed int64, foyers int) fire.Config {
	cfg := fire.DefaultConfig(w, h, seed)
	cfg.Fires = foyers
	return cfg
}

func venteuse(w, h int, seed int64, foyers int) fire.Config {
	cfg := carte(w, h, seed, foyers)
	cfg.Wind = 0.15 // bien au-delà du réglage de jeu : on veut exercer la règle
	return cfg
}

func seche(w, h int, seed int64, foyers int) fire.Config {
	cfg := carte(w, h, seed, foyers)
	cfg.Lakes, cfg.Rivers = 0, 0
	return cfg
}
