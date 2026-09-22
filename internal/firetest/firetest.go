// Package firetest traduit REGLES.md en tests. Toute implémentation de
// fire.Engine doit passer cette suite : une optimisation qui casse une règle
// est rejetée avant d'être mesurée.
//
// Chaque sous-test porte le numéro de la section qu'il vérifie.
package firetest

import (
	"testing"

	"gol-wildfire/internal/fire"
)

// ParseMap construit une carte depuis un croquis ASCII, une ligne par rangée :
// '~' eau, '.' plaine, '#' forêt. Le vent et les foyers se posent ensuite avec
// SetWind et Ignite, pour qu'ils restent lisibles dans le test.
func ParseMap(rows ...string) fire.Map {
	h := len(rows)
	w := len(rows[0])
	m := fire.NewMap(w, h)
	for y, row := range rows {
		if len(row) != w {
			panic("firetest: lignes de largeurs différentes")
		}
		for x, c := range row {
			switch c {
			case '~':
				m.Terrain[y*w+x] = fire.Water
			case '.':
				m.Terrain[y*w+x] = fire.Plain
			case '#':
				m.Terrain[y*w+x] = fire.Forest
			default:
				panic("firetest: caractère de terrain inconnu : " + string(c))
			}
		}
	}
	return m
}

// etat vérifie les deux compteurs d'une case.
func etat(t *testing.T, e fire.Engine, x, y int, feu, repos uint8, quand string) {
	t.Helper()
	if got := e.Fire(x, y); got != feu {
		t.Errorf("%s : feu(%d,%d) = %d, attendu %d", quand, x, y, got, feu)
	}
	if got := e.Rest(x, y); got != repos {
		t.Errorf("%s : repos(%d,%d) = %d, attendu %d", quand, x, y, got, repos)
	}
}

// isolee renvoie une carte 5x5 d'eau avec un seul îlot combustible au centre,
// en (2,2) : ce qui s'y passe ne doit rien à la propagation.
func isolee(terrain string) fire.Map {
	m := ParseMap(
		"~~~~~",
		"~~~~~",
		"~~"+terrain+"~~",
		"~~~~~",
		"~~~~~")
	m.Ignite(2, 2)
	return m
}

// Run exécute la suite de conformité sur une implémentation.
func Run(t *testing.T, f fire.Factory) {
	t.Run("§2_PlaineBrule1TourPuisRepos2", func(t *testing.T) {
		e := f(isolee("."))
		etat(t, e, 2, 2, 1, 0, "au départ")
		e.Step()
		etat(t, e, 2, 2, 0, 2, "après 1 tour")
		e.Step()
		etat(t, e, 2, 2, 0, 1, "après 2 tours")
		e.Step()
		etat(t, e, 2, 2, 0, 0, "après 3 tours : de nouveau inflammable")
	})

	t.Run("§2_ForetBrule2ToursPuisRepos3", func(t *testing.T) {
		e := f(isolee("#"))
		etat(t, e, 2, 2, 2, 0, "au départ")
		e.Step()
		etat(t, e, 2, 2, 1, 0, "après 1 tour : brûle encore")
		e.Step()
		etat(t, e, 2, 2, 0, 3, "après 2 tours")
		e.Step()
		e.Step()
		etat(t, e, 2, 2, 0, 1, "après 4 tours")
		e.Step()
		etat(t, e, 2, 2, 0, 0, "après 5 tours : de nouveau inflammable")
	})

	t.Run("§3_BurningCompteLesCasesEnFeu", func(t *testing.T) {
		e := f(isolee("."))
		if got := e.Burning(); got != 1 {
			t.Fatalf("au départ : Burning() = %d, attendu 1", got)
		}
		e.Step()
		if got := e.Burning(); got != 0 {
			t.Fatalf("après extinction : Burning() = %d, attendu 0", got)
		}
	})

	runPropagation(t, f)
	runVent(t, f)
	runBoucle(t, f)
	runConformite(t, f)
}

