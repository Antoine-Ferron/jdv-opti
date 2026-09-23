// Package snapshottest est la suite de conformité des formats de snapshot.
//
// Elle vérifie une seule chose, mais sur tous les formats : ce qui est écrit
// est relu à l'identique. Sans cette garantie, comparer les tailles produites
// n'aurait pas de sens, puisque le format le plus compact serait celui qui perd
// le plus d'information.
package snapshottest

import (
	"bytes"
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
	"gol-wildfire/internal/snapshot"
)

// etat engendre une carte, la fait brûler quelques tours, et relève son état :
// on veut des compteurs non nuls et variés, pas une grille vide.
func etat(w, h int, seed int64, tours int) *snapshot.State {
	cfg := fire.DefaultConfig(w, h, seed)
	cfg.Fires = 4
	e := naive.New(fire.Generate(cfg))
	for i := 0; i < tours; i++ {
		e.Step()
	}
	return snapshot.Capture(e, tours)
}

// Run exécute la suite de conformité sur un format.
func Run(t *testing.T, f snapshot.Format) {
	cas := []struct {
		nom   string
		w, h  int
		seed  int64
		tours int
	}{
		{"carte_courante", 64, 40, 42, 12},
		{"une_seule_case", 1, 1, 1, 0},
		{"largeur_non_multiple_de_8", 37, 11, 7, 5},
		{"tres_plate", 130, 3, 11, 6},
		{"etat_initial", 48, 48, 3, 0},
	}
	for _, c := range cas {
		t.Run("AllerRetour_"+c.nom, func(t *testing.T) {
			avant := etat(c.w, c.h, c.seed, c.tours)

			var buf bytes.Buffer
			if err := f.Write(&buf, avant); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			apres, err := f.Read(&buf)
			if err != nil {
				t.Fatalf("lecture : %v", err)
			}
			if err := avant.Equal(apres); err != nil {
				t.Fatalf("l'état relu diffère de l'état écrit : %v", err)
			}
		})
	}

	t.Run("LectureDeDonneesInvalides", func(t *testing.T) {
		if _, err := f.Read(bytes.NewReader([]byte("ceci n'est pas un snapshot"))); err == nil {
			t.Error("des données invalides devraient produire une erreur")
		}
	})
}
