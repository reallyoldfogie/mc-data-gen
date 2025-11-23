package loader

import (
    "path/filepath"
    "testing"
)

func TestLoadBlocksFile(t *testing.T) {
    path := filepath.Join("testdata", "blocks", "minecraft", "stone.json")
    m, err := LoadBlocksFile(path)
    if err != nil {
        t.Fatalf("LoadBlocksFile error: %v", err)
    }
    if len(m) != 1 {
        t.Fatalf("expected 1 state, got %d", len(m))
    }
    key := StateKey{BlockID: "minecraft:stone", PropsKey: ""}
    info, ok := m[key]
    if !ok {
        t.Fatalf("missing expected key: %+v", key)
    }
    if info.Air || !info.Opaque || !info.SolidBlock {
        t.Fatalf("unexpected info: %+v", info)
    }
}

func TestLoadBlocksDir(t *testing.T) {
    root := filepath.Join("testdata", "blocks")
    m, err := LoadBlocksDir(root)
    if err != nil {
        t.Fatalf("LoadBlocksDir error: %v", err)
    }
    wantKeys := []StateKey{
        {BlockID: "minecraft:stone", PropsKey: ""},
        {BlockID: "minecraft:dirt", PropsKey: "variant=coarse"},
    }
    for _, k := range wantKeys {
        if _, ok := m[k]; !ok {
            t.Fatalf("missing key: %+v", k)
        }
    }
}

