package mcgen

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	loader "github.com/reallyoldfogie/mc-data-gen/loader"
)

// PrepareProject copies the Fabric template into a per-version dir and
// injects minecraft_version, yarn_mappings, loader_version, and
// fabric_api_version into gradle.properties.
func PrepareProject(templateDir, projectDir string, meta *FabricMeta) error {
	// Always start fresh for this version
	if _, err := os.Stat(projectDir); err == nil {
		if err := os.RemoveAll(projectDir); err != nil {
			return fmt.Errorf("remove old project dir: %w", err)
		}
	}

	if err := CopyDir(templateDir, projectDir); err != nil {
		return fmt.Errorf("copy template: %w", err)
	}

	gradlePropsPath := filepath.Join(projectDir, "gradle.properties")

	props, err := os.ReadFile(gradlePropsPath)
	if err != nil {
		return fmt.Errorf("read gradle.properties: %w", err)
	}

	lines := strings.Split(string(props), "\n")
	foundMC := false
	foundYarn := false
	foundLoader := false
	foundAPI := false
	foundLoom := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "minecraft_version="):
			lines[i] = "minecraft_version=" + meta.MinecraftVersion
			foundMC = true
		case strings.HasPrefix(trimmed, "yarn_mappings="):
			// Only set if we have a Yarn version (< 26.1)
			if meta.YarnVersion != "" {
				lines[i] = "yarn_mappings=" + meta.YarnVersion
				foundYarn = true
			} else {
				// Remove the line for versions that don't need Yarn
				lines[i] = ""
			}
		case strings.HasPrefix(trimmed, "loader_version="):
			lines[i] = "loader_version=" + meta.LoaderVersion
			foundLoader = true
		case strings.HasPrefix(trimmed, "fabric_api_version="):
			lines[i] = "fabric_api_version=" + meta.FabricAPIVersion
			foundAPI = true
		case strings.HasPrefix(trimmed, "loom_version="):
			lines[i] = "loom_version=" + meta.LoomVersion
			foundLoom = true
		}
	}

	if !foundMC {
		lines = append(lines, "minecraft_version="+meta.MinecraftVersion)
	}
	// Only add yarn_mappings if version needs it (< 26.1)
	if meta.YarnVersion != "" && !foundYarn {
		lines = append(lines, "yarn_mappings="+meta.YarnVersion)
	}
	if meta.LoaderVersion != "" && !foundLoader {
		lines = append(lines, "loader_version="+meta.LoaderVersion)
	}
	if meta.FabricAPIVersion != "" && !foundAPI {
		lines = append(lines, "fabric_api_version="+meta.FabricAPIVersion)
	}
	if meta.LoomVersion != "" && !foundLoom {
		lines = append(lines, "loom_version="+meta.LoomVersion)
	}

	out := strings.Join(lines, "\n")
	if err := os.WriteFile(gradlePropsPath, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write gradle.properties: %w", err)
	}

	// Update Loom version in build.gradle
	if err := updateLoomVersion(projectDir, meta.LoomVersion); err != nil {
		return fmt.Errorf("update loom version: %w", err)
	}

	return nil
}

// updateLoomVersion replaces the Loom version in build.gradle
func updateLoomVersion(projectDir, loomVersion string) error {
	buildGradlePath := filepath.Join(projectDir, "build.gradle")
	
	data, err := os.ReadFile(buildGradlePath)
	if err != nil {
		return fmt.Errorf("read build.gradle: %w", err)
	}
	
	// Replace the Loom version line
	content := string(data)
	content = strings.ReplaceAll(content,
		`id "fabric-loom" version "1.11-SNAPSHOT"`,
		`id "fabric-loom" version "`+loomVersion+`"`)
	content = strings.ReplaceAll(content,
		`id "fabric-loom" version "1.14-SNAPSHOT"`,
		`id "fabric-loom" version "`+loomVersion+`"`)
	
	if err := os.WriteFile(buildGradlePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write build.gradle: %w", err)
	}
	
	return nil
}

// RunGradleWithArgs runs ./gradlew <args...> in the given projectDir.
func RunGradleWithArgs(projectDir string, args ...string) error {
	gradlew := "./gradlew"
	if _, err := os.Stat(filepath.Join(projectDir, "gradlew")); err != nil {
		return fmt.Errorf("gradlew not found in %s: %w", projectDir, err)
	}

	cmd := exec.Command(gradlew, append(args, "--no-daemon")...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gradle %v failed: %w", args, err)
	}
	return nil
}

// RunGradle runs ./gradlew <task> in the given projectDir.
func RunGradle(projectDir, task string) error {
	return RunGradleWithArgs(projectDir, task)
}

// CollectOutput copies the generated JSON from the project into outputRoot/version/.
func CollectOutput(projectDir, generatorOutputRel, outputRoot, version string) error {
	src := filepath.Join(projectDir, generatorOutputRel)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("generator output not found at %s: %w", src, err)
	}

	if err := collectBlocks(src, outputRoot, version); err != nil {
		return fmt.Errorf("collectBlocks: %w", err)
	}

	if err := collectItems(src, outputRoot, version); err != nil {
		return fmt.Errorf("collectItems: %w", err)
	}

	return nil
}

