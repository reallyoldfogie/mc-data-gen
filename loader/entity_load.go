package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEntityFile loads a single per-entity JSON file and returns an EntityInfo.
func LoadEntityFile(path string) (EntityInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EntityInfo{}, fmt.Errorf("read %s: %w", path, err)
	}

	var file EntityFile
	if err := json.Unmarshal(data, &file); err != nil {
		return EntityInfo{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}

	info := EntityInfo{
		ID:                 file.EntityID,
		SpawnGroup:         file.Data.SpawnGroup,
		FireImmune:         file.Data.FireImmune,
		DefaultDimensions:  file.Data.DefaultDimensions,
		PoseDimensions:     file.Data.PoseDimensions,
		SizeVariants:       append([]EntitySizeVariant(nil), file.Data.SizeVariants...),
		BabyDimensions:     file.Data.BabyDimensions,
		Attributes:         append([]EntityAttribute(nil), file.Data.Attributes...),
		Tags:               append([]string(nil), file.Data.Tags...),
	}
	return info, nil
}

// MergeEntityMaps merges multiple entity maps, preferring later entries when keys collide.
func MergeEntityMaps(maps ...map[string]EntityInfo) map[string]EntityInfo {
	out := make(map[string]EntityInfo)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// LoadEntitiesDir scans a directory tree of per-entity JSON files
// (grouped by namespace) and returns a map keyed by entity ID.
func LoadEntitiesDir(root string) (map[string]EntityInfo, error) {
	out := make(map[string]EntityInfo)

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

		info, err := LoadEntityFile(path)
		if err != nil {
			return fmt.Errorf("failed to load file %s: %w", path, err)
		}
		out[info.ID] = info
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
