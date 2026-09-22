package fire_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/naive"
)

var ansi = regexp.MustCompile(string(rune(27)) + "[[][0-9;]*m")

// rendu renvoie le dessin débarrassé de ses couleurs, seul le glyphe importe ici.
func rendu(t *testing.T, e fire.Engine) []string {
	t.Helper()
	var buf bytes.Buffer
	if err := fire.Render(&buf, e); err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(ansi.ReplaceAllString(buf.String(), ""), "\n"), "\n")
}

func TestRenderGlyphes(t *testing.T) {
	m := fire.NewMap(4, 2)
	m.Terrain = []fire.Terrain{
		fire.Water, fire.Plain, fire.Forest, fire.Plain,
		fire.Plain, fire.Plain, fire.Water, fire.Forest,
	}
	m.SetWind(3, 0, fire.East)
	m.Ignite(1, 1)
	e := naive.New(m)

	if got, want := rendu(t, e), []string{"~,#→", ",@~#"}; !equal(got, want) {
		t.Fatalf("au départ :\n%q\nattendu :\n%q", got, want)
	}

	e.Step()
	// La plaine s'est consumée : cendres. Ses voisins combustibles brûlent.
	if got, want := rendu(t, e), []string{"~@@→", "@%~#"}; !equal(got, want) {
		t.Fatalf("après un tour :\n%q\nattendu :\n%q", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRunAfficheEtatInitialPuisChaqueTour(t *testing.T) {
	cfg := fire.DefaultConfig(16, 8, 42)
	cfg.Fires = 4
	m := fire.Generate(cfg)

	var buf bytes.Buffer
	res := fire.Run(naive.New(m), fire.Options{Turns: 3, Render: &buf})

	images := strings.Count(buf.String(), "\n") / m.Height
	if want := res.Turns + 1; images != want {
		t.Fatalf("%d images dessinées, attendu %d (l'état initial, puis un par tour)", images, want)
	}
}
