package snapshot_test

import (
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
	"gol-wildfire/internal/snapshot"
)

// empreintes rejoue une simulation et rend l'empreinte de chaque tour.
func empreintes(cote, tours int) []uint64 {
	cfg := fire.DefaultConfig(cote, cote, 42)
	cfg.Fires = 1
	e := naive.New(fire.Generate(cfg))

	out := make([]uint64, 0, tours+1)
	for t := 0; t <= tours; t++ {
		out = append(out, snapshot.Capture(e, t).Empreinte())
		e.Step()
	}
	return out
}

// La déduplication de l'étape I/O-5 reposait sur une prémisse fausse : que
// l'incendie finisse par s'éteindre et se fige. Il ne s'éteint jamais — les
// cases redeviennent combustibles après leur repos — mais il entre dans un
// cycle, ce qui rend les états répétés bien plus fréquents que prévu.
//
// Ce test établit les deux chiffres dont dépend l'intérêt de l'étape : le tour
// où le cycle s'installe, et sa période. Il échoue si les états cessent de se
// répéter, auquel cas la conclusion du §3.3 devrait être reprise.
func TestCycleDesEtats(t *testing.T) {
	const (
		cote  = 128
		tours = 2000
	)
	emp := empreintes(cote, tours)

	vues := make(map[uint64]int, len(emp))
	premier, origine := -1, -1
	repetes := 0
	for t, e := range emp {
		if precedent, deja := vues[e]; deja {
			repetes++
			if premier < 0 {
				premier, origine = t, precedent
			}
			continue
		}
		vues[e] = t
	}

	if repetes == 0 {
		t.Fatal("aucun état répété en 2000 tours : la conclusion du §3.3 sur la " +
			"déduplication doit être reprise")
	}
	t.Logf("%d tours, %d états distincts, %d répétitions (%.1f %%)",
		len(emp), len(vues), repetes, float64(repetes)/float64(len(emp))*100)
	t.Logf("premier état répété au tour %d, identique au tour %d — période %d",
		premier, origine, premier-origine)

	// Le point décisif pour l'étape : un snapshot périodique ne capte le cycle
	// que si sa période et celle du cycle s'accordent. Ce tableau est la mesure
	// qui tranche, et il est cité au §3.3.
	t.Log("taux de doublons selon la période d'échantillonnage :")
	for _, periode := range []int{1, 2, 5, 7, 10, 25, 50} {
		d := snapshot.NewDedup(len(emp))
		for i := 0; i < len(emp); i += periode {
			d.DejaVu(emp[i])
		}
		t.Logf("  toutes les %2d tours : %3d relevés, %.1f %% évités",
			periode, d.Vus, d.Taux()*100)
	}
}
