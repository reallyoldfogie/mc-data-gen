package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadItemFile loads a single per-item JSON file and returns an ItemInfo.
func LoadItemFile(path string) (ItemInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ItemInfo{}, fmt.Errorf("read %s: %w", path, err)
	}

	var file ItemFile
	if err := json.Unmarshal(data, &file); err != nil {
		return ItemInfo{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}

	return ItemInfo{
		ID:             file.ItemID,
		MaxStackSize:   file.Data.MaxStackSize,
		TranslationKey: file.Data.TranslationKey,
		Rarity:         file.Data.Rarity,
		Fireproof:      file.Data.Fireproof,
		UseAnimation:   file.Data.UseAnimation,
		Tags:           append([]string(nil), file.Data.Tags...),
		Components:     file.Data.Components,
		IsWeapon:       file.Data.IsWeapon,
		IsFood:         file.Data.IsFood,
	}, nil
}

// MergeItemsMaps merges multiple item maps, preferring later entries when keys collide.
func MergeItemsMaps(maps ...map[string]ItemInfo) map[string]ItemInfo {
	out := make(map[string]ItemInfo)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// LoadItemsDir scans a directory tree of per-item JSON files
// (grouped by namespace) and returns a map keyed by item ID.
func LoadItemsDir(root string) (map[string]ItemInfo, error) {
	out := make(map[string]ItemInfo)

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

		info, err := LoadItemFile(path)
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
