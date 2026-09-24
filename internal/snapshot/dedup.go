package snapshot

// Dedup évite de réécrire un état déjà enregistré (étape I/O-5, hypothèse
// I/O-5 B).
//
// Une empreinte identifie un état de grille. Quand le feu s'éteint, l'état
// devient un point fixe et tous les snapshots suivants sont rigoureusement les
// mêmes : les réécrire consomme de la sérialisation et du disque pour ne rien
// ajouter à l'archive.
//
// La taille du cache est un paramètre parce qu'elle est précisément la question
// ouverte. Un point fixe est un cycle de période 1, capté par une seule entrée ;
// un cache plus grand ne sert que s'il existe des cycles plus longs. La valeur
// à retenir est celle que la mesure justifie, pas celle qui paraît la plus
// générale — voir BenchmarkSnapshotSerie.
//
// Le stockage est un anneau parcouru linéairement. Pour les tailles en jeu
// (une à quelques dizaines d'entrées), un parcours de tranche contiguë bat une
// table de hachage, et surtout il conserve l'ordre d'éviction sans structure
// annexe.
type Dedup struct {
	vues  []uint64
	i     int // prochaine case à écraser
	plein bool

	Vus      int // empreintes présentées
	Doublons int // empreintes déjà connues, donc snapshots évités
}

// NewDedup crée un cache des taille dernières empreintes. Une taille nulle ou
// négative donne un cache inactif, qui ne retient jamais rien : c'est la
// référence à laquelle comparer les autres.
func NewDedup(taille int) *Dedup {
	if taille < 0 {
		taille = 0
	}
	return &Dedup{vues: make([]uint64, taille)}
}

// DejaVu enregistre une empreinte et indique si elle était déjà connue.
// Lorsqu'elle renvoie true, le snapshot correspondant peut être omis.
func (d *Dedup) DejaVu(empreinte uint64) bool {
	d.Vus++
	if len(d.vues) == 0 {
		return false
	}

	borne := d.i
	if d.plein {
		borne = len(d.vues)
	}
	for j := 0; j < borne; j++ {
		if d.vues[j] == empreinte {
			d.Doublons++
			return true
		}
	}

	d.vues[d.i] = empreinte
	d.i++
	if d.i == len(d.vues) {
		d.i, d.plein = 0, true
	}
	return false
}

// Empreinte résume l'état mutable d'une grille : seuls le feu et le repos
// changent d'un tour à l'autre, le décor est immuable pour toute la partie.
//
// Volontairement indépendante de fire.Engine.Fingerprint, qui coûte environ
// 20 ms en 1024² — quinze fois le prix du snapshot qu'elle servirait à éviter.
// Une déduplication ne vaut que si son test coûte nettement moins cher que le
// travail qu'elle supprime ; c'est la condition que la mesure doit vérifier.
//
// FNV-1a octet par octet : deux multiplications par case. Un hachage par mots
// de 64 bits irait plus vite, au prix d'un code sensible à l'alignement et à
// l'endianness ; s'il apparaît que ce calcul pèse, c'est la première piste.
func (s *State) Empreinte() uint64 {
	const (
		amorce = 14695981039346656037
		nombre = 1099511628211
	)
	h := uint64(amorce)
	for i := range s.Fire {
		h = (h ^ uint64(s.Fire[i])) * nombre
		h = (h ^ uint64(s.Rest[i])) * nombre
	}
	return h
}

// Taux rend la proportion de snapshots évités, entre 0 et 1.
func (d *Dedup) Taux() float64 {
	if d.Vus == 0 {
		return 0
	}
	return float64(d.Doublons) / float64(d.Vus)
}