// DecompileSources runs genSources and extracts decompiled Minecraft sources.
func DecompileSources(projectDir string) error {
	// Run genSources task to decompile Minecraft
	if err := RunGradle(projectDir, "genSources"); err != nil {
		return fmt.Errorf("genSources failed: %w", err)
	}

	// Find the decompiled sources JAR in Loom cache
	// Loom stores sources in .gradle/loom-cache/
	loomCacheDir := filepath.Join(projectDir, ".gradle", "loom-cache")
	sourcesJar, err := findSourcesJar(loomCacheDir)
	if err != nil {
		return fmt.Errorf("find sources jar: %w", err)
	}

	// Extract to extracted_src directory
	extractDir := filepath.Join(projectDir, "extracted_src")
	if err := os.RemoveAll(extractDir); err != nil {
		return fmt.Errorf("remove old extracted_src: %w", err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return fmt.Errorf("create extracted_src: %w", err)
	}

	if err := extractJar(sourcesJar, extractDir); err != nil {
		return fmt.Errorf("extract jar: %w", err)
	}

	return nil
}

// findSourcesJar searches for the decompiled sources JAR in Loom cache.
func findSourcesJar(loomCacheDir string) (string, error) {
	var sourcesJar string

	err := filepath.Walk(loomCacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		
		// Look for -sources.jar files in the cache
		if strings.HasSuffix(path, "-sources.jar") {
			// Prefer minecraft-merged or minecraft-project sources
			if strings.Contains(path, "minecraft") {
				sourcesJar = path
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	if sourcesJar == "" {
		return "", fmt.Errorf("no sources jar found in %s", loomCacheDir)
	}
	return sourcesJar, nil
}

// extractJar extracts a JAR file to the specified directory.
func extractJar(jarPath, destDir string) error {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Only extract .java files
		if !strings.HasSuffix(f.Name, ".java") {
			continue
		}

		destPath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
			continue
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func collectBlocks(src, outputRoot, version string) error {
	blocksSrc := filepath.Join(src, "blocks.json")
	if _, err := os.Stat(blocksSrc); err != nil {
		return fmt.Errorf("generator output (blocks.json) not found at %s: %w", src, err)
	}

	blocksDestDir := filepath.Join(outputRoot, version, "blocks")
	if err := os.MkdirAll(blocksDestDir, 0o755); err != nil {
		return fmt.Errorf("create blocks dest dir: %w", err)
	}

	if err := shardFile(blocksSrc, blocksDestDir); err != nil {
		return fmt.Errorf("shard blocks dest file: %w", err)

	}

	return nil
}

func collectItems(src, outputRoot, version string) error {
	itemsDestDir := filepath.Join(outputRoot, version, "items")
	if err := os.MkdirAll(itemsDestDir, 0o755); err != nil {
		return fmt.Errorf("create item dir: %w", err)
	}

	itemsSrc := filepath.Join(src, "items.json")
	if _, err := os.Stat(itemsSrc); err != nil {
		return fmt.Errorf("generator output (items.json) not found at %s: %w", src, err)
	}
	itemDestFile := filepath.Join(itemsDestDir, "items.json")

	data, err := os.ReadFile(itemsSrc)
	if err != nil {
		return fmt.Errorf("read generator item output: %w", err)
	}

	if err := os.WriteFile(itemDestFile, data, 0o644); err != nil {
		return fmt.Errorf("write dest item file: %w", err)
	}
	return nil
}

func shardFile(inputPath, outRoot string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}

	var records []loader.BlockStateRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("unmarshal %s: %w", inputPath, err)
	}

	// Group by block_id
	byBlock := make(map[string][]loader.BlockStateRecordSlim)
	for _, r := range records {
		slim := loader.BlockStateRecordSlim{
			Properties:     r.Properties,
			CollisionBoxes: r.CollisionBoxes,
			OutlineBoxes:   r.OutlineBoxes,
			Air:            r.Air,
			Opaque:         r.Opaque,
			SolidBlock:     r.SolidBlock,
			Replaceable:    r.Replaceable,
			BlocksMovement: r.BlocksMovement,
			Climbable:      r.Climbable,
			DoorLike:       r.DoorLike,
			FenceLike:      r.FenceLike,
			Slab:           r.Slab,
			Stair:          r.Stair,
			LogOrLeaf:      r.LogOrLeaf,
			Water:          r.Water,
			Lava:           r.Lava,
			Fluid:          r.Fluid,
		}
		byBlock[r.BlockID] = append(byBlock[r.BlockID], slim)
	}

	// Process blocks in sorted order for deterministic output
	var blockIDs []string
	for blockID := range byBlock {
		blockIDs = append(blockIDs, blockID)
	}
	sort.Strings(blockIDs)

	for _, blockID := range blockIDs {
		states := byBlock[blockID]
		ns, path := splitBlockID(blockID) // e.g. "minecraft", "oak_fence"

		dir := filepath.Join(outRoot, ns)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}

		outFile := filepath.Join(dir, path+".json")
		file := loader.BlockStatesFile{
			BlockID: blockID,
			States:  states,
		}
		buf, err := json.MarshalIndent(file, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", blockID, err)
		}
		if err := os.WriteFile(outFile, buf, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outFile, err)
		}
	}

	return nil
}

func splitBlockID(blockID string) (namespace, path string) {
	// blockID like "minecraft:oak_fence"
	parts := strings.SplitN(blockID, ":", 2)
	if len(parts) == 1 {
		return "minecraft", parts[0]
	}
	return parts[0], parts[1]
}