// runPropagation regroupe les règles de propagation (§1, §4, §5) hors vent.
func runPropagation(t *testing.T, f fire.Factory) {
	t.Run("§5_PropageAuxHuitVoisins", func(t *testing.T) {
		m := ParseMap(
			".....",
			".....",
			".....",
			".....",
			".....")
		m.Ignite(2, 2)
		e := f(m)
		e.Step()
		etat(t, e, 2, 2, 0, 2, "le foyer s'est consumé")
		for _, p := range [][2]int{{1, 1}, {2, 1}, {3, 1}, {1, 2}, {3, 2}, {1, 3}, {2, 3}, {3, 3}} {
			etat(t, e, p[0], p[1], 1, 0, "voisin de Moore")
		}
	})

	t.Run("§1_PropagationSimultanee", func(t *testing.T) {
		// Les cases enflammées ce tour-ci ne propagent qu'au tour suivant :
		// après un seul tour, rien ne doit brûler à distance 2 du foyer.
		m := ParseMap(
			".....",
			".....",
			".....",
			".....",
			".....")
		m.Ignite(2, 2)
		e := f(m)
		e.Step()
		for _, p := range [][2]int{{0, 0}, {2, 0}, {4, 2}, {0, 4}, {4, 4}} {
			etat(t, e, p[0], p[1], 0, 0, "distance 2 : pas de cascade dans le même tour")
		}
	})

	t.Run("§1_TorePropageParLeBord", func(t *testing.T) {
		m := ParseMap(
			".....",
			".....",
			".....")
		m.Ignite(0, 1)
		e := f(m)
		e.Step()
		for _, p := range [][2]int{{4, 0}, {4, 1}, {4, 2}} {
			etat(t, e, p[0], p[1], 1, 0, "colonne opposée, atteinte par le bord")
		}
	})

	t.Run("§5_EauNePrendJamaisFeu", func(t *testing.T) {
		m := ParseMap(
			".....",
			"..~..",
			".....")
		m.Ignite(2, 2)
		e := f(m)
		e.Step()
		etat(t, e, 2, 1, 0, 0, "l'eau entourée de feu")
		e.Step()
		etat(t, e, 2, 1, 0, 0, "l'eau, deux tours plus tard")
	})

	t.Run("§5_NeTraversePasUneRiviereSansVent", func(t *testing.T) {
		// Colonne 3 entièrement en eau : pour passer de x<3 à x>3 il faut la
		// traverser, ou contourner par le bord — ce qui demande bien plus de
		// deux tours sur une carte de 7 de large.
		m := ParseMap(
			"...~...",
			"...~...",
			"...~...")
		m.Ignite(2, 1)
		e := f(m)
		e.Step()
		e.Step()
		for y := 0; y < 3; y++ {
			etat(t, e, 4, y, 0, 0, "derrière la rivière, sans vent")
		}
	})

	t.Run("§4_CaseEnFeuIgnoreContagion", func(t *testing.T) {
		// Deux forêts voisines : la seconde, enflammée au tour 1, recontamine la
		// première qui brûle encore. Son compteur ne doit pas repartir de 2.
		m := ParseMap(
			"~~~~",
			"~##~",
			"~~~~")
		m.Ignite(1, 1)
		e := f(m)
		etat(t, e, 1, 1, 2, 0, "au départ")
		e.Step()
		etat(t, e, 1, 1, 1, 0, "tour 1 : dernier tour de combustion")
		etat(t, e, 2, 1, 2, 0, "tour 1 : la voisine s'enflamme")
		e.Step()
		etat(t, e, 1, 1, 0, 3, "tour 2 : consumée malgré la contagion de sa voisine")
	})

	t.Run("§4_CaseEnReposIgnoreContagion", func(t *testing.T) {
		// La plaine s'éteint au tour 1 et entre en repos ; la forêt voisine
		// brûle encore deux tours et la contamine en vain.
		m := ParseMap(
			"~~~~",
			"~.#~",
			"~~~~")
		m.Ignite(1, 1)
		e := f(m)
		e.Step()
		etat(t, e, 1, 1, 0, 2, "tour 1 : la plaine est en repos")
		etat(t, e, 2, 1, 2, 0, "tour 1 : la forêt s'enflamme")
		e.Step()
		etat(t, e, 1, 1, 0, 1, "tour 2 : le repos s'écoule, le feu voisin est sans effet")
		e.Step()
		etat(t, e, 1, 1, 0, 0, "tour 3 : de nouveau disponible")
	})
}

