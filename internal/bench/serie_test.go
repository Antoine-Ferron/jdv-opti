package bench

import (
	"fmt"
	"io"
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
	"gol-wildfire/internal/snapshot"
)

// serie rejoue une simulation et relève un état tous les periode tours, comme
// le ferait une campagne qui archive son historique.
//
// Les paramètres ne sont pas libres, ils sont dictés par la dynamique mesurée
// (TestCycleDesEtats) : l'incendie ne s'éteint jamais, mais il entre en cycle
// de période 24 vers le tour 429. Une série qui s'arrête à 500 tours n'en voit
// presque rien ; il faut la pousser à 2 000 pour mesurer la déduplication en
// régime cyclique.
//
// La taille est ramenée à 128² parce que la série est conservée en mémoire :
// 201 états en 1024² pèseraient 800 Mio et mesureraient surtout la pression
// mémoire du banc. C'est aussi la taille à laquelle le cycle a été caractérisé.
func serie(n, tours, periode int) []*snapshot.State {
	cfg := fire.DefaultConfig(n, n, 42)
	cfg.Fires = 1
	e := naive.New(fire.Generate(cfg))

	var etats []*snapshot.State
	for t := 0; t <= tours; t++ {
		if t%periode == 0 {
			etats = append(etats, snapshot.Capture(e, t))
		}
		e.Step()
	}
	return etats
}

// puits empêche le compilateur de supprimer un calcul dont le résultat ne
// servirait à rien : sans lui, le banc mesurait 0,15 ns par case, soit moins
// d'un cycle pour deux multiplications dépendantes — un chiffre impossible.
var puits uint64

// BenchmarkSnapshotEmpreinte chiffre le test que paie la déduplication à chaque
// snapshot, qu'il en évite un ou non.
//
// C'est la moitié de l'inégalité qui décide de la rentabilité de l'étape I/O-5 :
// le hachage doit coûter moins que l'écriture qu'il permet d'éviter. Sans ce
// chiffre, le verdict du §3.3 ne serait pas reproductible.
func BenchmarkSnapshotEmpreinte(b *testing.B) {
	for _, n := range []int{128, 1024} {
		s := snapshot.Capture(naive.New(fire.Generate(fire.DefaultConfig(n, n, 42))), 0)
		b.Run(fmt.Sprintf("size=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				puits = s.Empreinte()
			}
		})
	}
}

// BenchmarkSnapshotSerie mesure l'archivage d'une série entière, avec et sans
// déduplication (étape I/O-5, hypothèse I/O-5 B).
//
// Le coût du calcul d'empreinte est **dans** le chronomètre des variantes
// déduplicantes : c'est le prix à payer pour savoir s'il faut écrire. L'omettre
// donnerait à la déduplication un avantage qu'elle n'a pas.
//
// La métrique `%evites` publie le taux de doublons captés : c'est elle qui
// tranche entre un cache d'une entrée et un cache plus grand, et non une
// préférence de conception.
func BenchmarkSnapshotSerie(b *testing.B) {
	const n, tours, periode = 128, 2000, 10
	etats := serie(n, tours, periode)

	for _, nomFormat := range []string{"packed", "proto"} {
		f, ok := snapshot.Get(nomFormat)
		if !ok {
			b.Fatalf("format %s non enregistré", nomFormat)
		}
		for _, cache := range []int{0, 1, 8, 12, 16, 32} {
			nom := fmt.Sprintf("format=%s/cache=%d", nomFormat, cache)
			b.Run(nom, func(b *testing.B) {
				var dernier *snapshot.Dedup
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					d := snapshot.NewDedup(cache)
					for _, s := range etats {
						if cache > 0 && d.DejaVu(s.Empreinte()) {
							continue
						}
						if err := f.Write(io.Discard, s); err != nil {
							b.Fatal(err)
						}
					}
					dernier = d
				}
				b.StopTimer()
				b.ReportMetric(dernier.Taux()*100, "%evites")
				b.ReportMetric(float64(len(etats)), "snapshots/op")
			})
		}
	}
}
