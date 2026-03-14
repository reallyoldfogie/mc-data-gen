# mc-data-gen: Minecraft data generator

This project automates running a Fabric-based data exporter for multiple
Minecraft versions, and provides Go structs + loader code for consuming the
generated data in your RL environment or other agents.

**Features:**
|- **Block Data**: Complete block state information with properties (sharded by namespace)
|- **Item Data**: Item definitions and metadata (sharded by namespace)
|- **Entity Data**: Entity types with dimensions per pose, size variants, attributes, and tags (sharded by namespace)
|- **Multi-Version Support**: Works with both Yarn-mapped (1.21.x) and non-obfuscated (26.1+) versions
|- **Source Decompilation**: Optional extraction of decompiled Minecraft sources to `work/<version>/extracted_src/`

It automatically queries Fabric Meta and the Fabric Maven to pick the
appropriate `minecraft_version`, `yarn_mappings` (for versions < 26.1), 
`loader_version`, and `fabric_api_version` for each game version, similar to 
using the https://fabricmc.net/develop/ UI.
  file utilities, and JSON loader.
- `fabric-template/` – minimal Fabric mod project used as a template for each version.
- `mc-data-gen.yaml` – configuration file listing versions and paths.

## Usage (high level)

1. Ensure you have Java + a Gradle-compatible environment.
2. Edit `mc-data-gen.yaml` to:
   - List the Minecraft versions you care about
   - Optionally enable `decompile_sources: true` to extract decompiled Java sources
3. Run:

   ```bash
   go run ./cmd/mc-data-gen -config mc-data-gen.yaml -work-dir ./work
   ```

4. For each version, the tool will:
   - Resolve loader and fabric-api versions via Fabric Meta + Maven.
   - For versions < 26.1: Also resolve Yarn mappings (deobfuscation).
   - For versions >= 26.1: Skip Yarn mappings (non-obfuscated by default).
   - Copy the Fabric template into `work/<version>`.
   - Set `minecraft_version`, `yarn_mappings` (if needed), `loader_version`,
     and `fabric_api_version` in `gradle.properties`.
   - Run `./gradlew runServer` (or your configured Gradle task).
   - Collect and shard `run/data/blocks.json` into `<cfg.output_dir>/<version>/blocks/<namespace>/<block>.json`.
   - Collect and shard `run/data/items.json` into `<cfg.output_dir>/<version>/items/<namespace>/<item>.json`.
   - Collect and shard `run/data/entities.json` into `<cfg.output_dir>/<version>/entities/<namespace>/<entity>.json`.
## Using the loader
You can consume generated data via the separate `loader` module.

Install:

```bash
go get github.com/reallyoldfogie/mc-data-gen/loader@latest
```

Examples:

**Loading blocks:**
```go
package main

import (
    "fmt"
    mdl "github.com/reallyoldfogie/mc-data-gen/loader"
)

func main() {
    // Load all block data from a version directory
    m, err := mdl.LoadBlocksDir("./data/1.21.5/blocks")
    if err != nil { panic(err) }

    key := mdl.StateKey{BlockID: "minecraft:stone", PropsKey: mdl.MakePropsKey(nil)}
    info := m[key]
    fmt.Println("passable?", info.IsPassable())
}
```

**Loading items:**
```go
package main

import (
    "fmt"
    mdl "github.com/reallyoldfogie/mc-data-gen/loader"
)

func main() {
    // Load all item data from a version directory
    items, err := mdl.LoadItemsDir("./data/1.21.5/items")
    if err != nil { panic(err) }

    ironSword := items["minecraft:iron_sword"]
    fmt.Println("max damage:", ironSword.Components.MaxDamage)
    fmt.Println("is weapon:", ironSword.IsWeapon)
}
```

**Loading entities:**
```go
package main

import (
    "fmt"
    mdl "github.com/reallyoldfogie/mc-data-gen/loader"
)

func main() {
    // Load all entity data from a version directory
    entities, err := mdl.LoadEntitiesDir("./data/1.21.5/entities")
    if err != nil { panic(err) }

    zombie := entities["minecraft:zombie"]
    fmt.Printf("Zombie default: %.2f x %.2f (eye: %.2f)\n",
        zombie.DefaultDimensions.Width,
        zombie.DefaultDimensions.Height,
        zombie.DefaultDimensions.EyeHeight)
    
    // Check dimensions in different poses
    if sleepDims, ok := zombie.PoseDimensions["SLEEPING"]; ok {
        fmt.Printf("Zombie sleeping: %.2f x %.2f\n",
            sleepDims.Width, sleepDims.Height)
    }
    
    // Check attributes for combat/movement
    for _, attr := range zombie.Attributes {
        if attr.Name == "minecraft:max_health" || attr.Name == "minecraft:movement_speed" {
            fmt.Printf("%s: %.2f\n", attr.Name, attr.BaseValue)
        }
    }
}
```

Using the loader presumes you have generated the data (see steps above) or downloaded a release. Adjust import paths if you fork/rename the module.
