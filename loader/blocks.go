package loader

import (
    "sort"
    "strings"
)

// Box represents a single AABB in block-local coordinates (0..1).
type Box struct {
    Min [3]float64 `json:"min"`
    Max [3]float64 `json:"max"`
}

// BlockStatesFile is the per-block file format.
type BlockStatesFile struct {
    BlockID string                 `json:"block_id"`
    States  []BlockStateRecordSlim `json:"states"`
}

// BlockStateRecord mirrors a single entry from blocks.json
// emitted by the Fabric collision exporter.
type BlockStateRecord struct {
    BlockID        string            `json:"block_id"`
    Properties     map[string]string `json:"properties"`
    CollisionBoxes []Box             `json:"collision_boxes"`
    OutlineBoxes   []Box             `json:"outline_boxes"`
    Air            bool              `json:"air"`
    Opaque         bool              `json:"opaque"`
    SolidBlock     bool              `json:"solid_block"`
    Replaceable    bool              `json:"replaceable"`
    BlocksMovement bool              `json:"blocks_movement"`
}

// BlockStateRecordSlim is used in per-block files (no BlockID).
type BlockStateRecordSlim struct {
    Properties     map[string]string `json:"properties"`
    CollisionBoxes []Box             `json:"collision_boxes"`
    OutlineBoxes   []Box             `json:"outline_boxes"`
    Air            bool              `json:"air"`
    Opaque         bool              `json:"opaque"`
    SolidBlock     bool              `json:"solid_block"`
    Replaceable    bool              `json:"replaceable"`
    BlocksMovement bool              `json:"blocks_movement"`
}

// StateKey uniquely identifies a blockstate: block ID + normalized properties.
type StateKey struct {
    BlockID  string
    PropsKey string
}

// ShapeInfo is what you actually use at runtime in your RL env.
type ShapeInfo struct {
    Collision      []Box
    Outline        []Box
    Air            bool
    Opaque         bool
    SolidBlock     bool
    Replaceable    bool
    BlocksMovement bool
}

// MakePropsKey deterministically encodes properties as "k1=v1,k2=v2".
func MakePropsKey(props map[string]string) string {
    if len(props) == 0 {
        return ""
    }
    keys := make([]string, 0, len(props))
    for k := range props {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    parts := make([]string, 0, len(keys))
    for _, k := range keys {
        parts = append(parts, k+"="+props[k])
    }
    return strings.Join(parts, ",")
}

