package loader

import (
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadBlocksFile(t *testing.T) {
    path := filepath.Join("testdata", "blocks", "minecraft", "stone.json")
    m, err := LoadBlocksFile(path)
    require.NoError(t, err)
    require.Len(t, m, 1)

    key := StateKey{BlockID: "minecraft:stone", PropsKey: ""}
    info, ok := m[key]
    require.True(t, ok, "missing expected key: %+v", key)

    assert.False(t, info.Air)
    assert.True(t, info.Opaque)
    assert.True(t, info.SolidBlock)

    assert.Equal(t, 1.5, info.Hardness)
    assert.Equal(t, 6.0, info.Resistance)
    assert.Equal(t, 64, info.StackSize)
    assert.True(t, info.Diggable)
    assert.Equal(t, []string{"mineable/pickaxe"}, info.Material)
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

