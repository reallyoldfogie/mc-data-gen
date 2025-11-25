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

	fmt.Printf("Using config: %s", *configPath)
	fmt.Printf("Work dir:     %s", *workDir)
	fmt.Printf("Output dir:   %s", cfg.OutputDir)

	for _, v := range cfg.Versions {
		fmt.Printf("\n=== Generating data for %s ===\n", v)

		meta, err := mcgen.ResolveFabricMeta(v)
		if err != nil {
			log.Fatalf("resolve fabric meta for %s: %v", v, err)
		}

		fmt.Printf("\n%#v\n\n", meta)

		fmt.Printf("  minecraft_version = %s\n", meta.MinecraftVersion)
		fmt.Printf("  yarn_mappings     = %s\n", meta.YarnVersion)
		fmt.Printf("  loader_version    = %s\n", meta.LoaderVersion)
		fmt.Printf("  fabric_api_version= %s\n", meta.FabricAPIVersion)

		projectDir := filepath.Join(*workDir, v)

		if err := mcgen.PrepareProject(cfg.FabricTemplateDir, projectDir, meta); err != nil {
			log.Panicf("prepare project for %s: %v", v, err)
		}

		if err := mcgen.RunGradle(projectDir, cfg.GradleTask); err != nil {
			log.Panicf("gradle failed for %s: %v", v, err)
		}

		if err := mcgen.CollectOutput(projectDir, cfg.GeneratorOutputRel, cfg.OutputDir, v); err != nil {
			log.Panicf("collect output for %s: %v", v, err)
		}

		fmt.Printf("Done %s\n", v)
		runtime.GC()
	}
}
