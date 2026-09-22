// Package fire définit le contrat commun à toutes les implémentations de la
// simulation d'incendie, et la carte sur laquelle elles travaillent.
//
// Les règles sont spécifiées dans REGLES.md, au même endroit. Ce document fait
// foi : la suite internal/firetest le traduit en tests, et toute implémentation
// doit la passer avant d'être mesurée.
package fire

// Terrain est la nature immuable d'une case.
type Terrain uint8

const (
	Water Terrain = iota
	Plain
	Forest
)

// Combustion renvoie la durée de combustion du terrain, en tours. Une durée
// nulle signale un terrain incombustible (§2).
func (t Terrain) Combustion() uint8 { return combustion[t] }

// Repos renvoie le nombre de tours pendant lesquels une case de ce terrain ne
// peut pas rebrûler après s'être consumée (§2).
func (t Terrain) Repos() uint8 { return repos[t] }

// Durées en tours, par terrain (§2).
var (
	combustion = [3]uint8{Water: 0, Plain: 1, Forest: 2}
	repos      = [3]uint8{Water: 0, Plain: 2, Forest: 3}
)

// Direction est l'une des 8 directions du vent, numérotées dans l'ordre
// trigonométrique : l'opposé de d est donc (d+4)%8 (§5). y croît vers le bas,
// comme à l'affichage.
type Direction uint8

const (
	East Direction = iota
	NorthEast
	North
	NorthWest
	West
	SouthWest
	South
	SouthEast
)

// Offsets donne le déplacement (dx, dy) de chaque direction.
var Offsets = [8][2]int{
	{1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}, {0, 1}, {1, 1},
}

// Map est une carte : le décor immuable plus les foyers de départ. Elle est
// engendrée hors du périmètre chronométré et partagée telle quelle par toutes
// les implémentations, pour qu'elles partent exactement du même état.
type Map struct {
	Width, Height int
	Terrain       []Terrain // ligne par ligne : Terrain[y*Width+x]
	Wind          []uint8   // 0 = pas de vent ; sinon 1+Direction
	Fires         []int     // indices des foyers initiaux
}

// NewMap alloue une carte, sans vent ni foyer. Le terrain vaut Water partout :
// c'est la valeur zéro de Terrain, aux appelants de le remplir.
func NewMap(w, h int) Map {
	return Map{Width: w, Height: h, Terrain: make([]Terrain, w*h), Wind: make([]uint8, w*h)}
}

// At renvoie l'indice de (x, y) sur le tore.
func (m Map) At(x, y int) int { return Mod(y, m.Height)*m.Width + Mod(x, m.Width) }

// SetWind pose un vent de direction d sur la case (x, y).
func (m Map) SetWind(x, y int, d Direction) { m.Wind[m.At(x, y)] = 1 + uint8(d) }

// WindAt renvoie la direction du vent de la case (x, y), et false s'il n'y en a pas.
func (m Map) WindAt(x, y int) (Direction, bool) {
	w := m.Wind[m.At(x, y)]
	return Direction(w - 1), w > 0
}

// Ignite ajoute un foyer de départ sur la case (x, y).
func (m *Map) Ignite(x, y int) { m.Fires = append(m.Fires, m.At(x, y)) }

// Mod est le modulo positif : Go rend -1 % 5 = -1, le tore veut 4.
func Mod(a, n int) int {
	a %= n
	if a < 0 {
		a += n
	}
	return a
}

// Engine est une simulation d'incendie en cours. Le terrain et le vent restent
// dans la Map : seul l'état de combustion évolue, et c'est tout ce que le
// contrat expose.
type Engine interface {
	// Step calcule le tour suivant.
	Step()
	// Fire renvoie le nombre de tours de combustion restants de (x, y), 0 si elle n'est pas en feu.
	Fire(x, y int) uint8
	// Rest renvoie le nombre de tours de repos restants de (x, y), 0 si elle est disponible.
	Rest(x, y int) uint8
	// Burning renvoie le nombre de cases actuellement en feu.
	Burning() int
	// Fingerprint renvoie une empreinte de l'état courant, pour la détection de cycles.
	Fingerprint() uint64
	Width() int
	Height() int
}

// Factory construit un Engine à partir d'une carte.
type Factory func(m Map) Engine

// Options pilote Run.
type Options struct {
	Turns int // nombre maximal de tours
}

// Result résume une exécution.
type Result struct {
	Turns   int  // tours effectivement calculés
	Extinct bool // vrai si l'incendie s'est éteint de lui-même
}

// Run déroule la simulation jusqu'à extinction ou épuisement des tours (§6).
func Run(e Engine, opt Options) Result {
	var res Result
	for res.Turns < opt.Turns {
		e.Step()
		res.Turns++
		if e.Burning() == 0 {
			res.Extinct = true
			break
		}
	}
	return res
}
