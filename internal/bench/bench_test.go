// Benchmarks communs à toutes les implémentations.
//
// Les noms de sous-benchmarks suivent la convention clé=valeur de benchstat,
// ce qui permet de produire directement le tableau comparatif :
//
//	benchstat -col /impl results/bench.txt
package bench

import (
	"fmt"
	"testing"

	_ "gol-wildfire/internal/engines"
	"gol-wildfire/internal/fire"
)

var sizes = []int{256, 1024, 2048}

// Deux régimes très différents, et une implémentation peut gagner sur l'un en
// perdant sur l'autre : c'est justement ce qu'on veut voir.
//
//	front       — un seul foyer sur une grande carte : presque rien ne brûle,
//	              le balayage intégral de la grille domine tout le reste.
//	embrasement — beaucoup de foyers et une mise en régime préalable : la carte
//	              est saturée, ce sont la localité mémoire et la bande passante
//	              qui décident.
var scenarios = []struct {
	nom     string
	foyers  int
	chauffe int
}{
	{"front", 1, 0},
	{"embrasement", 64, 50},
}

func carte(n, foyers int) fire.Map {
	cfg := fire.DefaultConfig(n, n, 42)
	cfg.Fires = foyers
	return fire.Generate(cfg)
}

func chauffe(e fire.Engine, tours int) fire.Engine {
	for i := 0; i < tours; i++ {
		e.Step()
	}
	return e
}

// BenchmarkStep mesure le coût d'un tour (le cœur du hot path).
//
// L'état dérive d'une itération à l'autre : l'incendie s'étend, puis atteint son
// régime. La mesure reste comparable entre implémentations puisqu'elles partent toutes
// de la même carte et de la même graine, mais elle n'est pas une moyenne sur un
// régime stationnaire : c'est BenchmarkRun qui mesure un scénario borné.
func BenchmarkStep(b *testing.B) {
	for _, impl := range fire.Names() {
		f, _ := fire.Get(impl)
		for _, s := range scenarios {
			for _, n := range sizes {
				b.Run(fmt.Sprintf("impl=%s/scenario=%s/size=%d", impl, s.nom, n), func(b *testing.B) {
					e := chauffe(f(carte(n, s.foyers)), s.chauffe)
					b.ReportAllocs()
					b.SetBytes(int64(n * n)) // débit exprimé en cases/s
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						e.Step()
					}
				})
			}
		}
	}
}

// BenchmarkFingerprint mesure le coût de l'empreinte.
func BenchmarkFingerprint(b *testing.B) {
	for _, impl := range fire.Names() {
		f, _ := fire.Get(impl)
		n := 1024
		b.Run(fmt.Sprintf("impl=%s/size=%d", impl, n), func(b *testing.B) {
			e := chauffe(f(carte(n, 64)), 50)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = e.Fingerprint()
			}
		})
	}
}

// BenchmarkRun mesure un scénario complet et borné : construction du moteur
// comprise, nombre de tours fixe, donc parfaitement reproductible.
func BenchmarkRun(b *testing.B) {
	for _, impl := range fire.Names() {
		f, _ := fire.Get(impl)
		n, tours := 512, 50
		for _, s := range scenarios {
			m := carte(n, s.foyers)
			b.Run(fmt.Sprintf("impl=%s/scenario=%s/size=%d", impl, s.nom, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					fire.Run(f(m), fire.Options{Turns: tours})
				}
			})
		}
	}
}