// runVent regroupe les règles propres aux cases vent (§5).
func runVent(t *testing.T, f fire.Factory) {
	t.Run("§5_VentEpargneLesTroisCasesAmont", func(t *testing.T) {
		m := ParseMap(
			".....",
			".....",
			".....",
			".....",
			".....")
		m.SetWind(2, 2, fire.East)
		m.Ignite(2, 2)
		e := f(m)
		e.Step()
		for _, p := range [][2]int{{1, 1}, {1, 2}, {1, 3}} {
			etat(t, e, p[0], p[1], 0, 0, "secteur amont : ouest et ses deux diagonales")
		}
		for _, p := range [][2]int{{2, 1}, {3, 1}, {3, 2}, {2, 3}, {3, 3}} {
			etat(t, e, p[0], p[1], 1, 0, "les cinq voisins restants brûlent")
		}
	})

	t.Run("§5_VentBruleAussiLaCaseIntermediaire", func(t *testing.T) {
		m := ParseMap(
			".....",
			".....",
			".....",
			".....",
			".....")
		m.SetWind(2, 2, fire.East)
		m.Ignite(2, 2)
		e := f(m)
		e.Step()
		etat(t, e, 3, 2, 1, 0, "distance 1 sous le vent")
		etat(t, e, 4, 2, 1, 0, "distance 2 sous le vent : le saut")
	})

	t.Run("§5_VentFranchitUneRiviere", func(t *testing.T) {
		m := ParseMap(
			"...~...",
			"...~...",
			"...~...")
		m.SetWind(2, 1, fire.East)
		m.Ignite(2, 1)
		e := f(m)
		e.Step()
		etat(t, e, 3, 1, 0, 0, "la rivière ne brûle pas")
		etat(t, e, 4, 1, 1, 0, "mais le feu a sauté par-dessus")
	})

	t.Run("§5_VentSurEauNePropagePas", func(t *testing.T) {
		// Le vent posé sur de l'eau est inerte : l'eau ne brûle jamais, donc
		// elle ne projette rien.
		m := ParseMap(
			".......",
			"..~....",
			".......")
		m.SetWind(2, 1, fire.East)
		m.Ignite(1, 1)
		e := f(m)
		e.Step()
		e.Step()
		etat(t, e, 2, 1, 0, 0, "l'eau ventée ne brûle pas")
		etat(t, e, 4, 1, 0, 0, "et ne projette pas de feu à deux cases")
	})

	t.Run("§5_EffetDuVentNeSeTransmetPas", func(t *testing.T) {
		// Les cases allumées par une case vent sont ordinaires : au tour
		// suivant elles repropagent vers l'amont, que le vent avait épargné.
		m := ParseMap(
			".......",
			".......",
			".......",
			".......",
			".......")
		m.SetWind(3, 2, fire.East)
		m.Ignite(3, 2)
		e := f(m)
		e.Step()
		for _, p := range [][2]int{{2, 1}, {2, 2}, {2, 3}} {
			etat(t, e, p[0], p[1], 0, 0, "tour 1 : amont épargné")
		}
		e.Step()
		for _, p := range [][2]int{{2, 1}, {2, 2}, {2, 3}} {
			etat(t, e, p[0], p[1], 1, 0, "tour 2 : l'amont brûle, le vent ne s'est pas transmis")
		}
	})
}

// runBoucle vérifie l'empreinte et les conditions d'arrêt (§6).
func runBoucle(t *testing.T, f fire.Factory) {
	t.Run("Empreinte_EtatsIdentiquesMemeEmpreinte", func(t *testing.T) {
		a, b := f(isolee(".")), f(isolee("."))
		if a.Fingerprint() != b.Fingerprint() {
			t.Fatal("deux états identiques ont des empreintes différentes")
		}
		b.Step()
		if a.Fingerprint() == b.Fingerprint() {
			t.Fatal("deux états différents ont la même empreinte")
		}
	})

	t.Run("§6_ArretQuandPlusRienNeBrule", func(t *testing.T) {
		res := fire.Run(f(isolee(".")), fire.Options{Turns: 100})
		if !res.Extinct {
			t.Error("l'incendie devait s'éteindre")
		}
		if res.Turns != 1 {
			t.Errorf("arrêt au tour %d, attendu 1 : la plaine isolée brûle un seul tour", res.Turns)
		}
	})

	t.Run("§6_ArretApresNTours", func(t *testing.T) {
		m := ParseMap(
			".........",
			".........",
			".........",
			".........",
			".........")
		m.Ignite(4, 2)
		res := fire.Run(f(m), fire.Options{Turns: 3})
		if res.Turns != 3 {
			t.Errorf("%d tours calculés, attendu 3", res.Turns)
		}
		if res.Extinct {
			t.Error("l'incendie ne devait pas être éteint après 3 tours")
		}
	})
}
