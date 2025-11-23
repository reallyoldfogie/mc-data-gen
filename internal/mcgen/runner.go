package mcgen

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "minecraft_version="):
			lines[i] = "minecraft_version=" + meta.MinecraftVersion
			foundMC = true
		case strings.HasPrefix(trimmed, "yarn_mappings="):
			lines[i] = "yarn_mappings=" + meta.YarnVersion
			foundYarn = true
		case strings.HasPrefix(trimmed, "loader_version="):
			lines[i] = "loader_version=" + meta.LoaderVersion
			foundLoader = true
		case strings.HasPrefix(trimmed, "fabric_api_version="):
			lines[i] = "fabric_api_version=" + meta.FabricAPIVersion
			foundAPI = true
		}
	}

	if !foundMC {
		lines = append(lines, "minecraft_version="+meta.MinecraftVersion)
	}
	if meta.YarnVersion != "" && !foundYarn {
		lines = append(lines, "yarn_mappings="+meta.YarnVersion)
	}
	if meta.LoaderVersion != "" && !foundLoader {
		lines = append(lines, "loader_version="+meta.LoaderVersion)
	}
	if meta.FabricAPIVersion != "" && !foundAPI {
		lines = append(lines, "fabric_api_version="+meta.FabricAPIVersion)
	}

	out := strings.Join(lines, "\n")
	if err := os.WriteFile(gradlePropsPath, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write gradle.properties: %w", err)
	}

	return nil
}

// RunGradleWithArgs runs ./gradlew <args...> in the given projectDir.
func RunGradleWithArgs(projectDir string, args ...string) error {
	gradlew := "./gradlew"
	if _, err := os.Stat(filepath.Join(projectDir, "gradlew")); err != nil {
		return fmt.Errorf("gradlew not found in %s: %w", projectDir, err)
	}

	cmd := exec.Command(gradlew, args...)
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

	destDir := filepath.Join(outputRoot, version, "blocks")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	// dest := filepath.Join(destDir, "blocks.json")

	// data, err := os.ReadFile(src)
	// if err != nil {
	// 	return fmt.Errorf("read generator output: %w", err)
	// }
	// if err := os.WriteFile(dest, data, 0o644); err != nil {
	// 	return fmt.Errorf("write dest file: %w", err)
	// }

	shardFile(src, destDir)

	return nil
}

func shardFile(inputPath, outRoot string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}

	var records []BlockStateRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("unmarshal %s: %w", inputPath, err)
	}

	// Group by block_id
	byBlock := make(map[string][]BlockStateRecordSlim)
	for _, r := range records {
		slim := BlockStateRecordSlim{
			Properties:     r.Properties,
			CollisionBoxes: r.CollisionBoxes,
			OutlineBoxes:   r.OutlineBoxes,
			Air:            r.Air,
			Opaque:         r.Opaque,
			SolidBlock:     r.SolidBlock,
			Replaceable:    r.Replaceable,
			BlocksMovement: r.BlocksMovement,
		}
		byBlock[r.BlockID] = append(byBlock[r.BlockID], slim)
	}

	for blockID, states := range byBlock {
		ns, path := splitBlockID(blockID) // e.g. "minecraft", "oak_fence"

		dir := filepath.Join(outRoot, ns)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}

		outFile := filepath.Join(dir, path+".json")
		file := BlockStatesFile{
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
