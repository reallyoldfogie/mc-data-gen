package mcgen

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

type Config struct {
    OutputDir          string   `yaml:"output_dir"`
    FabricTemplateDir  string   `yaml:"fabric_template_dir"`
    GradleTask         string   `yaml:"gradle_task"`
    Versions           []string `yaml:"versions"`
    GeneratorOutputRel string   `yaml:"generator_output_rel"`
    DecompileSources   bool     `yaml:"decompile_sources"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("unmarshal yaml: %w", err)
    }
    if cfg.OutputDir == "" {
        return nil, fmt.Errorf("output_dir is required")
    }
    if cfg.FabricTemplateDir == "" {
        return nil, fmt.Errorf("fabric_template_dir is required")
    }
    if cfg.GradleTask == "" {
        cfg.GradleTask = "runServer"
    }
    if cfg.GeneratorOutputRel == "" {
        cfg.GeneratorOutputRel = "run/collision-data/blocks.json"
    }
    if len(cfg.Versions) == 0 {
        return nil, fmt.Errorf("versions list is empty")
    }
    return &cfg, nil
}
