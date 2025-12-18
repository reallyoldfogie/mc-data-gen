package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/reallyoldfogie/mc-data-gen/internal/mcgen"
)

type versionResult struct {
	version string
	success bool
	err     error
}

func main() {
	configPath := flag.String("config", "mc-data-gen.yaml", "path to config file (YAML)")
	workDir := flag.String("work-dir", "./work", "directory for generated per-version Fabric projects")
	flag.Parse()

	cfg, err := mcgen.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := os.MkdirAll(*workDir, 0o755); err != nil {
		log.Fatalf("create work dir: %v", err)
	}

	fmt.Printf("Using config: %s\n", *configPath)
	fmt.Printf("Work dir:     %s\n", *workDir)
	fmt.Printf("Output dir:   %s\n", cfg.OutputDir)

	// Track results for all versions
	var results []versionResult

	for _, v := range cfg.Versions {
		fmt.Printf("\n=== Generating data for %s ===\n", v)

		if err := processVersion(v, *workDir, cfg); err != nil {
			fmt.Printf("❌ FAILED: %s - %v\n", v, err)
			results = append(results, versionResult{version: v, success: false, err: err})
		} else {
			fmt.Printf("✅ Done %s\n", v)
			results = append(results, versionResult{version: v, success: true})
		}

		runtime.GC()
	}

	// Print summary
	fmt.Printf("\n=== Summary ===\n")
	successCount := 0
	for _, r := range results {
		if r.success {
			successCount++
			fmt.Printf("✅ %s\n", r.version)
		} else {
			fmt.Printf("❌ %s: %v\n", r.version, r.err)
		}
	}

	fmt.Printf("\nTotal: %d/%d succeeded\n", successCount, len(results))

	if successCount < len(results) {
		os.Exit(1)
	}
}

// processVersion handles the complete workflow for a single version
func processVersion(version, workDir string, cfg *mcgen.Config) error {
	meta, err := mcgen.ResolveFabricMeta(version)
	if err != nil {
		return fmt.Errorf("resolve fabric meta: %w", err)
	}

	fmt.Printf("\n%#v\n\n", meta)

	fmt.Printf("  minecraft_version = %s\n", meta.MinecraftVersion)
	fmt.Printf("  yarn_mappings     = %s\n", meta.YarnVersion)
	fmt.Printf("  loader_version    = %s\n", meta.LoaderVersion)
	fmt.Printf("  fabric_api_version= %s\n", meta.FabricAPIVersion)

	projectDir := filepath.Join(workDir, version)

	if err := mcgen.PrepareProject(cfg.FabricTemplateDir, projectDir, meta); err != nil {
		return fmt.Errorf("prepare project: %w", err)
	}

	if err := mcgen.RunGradle(projectDir, cfg.GradleTask); err != nil {
		return fmt.Errorf("gradle failed: %w", err)
	}

	if err := mcgen.CollectOutput(projectDir, cfg.GeneratorOutputRel, cfg.OutputDir, version); err != nil {
		return fmt.Errorf("collect output: %w", err)
	}

	// Decompile sources if enabled
	if cfg.DecompileSources {
		fmt.Printf("  Decompiling sources...\n")
		if err := mcgen.DecompileSources(projectDir); err != nil {
			return fmt.Errorf("decompile sources: %w", err)
		}
		fmt.Printf("  ✅ Sources extracted to %s/extracted_src\n", projectDir)
	}

	return nil
}
