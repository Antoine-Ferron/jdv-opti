package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gol-wildfire/internal/fire"
)

// call exécute une requête sur l'API et rend la réponse décodée.
func call(t *testing.T, h http.Handler, body string) payload {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/simulation", bytes.NewBufferString(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	var out payload
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestResetRenvoieUneCarteJouable(t *testing.T) {
	out := call(t, handler(), `{"action":"reset","preset":"varie","seed":42}`)

	if out.Width != width || out.Height != height {
		t.Fatalf("carte %dx%d, attendu %dx%d", out.Width, out.Height, width, height)
	}
	for _, champ := range []struct {
		nom string
		buf []byte
	}{{"terrain", out.Terrain}, {"wind", out.Wind}, {"fire", out.Fire}, {"rest", out.Rest}} {
		if len(champ.buf) != width*height {
			t.Errorf("%s : %d cases, attendu %d", champ.nom, len(champ.buf), width*height)
		}
	}
	for i, t2 := range out.Terrain {
		if fire.Terrain(t2) > fire.Forest {
			t.Fatalf("case %d : terrain inconnu %d", i, t2)
		}
	}
	// Les foyers de départ de la carte doivent déjà brûler (REGLES.md §6).
	if out.Burning == 0 || out.Extinct {
		t.Errorf("aucun foyer allumé au départ : burning=%d extinct=%v", out.Burning, out.Extinct)
	}
	if out.Turn != 0 {
		t.Errorf("tour = %d au départ, attendu 0", out.Turn)
	}
}

func TestMemeGraineMemeCarte(t *testing.T) {
	a := call(t, handler(), `{"action":"reset","preset":"varie","seed":7}`)
	b := call(t, handler(), `{"action":"reset","preset":"varie","seed":7}`)
	if !bytes.Equal(a.Terrain, b.Terrain) || !bytes.Equal(a.Fire, b.Fire) {
		t.Fatal("deux resets de même graine donnent des cartes différentes")
	}
	c := call(t, handler(), `{"action":"reset","preset":"varie","seed":8}`)
	if bytes.Equal(a.Terrain, c.Terrain) {
		t.Fatal("deux graines différentes donnent la même carte")
	}
}

func TestStepAvanceLaSimulation(t *testing.T) {
	h := handler()
	avant := call(t, h, `{"action":"reset","preset":"vent","seed":3}`)
	apres := call(t, h, `{"action":"step"}`)

	if apres.Turn != 1 {
		t.Errorf("tour = %d après un pas, attendu 1", apres.Turn)
	}
	if bytes.Equal(avant.Fire, apres.Fire) {
		t.Error("l'état de combustion n'a pas changé après un tour")
	}
	// Le décor n'est pas renvoyé à chaque pas : c'est tout l'intérêt.
	if apres.Terrain != nil || apres.Width != 0 {
		t.Error("le décor ne devrait être envoyé qu'au reset")
	}
}

func TestRequetesInvalides(t *testing.T) {
	cas := map[string]string{
		"action inconnue": `{"action":"galope"}`,
		"paysage inconnu": `{"action":"reset","preset":"lune"}`,
		"corps illisible": `pas du json`,
	}
	for nom, body := range cas {
		t.Run(nom, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/simulation", bytes.NewBufferString(body)))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("statut = %d, attendu 400", w.Code)
			}
		})
	}
}

func TestPage(t *testing.T) {
	w := httptest.NewRecorder()
	handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("<canvas")) {
		t.Fatal("page indisponible")
	}
}
