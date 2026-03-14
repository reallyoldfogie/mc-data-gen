package loader

import (
	"path/filepath"
	"testing"
)

func TestLoadItemFile(t *testing.T) {
	path := filepath.Join("testdata", "items", "minecraft", "iron_sword.json")
	info, err := LoadItemFile(path)
	if err != nil {
		t.Fatalf("LoadItemFile error: %v", err)
	}
	if info.ID != "minecraft:iron_sword" {
		t.Fatalf("expected ID minecraft:iron_sword, got %s", info.ID)
	}
	if info.MaxStackSize != 1 {
		t.Fatalf("expected max_stack_size 1, got %d", info.MaxStackSize)
	}
	if !info.IsWeapon {
		t.Fatalf("expected is_weapon true, got false")
	}
	if info.IsFood {
		t.Fatalf("expected is_food false, got true")
	}
}

func TestLoadItemsDir(t *testing.T) {
	root := filepath.Join("testdata", "items")
	m, err := LoadItemsDir(root)
	if err != nil {
		t.Fatalf("LoadItemsDir error: %v", err)
	}
	wantItems := []string{
		"minecraft:iron_sword",
		"minecraft:apple",
	}
	for _, id := range wantItems {
		if _, ok := m[id]; !ok {
			t.Fatalf("missing item: %s", id)
		}
	}
	if len(m) != len(wantItems) {
		t.Fatalf("expected %d items, got %d", len(wantItems), len(m))
	}
}
