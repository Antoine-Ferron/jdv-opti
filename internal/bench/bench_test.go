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
type scenario struct {
	nom     string
	foyers  int
	chauffe int
}

// etabli est le seul régime où mesurer un Step isolé a un sens : la carte est
// saturée, l'état ne dérive plus d'une itération à l'autre.
var etabli = scenario{"embrasement", 64, 50}

var scenarios = []scenario{{"front", 1, 0}, etabli}

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

// BenchmarkStep mesure le coût d'un tour (le cœur du hot path), et uniquement
// en régime établi.
//
// Un Step isolé ne se mesure que sur un état stable : go test choisit lui-même
// le nombre d'itérations, or en scénario front l'incendie s'étend pendant la
// mesure, si bien que le coût par tour dépend de b.N. Mesuré ainsi, front
// donnait ±33 % de variance à 2048² — inexploitable pour comparer deux
// implémentations. Il est mesuré par BenchmarkRun, qui borne le travail par
// itération.
func BenchmarkStep(b *testing.B) {
	for _, impl := range fire.Names() {
		f, _ := fire.Get(impl)
		for _, n := range sizes {
			b.Run(fmt.Sprintf("impl=%s/scenario=%s/size=%d", impl, etabli.nom, n), func(b *testing.B) {
				e := chauffe(f(carte(n, etabli.foyers)), etabli.chauffe)
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
// comprise, nombre de tours fixe, donc parfaitement reproductible — c'est ici
// que le scénario front est mesuré.
func BenchmarkRun(b *testing.B) {
	const tours = 50
	for _, impl := range fire.Names() {
		f, _ := fire.Get(impl)
		for _, s := range scenarios {
			for _, n := range []int{512, 1024} {
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
}

// BenchmarkNew isole la construction du moteur, que BenchmarkRun inclut dans sa
// mesure et qu'Hyperfine amortit sur dix fois plus de tours.
//
// Elle a été écrite pour trancher une hypothèse — que l'avance de flat sur
// Run/front venait d'un coût fixe de construction — et l'a réfutée : construire
// coûte 0,32 ms à naive et 0,37 ms à flat sur 1024², quand l'écart à expliquer
// est de 273 ms. Le gain est donc bien par tour, et dépend de la densité de feu.
//
// Elle reste utile telle quelle : elle borne ce qu'un coût fixe peut expliquer,
// et montre que le stockage contigu ne rend pas la construction plus rapide —
// flat alloue même 5,2 Mo contre 2,1 Mo, en 4 allocations contre 1026.
func BenchmarkNew(b *testing.B) {
	for _, impl := range fire.Names() {
		f, _ := fire.Get(impl)
		for _, n := range sizes {
			m := carte(n, 64)
			b.Run(fmt.Sprintf("impl=%s/size=%d", impl, n), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = f(m)
				}
			})
		}
	}
}
