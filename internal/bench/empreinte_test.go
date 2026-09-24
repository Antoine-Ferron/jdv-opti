package bench

import (
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
	"gol-wildfire/internal/snapshot"
)

// BenchmarkEmpreinte oppose le Fingerprint des moteurs, identique dans les
// trois implémentations, au hachage écrit pour la déduplication de l'axe I/O.
//
// Les deux répondent à la même question — deux grilles sont-elles dans le même
// état — mais l'un passe par un fmt.Sprintf et une concaténation par case
// ([F6] de la baseline), l'autre par un FNV-1a sans allocation. Ce banc chiffre
// le défaut [F6] et fournit son remplaçant tout prêt ; il relève de l'axe CPU,
// pas de l'axe I/O.
func BenchmarkEmpreinte(b *testing.B) {
	const n = 1024
	cfg := fire.DefaultConfig(n, n, 42)
	cfg.Fires = 64
	e := naive.New(fire.Generate(cfg))
	for i := 0; i < 50; i++ {
		e.Step()
	}
	etat := snapshot.Capture(e, 50)

	b.Run("methode=Fingerprint/size=1024", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = e.Fingerprint()
		}
	})
	b.Run("methode=Empreinte/size=1024", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = etat.Empreinte()
		}
	})
}
