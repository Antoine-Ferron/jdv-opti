package snapshot

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/snapshot/pb"
)

// Proto sérialise via Protobuf, le format binaire que le sujet met en avant.
//
// Il corrige [S1] — plus de nom de champ par case, le wire format n'écrit que
// des numéros de champ — et [S3] — plus d'encodage texte. Il conserve [S2] : le
// décor est réécrit à chaque snapshot.
//
// Deux limites lui restent propres, et ce sont elles que le format bit-packé
// lèvera :
//
//	[S5] Le plus petit varint occupe un octet, alors qu'une valeur d'état tient
//	     sur deux bits : le format ne peut pas descendre sous 4 octets par case.
//	[S6] L'API travaille sur des []uint32 ; convertir les quatre grilles depuis
//	     des []uint8 alloue quatre fois la taille de la carte avant d'écrire.
type Proto struct{}

func init() { Register("proto", Proto{}) }

func (Proto) Ext() string { return "pb" }

func (Proto) Write(w io.Writer, s *State) error {
	msg := &pb.Snapshot{
		Turn:    uint32(s.Turn),
		Width:   uint32(s.Width),
		Height:  uint32(s.Height),
		Terrain: make([]uint32, len(s.Fire)), // [S6]
		Wind:    make([]uint32, len(s.Fire)),
		Fire:    make([]uint32, len(s.Fire)),
		Rest:    make([]uint32, len(s.Fire)),
	}
	for i := range s.Fire {
		msg.Terrain[i] = uint32(s.Terrain[i])
		msg.Wind[i] = uint32(s.Wind[i])
		msg.Fire[i] = uint32(s.Fire[i])
		msg.Rest[i] = uint32(s.Rest[i])
	}
	buf, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

func (Proto) Read(r io.Reader) (*State, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var msg pb.Snapshot
	if err := proto.Unmarshal(buf, &msg); err != nil {
		return nil, err
	}

	n := int(msg.Width) * int(msg.Height)
	if n == 0 || len(msg.Fire) != n || len(msg.Rest) != n ||
		len(msg.Terrain) != n || len(msg.Wind) != n {
		return nil, fmt.Errorf("snapshot: grille %dx%d incohérente avec %d cases lues",
			msg.Width, msg.Height, len(msg.Fire))
	}

	s := &State{
		Turn: int(msg.Turn), Width: int(msg.Width), Height: int(msg.Height),
		Terrain: make([]fire.Terrain, n),
		Wind:    make([]uint8, n),
		Fire:    make([]uint8, n),
		Rest:    make([]uint8, n),
	}
	for i := 0; i < n; i++ {
		s.Terrain[i] = fire.Terrain(msg.Terrain[i])
		s.Wind[i] = uint8(msg.Wind[i])
		s.Fire[i] = uint8(msg.Fire[i])
		s.Rest[i] = uint8(msg.Rest[i])
	}
	return s, nil
}
