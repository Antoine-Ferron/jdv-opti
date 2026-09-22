package naive

import (
	"testing"

	"gol-wildfire/internal/fire"
	"gol-wildfire/internal/firetest"
)

func TestConformite(t *testing.T) {
	firetest.Run(t, func(m fire.Map) fire.Engine { return New(m) })
}
