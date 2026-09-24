// Package store persiste l'historique d'une simulation dans PostgreSQL.
//
// C'est le volet « I/O réseau & persistance » de l'audit (§3.3 du rapport). Il
// sert deux mesures, et le code est écrit pour qu'elles soient comparables :
//
//	InsertTurns    une requête par tour — le nombre d'allers-retours domine
//	CopyTurns      un seul COPY pour toute la série
//
// Les deux écrivent exactement les mêmes lignes ; seule la façon de les
// transmettre change. Toute autre différence rendrait la comparaison douteuse.
//
// La connexion se fait par DSN, jamais en dur : le même code doit tourner sur
// les deux bancs, et la base est locale ici mais pourrait ne pas l'être.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Turn est une ligne d'historique : ce qu'on veut pouvoir interroger après coup.
type Turn struct {
	Turn        int
	Fingerprint uint64
	Burning     int
	Burned      int // surface parcourue par le feu depuis le début
}

// Store est une connexion à la base.
type Store struct{ pool *pgxpool.Pool }

// Open ouvre un pool de connexions et vérifie que la base répond.
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: connexion : %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: la base ne répond pas : %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// schema crée les tables si elles n'existent pas.
//
// Aucun index sur turns.fingerprint : c'est l'objet même de la mesure, il est
// ajouté puis retiré par AddIndex et DropIndex.
const schema = `
CREATE TABLE IF NOT EXISTS runs (
    id         BIGSERIAL PRIMARY KEY,
    impl       TEXT        NOT NULL,
    width      INT         NOT NULL,
    height     INT         NOT NULL,
    seed       BIGINT      NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS turns (
    run_id      BIGINT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    turn        INT    NOT NULL,
    fingerprint BIGINT NOT NULL,
    burning     INT    NOT NULL,
    burned      INT    NOT NULL,
    PRIMARY KEY (run_id, turn)
);`

// Migrate crée le schéma.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schema)
	return err
}

// Reset vide les tables. Les mesures doivent partir d'un état connu : un index
// évalué sur une table dont on ignore le contenu ne prouve rien.
func (s *Store) Reset(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `TRUNCATE runs RESTART IDENTITY CASCADE`)
	return err
}

// NewRun enregistre une exécution et renvoie son identifiant.
func (s *Store) NewRun(ctx context.Context, impl string, w, h int, seed int64) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO runs (impl, width, height, seed) VALUES ($1, $2, $3, $4) RETURNING id`,
		impl, w, h, seed).Scan(&id)
	return id, err
}

// InsertTurns écrit les tours un par un : une requête, donc un aller-retour
// réseau, par tour. C'est la version naïve de l'axe.
//
// fingerprint est un uint64 stocké dans un BIGINT signé : la conversion est un
// changement d'interprétation des mêmes 64 bits, sans perte, et la valeur ne
// sert qu'à des comparaisons d'égalité.
func (s *Store) InsertTurns(ctx context.Context, runID int64, turns []Turn) error {
	for _, t := range turns {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO turns (run_id, turn, fingerprint, burning, burned) VALUES ($1, $2, $3, $4, $5)`,
			runID, t.Turn, int64(t.Fingerprint), t.Burning, t.Burned)
		if err != nil {
			return fmt.Errorf("store: tour %d : %w", t.Turn, err)
		}
	}
	return nil
}

// CopyTurns écrit les mêmes lignes en un seul COPY.
func (s *Store) CopyTurns(ctx context.Context, runID int64, turns []Turn) error {
	_, err := s.pool.CopyFrom(ctx,
		pgx.Identifier{"turns"},
		[]string{"run_id", "turn", "fingerprint", "burning", "burned"},
		pgx.CopyFromSlice(len(turns), func(i int) ([]any, error) {
			t := turns[i]
			return []any{runID, t.Turn, int64(t.Fingerprint), t.Burning, t.Burned}, nil
		}))
	return err
}

// CountTurns renvoie le nombre de lignes de turns, pour vérifier une insertion.
func (s *Store) CountTurns(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM turns`).Scan(&n)
	return n, err
}
