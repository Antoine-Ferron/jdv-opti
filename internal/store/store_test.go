package store_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"gol-wildfire/internal/store"
)

// DSN par défaut : celui du docker-compose.yml du dépôt. Surchargeable pour
// pointer ailleurs, notamment si Docker n'expose pas la base sur localhost.
const dsnDefaut = "postgres://wildfire:wildfire@localhost:5432/wildfire?sslmode=disable"

// ouvre rend un Store prêt à l'emploi, ou saute le test si la base est absente.
//
// Ce saut est délibéré : la suite doit rester verte sur une machine sans Docker,
// sans quoi `make test` deviendrait un obstacle pour celui des deux qui ne
// travaille pas sur cet axe. Un test d'intégration qui échoue faute de base ne
// signale rien d'utile.
func ouvre(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("WILDFIRE_DSN")
	if dsn == "" {
		dsn = dsnDefaut
	}
	ctx := context.Background()
	s, err := store.Open(ctx, dsn)
	if err != nil {
		t.Skipf("base indisponible (%v) — lancer `make db` pour exécuter ce test", err)
	}
	t.Cleanup(s.Close)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migration : %v", err)
	}
	if err := s.Reset(ctx); err != nil {
		t.Fatalf("remise à zéro : %v", err)
	}
	return s
}

// tours engendre n lignes. modulo contrôle la distribution des empreintes :
// 0 les rend uniques, comme dans une vraie exécution où deux tours identiques
// signalent un cycle ; une valeur k les fait se répéter une fois sur k.
//
// Cette distribution n'est pas un détail de jeu d'essai : elle décide du plan
// d'exécution. Un prédicat qui retient 10 % de la table ne justifie aucun index,
// quel que soit le nombre de lignes.
func tours(n, modulo int) []store.Turn {
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

// Les deux modes d'insertion doivent écrire exactement les mêmes lignes : sans
// cela, comparer leurs temps n'aurait aucun sens.
func TestInsertionsEquivalentes(t *testing.T) {
	ctx := context.Background()
	s := ouvre(t)

	unitaire, err := s.NewRun(ctx, "naive", 1024, 1024, 42)
	if err != nil {
		t.Fatal(err)
	}
	groupe, err := s.NewRun(ctx, "naive", 1024, 1024, 42)
	if err != nil {
		t.Fatal(err)
	}

	lignes := tours(50, 10)
	if err := s.InsertTurns(ctx, unitaire, lignes); err != nil {
		t.Fatalf("insertion unitaire : %v", err)
	}
	if err := s.CopyTurns(ctx, groupe, lignes); err != nil {
		t.Fatalf("insertion groupée : %v", err)
	}

	n, err := s.CountTurns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := 2 * len(lignes); n != want {
		t.Fatalf("%d lignes écrites, attendu %d", n, want)
	}

	occurrences, err := s.Interroge(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := 2 * len(lignes) / 10; occurrences != want {
		t.Fatalf("empreinte 3 : %d occurrences, attendu %d", occurrences, want)
	}
}

// L'index doit changer le plan d'exécution, pas seulement le temps : c'est le
// passage de Seq Scan à Index Scan qui est la preuve attendue au §3.3.
//
// Empreintes uniques, comme dans une vraie exécution : c'est la seule
// distribution pour laquelle un index a un sens, et donc la seule dont le
// changement de plan démontre quelque chose.
func TestIndexChangeLePlan(t *testing.T) {
	ctx := context.Background()
	s := ouvre(t)

	run, err := s.NewRun(ctx, "naive", 1024, 1024, 42)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CopyTurns(ctx, run, tours(20000, 0)); err != nil {
		t.Fatal(err)
	}

	if err := s.DropIndex(ctx); err != nil {
		t.Fatal(err)
	}
	sans, err := s.Explain(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sans, "Seq Scan") {
		t.Errorf("sans index, un parcours séquentiel était attendu ; plan obtenu :\n%s", sans)
	}

	if err := s.AddIndex(ctx); err != nil {
		t.Fatal(err)
	}
	avec, err := s.Explain(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(avec, "Index") {
		t.Errorf("avec index, un parcours par index était attendu ; plan obtenu :\n%s", avec)
	}

	t.Logf("=== sans index ===\n%s\n=== avec index ===\n%s", sans, avec)
}
