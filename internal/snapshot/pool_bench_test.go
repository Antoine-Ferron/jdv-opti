package snapshot

import (
	"fmt"
	"io"
	"testing"
)

// BenchmarkPool confronte l'hypothèse I/O-5 A : le recyclage des tampons
// réduit-il le *temps*, et pas seulement le compte d'allocations ?
//
// Les deux variantes sont mesurées dans un même fichier de résultats. Les
// comparer entre deux campagnes séparées ferait porter l'écart autant sur
// l'état de la machine que sur le changement de code — c'est précisément le
// biais que le protocole à deux bancs cherche à éviter.
//
// Le premier appel de chaque variante est hors mesure : avec un pool vide, il
// alloue comme la variante sans pool, et l'inclure diluerait l'effet cherché.
func BenchmarkPool(b *testing.B) {
	defer func() { poolActif = true }()

	for _, taille := range []int{256, 1024} {
		s := etatSimple(taille, taille, 3)
		for _, format := range []struct {
			nom string
			f   Format
		}{
			{"packed", Packed{}},
			{"proto", Proto{}},
		} {
			for _, actif := range []bool{false, true} {
				nom := fmt.Sprintf("pool=%t/format=%s/size=%d", actif, format.nom, taille)
				b.Run(nom, func(b *testing.B) {
					videLesPools()
					poolActif = actif
					if err := format.f.Write(io.Discard, s); err != nil {
						b.Fatal(err)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if err := format.f.Write(io.Discard, s); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}
}
