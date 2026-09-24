package snapshot

import "sync"

// Recyclage des tampons de sérialisation (étape I/O-5, hypothèse I/O-5 A).
//
// Les tampons visés sont volumineux, de taille constante pour une carte donnée,
// et rendus dès l'écriture terminée : le cas d'usage pour lequel sync.Pool
// existe. Ce qu'on évite n'est pas le compte d'allocations — qui n'intéresse
// personne en soi — mais la remise à zéro de plusieurs mégaoctets par snapshot,
// Go garantissant une mémoire nulle à l'allocation, et le travail que ces
// déchets imposent au ramasse-miettes.
//
// Les pools stockent des *pointeurs* vers tranches : ranger une tranche
// directement dans une interface vide alloue, ce qui annulerait une part du
// gain recherché.
//
// ATTENTION — un tampon repris n'est pas vierge. Il porte les octets du
// snapshot précédent. Tout code qui en prend un doit écrire chacune de ses
// cases avant de le transmettre, sans quoi des fragments de l'état précédent
// fuiteraient dans le fichier. TestPackedSansResidu et TestProtoSansResidu
// vérifient précisément cela.
var (
	octets sync.Pool // *[]byte
	mots   sync.Pool // *[]uint32, pour l'API Protobuf
)

// poolActif permet de désactiver le recyclage le temps d'une mesure.
//
// C'est de l'échafaudage de mesure assumé, et non une option de configuration :
// il n'existe que pour que BenchmarkPool compare les deux comportements dans un
// même fichier de résultats, au lieu de confronter deux campagnes séparées dont
// l'état machine diffère. Le coût est une lecture de variable globale et une
// branche parfaitement prédite, négligeable devant les mégaoctets recopiés.
var poolActif = true

// prendreOctets rend un tampon d'au moins n octets, de longueur exactement n.
func prendreOctets(n int) []byte {
	if !poolActif {
		return make([]byte, n)
	}
	if v := octets.Get(); v != nil {
		if b := v.(*[]byte); cap(*b) >= n {
			return (*b)[:n]
		}
		// Trop court : on le laisse partir plutôt que de le remettre au pot,
		// où il serait repris indéfiniment pour être rejeté à chaque fois.
	}
	return make([]byte, n)
}

func rendreOctets(b []byte) {
	if poolActif {
		octets.Put(&b)
	}
}

// prendreMots rend une tranche d'au moins n entiers, de longueur exactement n.
func prendreMots(n int) []uint32 {
	if !poolActif {
		return make([]uint32, n)
	}
	if v := mots.Get(); v != nil {
		if m := v.(*[]uint32); cap(*m) >= n {
			return (*m)[:n]
		}
	}
	return make([]uint32, n)
}

func rendreMots(m []uint32) {
	if poolActif {
		mots.Put(&m)
	}
}
