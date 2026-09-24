package snapshot

import (
	"bytes"
	"sync"
	"testing"

	"gol-wildfire/internal/fire"
)

// videLesPools remet les deux pots à neuf, pour qu'un test parte d'un état
// connu. Sans cela, un tampon laissé par un test précédent rendrait le
// scénario « premier appel » impossible à reproduire.
func videLesPools() {
	octets = sync.Pool{}
	mots = sync.Pool{}
}

// etatSimple fabrique un état dont chaque case porte une valeur distincte, pour
// que d'éventuels octets rémanents se voient.
func etatSimple(w, h int, motif byte) *State {
	n := w * h
	s := &State{Width: w, Height: h, Turn: int(motif)}
	s.Terrain = make([]fire.Terrain, n)
	s.Wind = make([]uint8, n)
	s.Fire = make([]uint8, n)
	s.Rest = make([]uint8, n)
	for i := 0; i < n; i++ {
		s.Terrain[i] = fire.Terrain((int(motif) + i) % 3)
		s.Wind[i] = uint8((int(motif) + i) % 9)
		s.Fire[i] = uint8((int(motif) + i) % 3)
		s.Rest[i] = uint8((int(motif) + i) % 4)
	}
	return s
}

// Le risque propre à cette étape : un tampon repris porte les octets du
// snapshot précédent. Si une seule case n'est pas réécrite, le fichier produit
// contient un fragment de l'état d'avant — corruption silencieuse, qu'aucun
// test de round-trip ne détecte forcément puisque la relecture reste cohérente.
//
// Le test compare donc la sortie obtenue avec un pool vierge à celle obtenue
// avec un pool contenant un tampon plus grand et rempli d'autre chose. Toute
// différence est une fuite.
func TestPackedSansResidu(t *testing.T) {
	verifieSansResidu(t, func() Format { return Packed{} })
}

func TestPackedSansDecorSansResidu(t *testing.T) {
	verifieSansResidu(t, func() Format { return Packed{SansDecor: true} })
}

func TestProtoSansResidu(t *testing.T) {
	verifieSansResidu(t, func() Format { return Proto{} })
}

func verifieSansResidu(t *testing.T, fabrique func() Format) {
	t.Helper()

	// Dimensions impaires : le dernier octet de l'état n'accueille qu'une case,
	// c'est là que la rémanence se logerait.
	const w, h = 31, 17
	petit := etatSimple(w, h, 1)

	videLesPools()
	var vierge bytes.Buffer
	if err := fabrique().Write(&vierge, petit); err != nil {
		t.Fatal(err)
	}

	// Un premier passage sur un état plus grand et de contenu différent remplit
	// les pots de tampons sales et surdimensionnés.
	videLesPools()
	grand := etatSimple(w*3, h*3, 200)
	if err := fabrique().Write(&bytes.Buffer{}, grand); err != nil {
		t.Fatal(err)
	}
	var recycle bytes.Buffer
	if err := fabrique().Write(&recycle, petit); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(vierge.Bytes(), recycle.Bytes()) {
		t.Fatalf("tampon recyclé : %d octets écrits contre %d avec un pool vierge — "+
			"des octets du snapshot précédent ont fuité",
			recycle.Len(), vierge.Len())
	}
}

func TestDedup(t *testing.T) {
	t.Run("taille nulle ne retient rien", func(t *testing.T) {
		d := NewDedup(0)
		for i := 0; i < 5; i++ {
			if d.DejaVu(42) {
				t.Fatal("un cache inactif ne doit jamais signaler de doublon")
			}
		}
		if d.Vus != 5 || d.Doublons != 0 {
			t.Fatalf("Vus=%d Doublons=%d, attendu 5 et 0", d.Vus, d.Doublons)
		}
	})

	t.Run("une entrée capte un point fixe", func(t *testing.T) {
		d := NewDedup(1)
		suite := []uint64{7, 7, 7, 7}
		doublons := 0
		for _, f := range suite {
			if d.DejaVu(f) {
				doublons++
			}
		}
		if doublons != 3 {
			t.Fatalf("%d doublons sur un point fixe, attendu 3", doublons)
		}
		if got := d.Taux(); got != 0.75 {
			t.Fatalf("taux %v, attendu 0.75", got)
		}
	})

	// La limite d'un cache à une entrée, et la seule raison d'en vouloir un
	// plus grand : une alternance ne se répète jamais à l'entrée précédente.
	t.Run("une entrée rate un cycle de période 2", func(t *testing.T) {
		d := NewDedup(1)
		for _, f := range []uint64{1, 2, 1, 2, 1, 2} {
			d.DejaVu(f)
		}
		if d.Doublons != 0 {
			t.Fatalf("%d doublons, attendu 0 : une seule entrée ne peut pas voir l'alternance", d.Doublons)
		}
	})

	t.Run("deux entrées captent un cycle de période 2", func(t *testing.T) {
		d := NewDedup(2)
		for _, f := range []uint64{1, 2, 1, 2, 1, 2} {
			d.DejaVu(f)
		}
		if d.Doublons != 4 {
			t.Fatalf("%d doublons, attendu 4", d.Doublons)
		}
	})

	t.Run("l'anneau évince la plus ancienne", func(t *testing.T) {
		d := NewDedup(2)
		for _, f := range []uint64{1, 2, 3} { // 1 est évincée par 3
			d.DejaVu(f)
		}
		if d.DejaVu(1) {
			t.Fatal("l'empreinte 1 aurait dû être évincée du cache")
		}
	})
}
