package mcgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// LoadBlocksFile loads a single blocks.json and returns a map keyed by StateKey.
func LoadBlocksFile(path string) (map[StateKey]ShapeInfo, error) {
	out := make(map[StateKey]ShapeInfo)
	data, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("read %s: %w", path, err)
	}

	// Per-block file format
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

// MergeBlocksMaps merges multiple version maps, preferring later entries
// in the slice when keys collide.
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

	fmt.Printf("loading blocks from %s\n", root)
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
