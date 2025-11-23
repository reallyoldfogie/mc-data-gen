package loader

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// LoadBlocksFile loads a single per-block JSON file and returns a map keyed by StateKey.
func LoadBlocksFile(path string) (map[StateKey]ShapeInfo, error) {
    out := make(map[StateKey]ShapeInfo)
    data, err := os.ReadFile(path)
    if err != nil {
        return out, fmt.Errorf("read %s: %w", path, err)
    }

    var file struct {
        BlockID string                 `json:"block_id"`
        States  []BlockStateRecordSlim `json:"states"`
    }
    if err := json.Unmarshal(data, &file); err != nil {
        return out, fmt.Errorf("unmarshal %s: %w", path, err)
    }

    for _, s := range file.States {
        key := StateKey{
            BlockID:  file.BlockID,
            PropsKey: MakePropsKey(s.Properties),
        }
        out[key] = ShapeInfo{
            Collision:      append([]Box(nil), s.CollisionBoxes...),
            Outline:        append([]Box(nil), s.OutlineBoxes...),
            Air:            s.Air,
            Opaque:         s.Opaque,
            SolidBlock:     s.SolidBlock,
            Replaceable:    s.Replaceable,
            BlocksMovement: s.BlocksMovement,
        }
    }

    return out, nil
}

// MergeBlocksMaps merges multiple version maps, preferring later entries when keys collide.
func MergeBlocksMaps(maps ...map[StateKey]ShapeInfo) map[StateKey]ShapeInfo {
    out := make(map[StateKey]ShapeInfo)
    for _, m := range maps {
        for k, v := range m {
            out[k] = v
        }
    }
    return out
}

// LoadBlocksDir scans a directory tree of per-block JSON files
// (grouped by namespace) and returns the same map[StateKey]ShapeInfo.
func LoadBlocksDir(root string) (map[StateKey]ShapeInfo, error) {
    out := make(map[StateKey]ShapeInfo)

    err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }
        if !strings.HasSuffix(d.Name(), ".json") {
            return nil
        }
        tmpOut, err := LoadBlocksFile(path)
        if err != nil {
            return fmt.Errorf("failed to load file %s: %s", path, err.Error())
        }
        out = MergeBlocksMaps(out, tmpOut)
        return nil
    })
    if err != nil {
        return nil, err
    }
    return out, nil
}

