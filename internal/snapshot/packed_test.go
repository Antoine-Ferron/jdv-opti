package snapshot_test

import (
	"bytes"
	"testing"

	"gol-wildfire/internal/snapshot"
	"gol-wildfire/internal/snapshot/snapshottest"
)

func TestPackedConformite(t *testing.T) {
	snapshottest.Run(t, snapshot.Packed{})
}

// Le mode sans décor sert aux snapshots qui suivent le premier d'une série :
// l'état doit être identique, seul le décor manque à la relecture.
func TestPackedSansDecor(t *testing.T) {
	s := snapshottest.Etat(48, 32, 7, 9)

	var avec, sans bytes.Buffer
	if err := (snapshot.Packed{}).Write(&avec, s); err != nil {
		t.Fatal(err)
	}
	if err := (snapshot.Packed{SansDecor: true}).Write(&sans, s); err != nil {
		t.Fatal(err)
	}
	if gagne := avec.Len() - sans.Len(); gagne != s.Width*s.Height {
		t.Errorf("omettre le décor économise %d octets, attendu %d", gagne, s.Width*s.Height)
	}

	relu, err := (snapshot.Packed{}).Read(&sans)
	if err != nil {
		t.Fatal(err)
	}
	for i := range s.Fire {
		if relu.Fire[i] != s.Fire[i] || relu.Rest[i] != s.Rest[i] {
			t.Fatalf("case %d : l'état n'a pas survécu à l'omission du décor", i)
		}
	}
}
