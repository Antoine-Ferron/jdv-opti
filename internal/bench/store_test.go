package bench

import (
	"context"
	"fmt"
	"os"
	"testing"

	"gol-wildfire/internal/store"
)

// base ouvre la base de l'axe I/O, ou saute le banc si elle est absente.
// Même parti pris que les tests d'intégration : `make bench` doit rester
// utilisable sans Docker, les autres bancs n'ayant rien à voir avec PostgreSQL.
func base(b *testing.B) (context.Context, *store.Store) {
	b.Helper()
	dsn := os.Getenv("WILDFIRE_DSN")
	if dsn == "" {
		dsn = "postgres://wildfire:wildfire@localhost:5432/wildfire?sslmode=disable"
	}
	ctx := context.Background()
	s, err := store.Open(ctx, dsn)
	if err != nil {
		b.Skipf("base indisponible (%v) — lancer `make db` pour exécuter ce banc", err)
	}
	b.Cleanup(s.Close)
	if err := s.Migrate(ctx); err != nil {
		b.Fatal(err)
	}
	return ctx, s
}

// lignes engendre n tours. modulo = 0 donne des empreintes uniques, une valeur
// k les fait se répéter une fois sur k. Voir BenchmarkStoreRequete : cette
// distribution pèse plus lourd sur le plan d'exécution que le volume.
func lignes(n, modulo int) []store.Turn {
	out := make([]store.Turn, n)
	for i := range out {
		f := uint64(i)
		if modulo > 0 {
			f = uint64(i % modulo)
		}
		out[i] = store.Turn{Turn: i, Fingerprint: f, Burning: i * 3, Burned: i * 7}
	}
	return out
}

// BenchmarkStoreInsertion oppose une requête par tour à un seul COPY.
//
// C'est le vrai sujet « I/O réseau » de l'axe : les deux modes transmettent
// exactement les mêmes octets utiles, seul le nombre d'allers-retours change.
// La base est ici sur localhost, donc dans les conditions les *plus* favorables
// à la version unitaire — une latence réseau réelle ne ferait qu'écarter
// davantage les deux courbes.
func BenchmarkStoreInsertion(b *testing.B) {
	ctx, s := base(b)
	// 500 tours, la longueur d'une campagne, et 5 000 pour vérifier que l'écart
	// suit bien le nombre d'allers-retours et non le volume transmis.
	for _, n := range []int{500, 5000} {
		lot := lignes(n, 0)
		for _, cas := range []struct {
			nom    string
			insere func(context.Context, int64, []store.Turn) error
		}{
			{"unitaire", s.InsertTurns},
			{"copy", s.CopyTurns},
		} {
			b.Run(fmt.Sprintf("mode=%s/tours=%d", cas.nom, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					// Hors chronomètre : la table repart vide à chaque
					// itération, sans quoi elle enflerait et le temps mesuré
					// dériverait avec la taille des index de clé primaire.
					b.StopTimer()
					if err := s.Reset(ctx); err != nil {
						b.Fatal(err)
					}
					run, err := s.NewRun(ctx, "bench", 1024, 1024, 42)
					if err != nil {
						b.Fatal(err)
					}
					b.StartTimer()

					if err := cas.insere(ctx, run, lot); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(n), "tours/op")
			})
		}
	}
}

// BenchmarkStoreRequete mesure la requête d'analyse sans puis avec index, en
// croisant deux dimensions : le volume de la table et la distribution des
// empreintes.
//
// La seconde dimension n'était pas prévue. Elle a été ajoutée après une mesure
// qui contredisait l'hypothèse A : sur 20 000 lignes, l'index *allongeait* la
// requête. Le plan l'expliquait — avec une empreinte répétée une fois sur dix,
// le prédicat retient 10 % de la table, PostgreSQL doit de toute façon visiter
// presque toutes les pages, et le détour par l'index s'ajoute au parcours au
// lieu de l'éviter.
//
// Les deux distributions sont donc conservées : « unique » est le cas réel, où
// une empreinte répétée signale un cycle ; « repetee » est le contre-exemple
// qui montre qu'un index ne se juge pas au nombre de lignes mais à la
// sélectivité du prédicat.
func BenchmarkStoreRequete(b *testing.B) {
	ctx, s := base(b)
	for _, dist := range []struct {
		nom    string
		modulo int
	}{
		{"unique", 0},
		{"repetee", 10},
	} {
		for _, n := range []int{200, 20_000, 500_000} {
			if err := s.Reset(ctx); err != nil {
				b.Fatal(err)
			}
			run, err := s.NewRun(ctx, "bench", 1024, 1024, 42)
			if err != nil {
				b.Fatal(err)
			}
			if err := s.CopyTurns(ctx, run, lignes(n, dist.modulo)); err != nil {
				b.Fatal(err)
			}

			for _, cas := range []struct {
				nom     string
				prepare func(context.Context) error
			}{
				{"sans-index", s.DropIndex},
				{"avec-index", s.AddIndex},
			} {
				if err := cas.prepare(ctx); err != nil {
					b.Fatal(err)
				}
				b.Run(fmt.Sprintf("index=%s/empreintes=%s/lignes=%d", cas.nom, dist.nom, n), func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						if _, err := s.Interroge(ctx, 3); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}
	if err := s.DropIndex(ctx); err != nil {
		b.Fatal(err)
	}
}
