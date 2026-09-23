package fire

import (
	"math"
	"math/rand"
	"sort"
)

// Config décrit la carte à engendrer (§7).
type Config struct {
	Width, Height int
	Seed          int64
	Lakes         float64 // proportion de lacs et étangs
	Rivers        float64 // proportion de rivières
	Forest        float64 // proportion de forêt hors de l'eau
	Wind          float64 // proportion de cases vent parmi les combustibles
	Scale         int     // passes de lissage : plus haut = massifs plus vastes
	Fires         int     // foyers de départ
}

// DefaultConfig renvoie les réglages par défaut pour une carte w×h (§8).
func DefaultConfig(w, h int, seed int64) Config {
	return Config{
		Width: w, Height: h, Seed: seed, Scale: 4,
		Lakes: 0.08, Rivers: 0.04, Forest: 0.45, Wind: 0.01, Fires: 1,
	}
}

// Generate engendre une carte déterministe : même graine, même carte et mêmes
// foyers. Deux champs de bruit lissés donnent le relief et la végétation ; les
// seuils sont des quantiles, ce qui rend les proportions demandées exactes.
func Generate(cfg Config) Map {
	m := NewMap(cfg.Width, cfg.Height)
	r := rand.New(rand.NewSource(cfg.Seed))
	alt := champLisse(cfg.Width, cfg.Height, cfg.Scale, r)
	veg := champLisse(cfg.Width, cfg.Height, cfg.Scale, r)

	// Les lacs occupent le fond des cuvettes ; les rivières longent la ligne de
	// niveau médiane, d'où leur forme de ruban sinueux.
	fondDeCuvette := quantile(alt, cfg.Lakes)
	median := quantile(alt, 0.5)
	ecart := make([]float64, len(alt))
	for i, a := range alt {
		ecart[i] = math.Abs(a - median)
	}
	berge := quantile(ecart, cfg.Rivers)

	sec := make([]float64, 0, len(veg))
	for i := range m.Terrain {
		if alt[i] < fondDeCuvette || ecart[i] < berge {
			m.Terrain[i] = Water
			continue
		}
		m.Terrain[i] = Plain
		sec = append(sec, veg[i])
	}
	// Seuil de forêt calculé sur les seules cases sèches : la proportion
	// demandée porte sur la terre ferme, pas sur la carte entière.
	lisiere := quantile(sec, 1-cfg.Forest)

	combustibles := make([]int, 0, len(m.Terrain))
	for i := range m.Terrain {
		if m.Terrain[i] == Water {
			continue
		}
		if veg[i] >= lisiere {
			m.Terrain[i] = Forest
		}
		combustibles = append(combustibles, i)
	}

	// Le vent ne se pose que sur du combustible : ailleurs il serait inerte (§2).
	for _, i := range tirage(combustibles, int(cfg.Wind*float64(len(combustibles))+0.5), r) {
		m.Wind[i] = 1 + uint8(r.Intn(8))
	}
	m.Fires = tirage(combustibles, cfg.Fires, r)
	return m
}

// champLisse engendre un champ aléatoire puis le lisse par moyennes 3×3
// successives sur le tore : les taches grossissent avec le nombre de passes.
func champLisse(w, h, passes int, r *rand.Rand) []float64 {
	f := make([]float64, w*h)
	for i := range f {
		f[i] = r.Float64()
	}
	tmp := make([]float64, len(f))
	for p := 0; p < passes; p++ {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				s := 0.0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						s += f[Mod(y+dy, h)*w+Mod(x+dx, w)]
					}
				}
				tmp[y*w+x] = s / 9
			}
		}
		f, tmp = tmp, f
	}
	return f
}

// quantile renvoie le seuil en dessous duquel se trouve la proportion q des
// valeurs. Comparer avec < donne alors exactement cette proportion.
func quantile(f []float64, q float64) float64 {
	if len(f) == 0 {
		return math.Inf(1)
	}
	c := append([]float64(nil), f...)
	sort.Float64s(c)
	i := int(q * float64(len(c)))
	if i <= 0 {
		return math.Inf(-1)
	}
	if i >= len(c) {
		return math.Inf(1)
	}
	return c[i]
}

// tirage prélève n éléments distincts de libres, sans modifier celui-ci.
func tirage(libres []int, n int, r *rand.Rand) []int {
	if n <= 0 {
		return nil
	}
	reste := append([]int(nil), libres...)
	out := make([]int, 0, n)
	for len(out) < n && len(reste) > 0 {
		j := r.Intn(len(reste))
		out = append(out, reste[j])
		reste[j] = reste[len(reste)-1]
		reste = reste[:len(reste)-1]
	}
	return out
}
