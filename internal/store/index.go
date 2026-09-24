package store

import (
	"context"
	"fmt"
	"strings"
)

// La requête d'analyse mesurée : retrouver les tours dont l'empreinte se répète.
//
// Elle a un sens pour la simulation — deux tours de même empreinte signalent un
// état déjà vu, donc un cycle — et c'est justement le genre de recherche qu'un
// index sur fingerprint doit transformer. Sans index, PostgreSQL n'a d'autre
// choix qu'un parcours séquentiel de toute la table.
const RequeteAnalyse = `
SELECT fingerprint, count(*) AS occurrences
FROM turns
WHERE fingerprint = $1
GROUP BY fingerprint`

// IndexName est le nom de l'index évalué.
const IndexName = "idx_turns_fingerprint"

// AddIndex crée l'index sur fingerprint, puis met à jour les statistiques :
// sans ANALYZE, le planificateur raisonne sur des estimations périmées et peut
// ignorer un index qu'il vient de recevoir.
func (s *Store) AddIndex(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx,
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON turns (fingerprint)`, IndexName)); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `ANALYZE turns`)
	return err
}

// DropIndex retire l'index et remet les statistiques à jour.
func (s *Store) DropIndex(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx,
		fmt.Sprintf(`DROP INDEX IF EXISTS %s`, IndexName)); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `ANALYZE turns`)
	return err
}

// Explain exécute EXPLAIN (ANALYZE, BUFFERS) sur la requête d'analyse et rend le
// plan tel que PostgreSQL le produit — c'est la pièce à conviction du rapport,
// on ne la reformule pas.
func (s *Store) Explain(ctx context.Context, fingerprint uint64) (string, error) {
	rows, err := s.pool.Query(ctx,
		`EXPLAIN (ANALYZE, BUFFERS, COSTS) `+RequeteAnalyse, int64(fingerprint))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var ligne string
		if err := rows.Scan(&ligne); err != nil {
			return "", err
		}
		plan.WriteString(ligne)
		plan.WriteByte('\n')
	}
	return plan.String(), rows.Err()
}

// Interroge exécute la requête d'analyse et renvoie le nombre d'occurrences.
// C'est ce que mesure le benchmark : le coût réel de la requête, hors EXPLAIN.
func (s *Store) Interroge(ctx context.Context, fingerprint uint64) (int, error) {
	var empreinte int64
	var occurrences int
	err := s.pool.QueryRow(ctx, RequeteAnalyse, int64(fingerprint)).Scan(&empreinte, &occurrences)
	if err != nil {
		return 0, nil // aucune ligne : l'empreinte n'apparaît pas, ce n'est pas une erreur
	}
	return occurrences, nil
}
