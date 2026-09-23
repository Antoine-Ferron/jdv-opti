package fire

import "io"

// Codes ANSI de couleur, compris par Windows Terminal comme par les terminaux
// Unix.
const (
	ansiReset = "\033[0m"
	ansiFeu   = "\033[91m"
	ansiCendr = "\033[90m"
	ansiVent  = "\033[96m"
	ansiEau   = "\033[94m"
	ansiForet = "\033[32m"
	ansiPlain = "\033[93m"
)

// fleches donne le glyphe de chaque direction, dans l'ordre de Offsets.
var fleches = [8]string{"→", "↗", "↑", "↖", "←", "↙", "↓", "↘"}

// Render dessine la carte et son état de combustion. Réservé aux petites
// cartes : une image coûte bien plus cher qu'un tour de simulation.
//
// Le feu prime sur les cendres, qui priment sur le vent, qui prime sur le
// terrain : on veut voir d'abord ce qui bouge.
func Render(out io.Writer, e Engine) error {
	m := e.Map()
	buf := make([]byte, 0, len(m.Terrain)*7+m.Height)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			switch {
			case e.Fire(x, y) > 0:
				buf = append(buf, ansiFeu+"@"...)
			case e.Rest(x, y) > 0:
				buf = append(buf, ansiCendr+"%"...)
			case m.Wind[i] > 0:
				buf = append(buf, ansiVent+fleches[m.Wind[i]-1]...)
			case m.Terrain[i] == Water:
				buf = append(buf, ansiEau+"~"...)
			case m.Terrain[i] == Forest:
				buf = append(buf, ansiForet+"#"...)
			default:
				buf = append(buf, ansiPlain+","...)
			}
		}
		buf = append(buf, '\n')
	}
	buf = append(buf, ansiReset...)
	_, err := out.Write(buf)
	return err
}
