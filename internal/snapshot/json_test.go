package snapshot_test

import (
	"testing"

	"gol-wildfire/internal/snapshot"
	"gol-wildfire/internal/snapshot/snapshottest"
)

func TestJSONConformite(t *testing.T) {
	snapshottest.Run(t, snapshot.JSON{})
}
