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
