package bench

import (
	"fmt"
	"io"
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
	"gol-wildfire/internal/snapshot"
)

// etatCapture prépare un état représentatif : carte engendrée, incendie mené
// jusqu'à son régime. Le relevé lui-même est hors chronomètre — on mesure la
// sérialisation, pas la lecture du moteur.
func etatCapture(n int) *snapshot.State {
	cfg := fire.DefaultConfig(n, n, 42)
	cfg.Fires = 64
	e := naive.New(fire.Generate(cfg))
	for i := 0; i < 50; i++ {
		e.Step()
	}
	return snapshot.Capture(e, 50)
}

// BenchmarkSnapshotWrite mesure le coût de sérialisation de chaque format.
//
// SetBytes reçoit la taille produite, si bien que la colonne B/s de benchstat
// donne le débit d'écriture — et non un débit de cases, qui n'aurait pas de
// sens quand les formats produisent des volumes aussi différents.
//
// La taille par case est publiée comme métrique (`o/case`) plutôt que relevée à
// la main : c'est le chiffre central de l'axe I/O, il doit atterrir dans
// results/ avec les autres et non dans un test jetable. Elle ne dépend pas de
// la machine, mais la mesurer ici garantit qu'elle est reproductible.
func BenchmarkSnapshotWrite(b *testing.B) {
	for _, n := range []int{256, 1024} {
		s := etatCapture(n)
		for _, nom := range snapshot.Names() {
			f, _ := snapshot.Get(nom)
			b.Run(fmt.Sprintf("format=%s/size=%d", nom, n), func(b *testing.B) {
				taille := &compteur{}
				if err := f.Write(taille, s); err != nil {
					b.Fatal(err)
				}
				b.SetBytes(taille.n)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := f.Write(io.Discard, s); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(taille.n)/float64(n*n), "o/case")
				b.ReportMetric(float64(taille.n), "o/snapshot")
			})
		}
	}
}

// BenchmarkSnapshotPacked mesure la variante sans décor : dans une série, seul
// le premier snapshot a besoin du terrain et du vent, qui sont immuables.
func BenchmarkSnapshotPacked(b *testing.B) {
	n := 1024
	s := etatCapture(n)
	for _, cas := range []struct {
		nom string
		f   snapshot.Packed
	}{
		{"avec-decor", snapshot.Packed{}},
		{"sans-decor", snapshot.Packed{SansDecor: true}},
	} {
		b.Run(fmt.Sprintf("variante=%s/size=%d", cas.nom, n), func(b *testing.B) {
			taille := &compteur{}
			if err := cas.f.Write(taille, s); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(taille.n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := cas.f.Write(io.Discard, s); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(taille.n)/float64(n*n), "o/case")
			b.ReportMetric(float64(taille.n), "o/snapshot")
		})
	}
}

// BenchmarkSnapshotRead mesure la relecture, qui compte autant que l'écriture
// pour un format d'archive.
func BenchmarkSnapshotRead(b *testing.B) {
	n := 1024
	s := etatCapture(n)
	for _, nom := range snapshot.Names() {
		f, _ := snapshot.Get(nom)
		var buf tampon
		if err := f.Write(&buf, s); err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("format=%s/size=%d", nom, n), func(b *testing.B) {
			b.SetBytes(int64(len(buf.b)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := f.Read(buf.lecteur()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// compteur compte les octets écrits sans rien conserver.
type compteur struct{ n int64 }

func (c *compteur) Write(p []byte) (int, error) { c.n += int64(len(p)); return len(p), nil }

// tampon conserve les octets écrits pour pouvoir les relire plusieurs fois.
type tampon struct{ b []byte }

func (t *tampon) Write(p []byte) (int, error) { t.b = append(t.b, p...); return len(p), nil }

func (t *tampon) lecteur() io.Reader { return &relecture{b: t.b} }

type relecture struct {
	b []byte
	i int
}

func (r *relecture) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
