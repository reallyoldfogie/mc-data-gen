package loader

import (
	"path/filepath"
	"testing"
)

func TestLoadEntityFile(t *testing.T) {
	path := filepath.Join("testdata", "entities", "minecraft", "zombie.json")
	info, err := LoadEntityFile(path)
	if err != nil {
		t.Fatalf("LoadEntityFile error: %v", err)
	}
	if info.ID != "minecraft:zombie" {
		t.Fatalf("expected ID minecraft:zombie, got %s", info.ID)
	}
	if info.SpawnGroup != "MONSTER" {
		t.Fatalf("expected spawn_group MONSTER, got %s", info.SpawnGroup)
	}
	if info.FireImmune {
		t.Fatalf("expected fire_immune false, got true")
	}
	if info.DefaultDimensions.Width != 0.6 {
		t.Fatalf("expected width 0.6, got %f", info.DefaultDimensions.Width)
	}
	if len(info.PoseDimensions) == 0 {
		t.Fatalf("expected pose_dimensions to be set")
	}
	if len(info.Attributes) == 0 {
		t.Fatalf("expected attributes to be set")
	}
	if len(info.Tags) == 0 {
		t.Fatalf("expected tags to be set")
	}
}

func TestLoadEntitiesDir(t *testing.T) {
	root := filepath.Join("testdata", "entities")
	m, err := LoadEntitiesDir(root)
	if err != nil {
		t.Fatalf("LoadEntitiesDir error: %v", err)
	}
	wantEntities := []string{
		"minecraft:zombie",
		"minecraft:slime",
	}
	for _, id := range wantEntities {
		if _, ok := m[id]; !ok {
			t.Fatalf("missing entity: %s", id)
		}
	}
	if len(m) != len(wantEntities) {
		t.Fatalf("expected %d entities, got %d", len(wantEntities), len(m))
	}
}
