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

	_ "gol/internal/engines"
	"gol/internal/life"
)

var sizes = []int{256, 1024, 2048}

// BenchmarkStep mesure le coût d'une génération (le cœur du hot path).
func BenchmarkStep(b *testing.B) {
	for _, impl := range life.Names() {
		f, _ := life.Get(impl)
		for _, n := range sizes {
			b.Run(fmt.Sprintf("impl=%s/size=%d", impl, n), func(b *testing.B) {
				e := f(n, n, life.RandomCells(n, n, 42, 0.3))
				b.ReportAllocs()
				b.SetBytes(int64(n * n)) // débit exprimé en cellules/s (1 "octet" = 1 cellule)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					e.Step()
				}
			})
		}
	}
}

// BenchmarkFingerprint mesure le coût de l'empreinte (détection de cycles).
func BenchmarkFingerprint(b *testing.B) {
	for _, impl := range life.Names() {
		f, _ := life.Get(impl)
		n := 1024
		b.Run(fmt.Sprintf("impl=%s/size=%d", impl, n), func(b *testing.B) {
			e := f(n, n, life.RandomCells(n, n, 42, 0.3))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = e.Fingerprint()
			}
		})
	}
}

// BenchmarkRun mesure un scénario complet (Step + détection de cycles).
func BenchmarkRun(b *testing.B) {
	for _, impl := range life.Names() {
		f, _ := life.Get(impl)
		n, gens := 512, 50
		cells := life.RandomCells(n, n, 42, 0.3)
		b.Run(fmt.Sprintf("impl=%s/size=%d", impl, n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				e := f(n, n, cells)
				if _, err := life.Run(e, life.Options{Generations: gens, DetectCycles: true}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
