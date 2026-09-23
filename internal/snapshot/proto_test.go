package snapshot_test

import (
	"testing"

	"gol-wildfire/internal/snapshot"
	"gol-wildfire/internal/snapshot/snapshottest"
)

func TestProtoConformite(t *testing.T) {
	snapshottest.Run(t, snapshot.Proto{})
}
