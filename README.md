# mc-data-gen: Minecraft data generator

This project automates running a Fabric-based data exporter for multiple
Minecraft versions, and provides Go structs + loader code for consuming the
generated data in your RL environment or other agents.

**Features:**
- **Block Data**: Complete block state information with properties
- **Item Data**: Item definitions and metadata
- **Protocol Extraction**: Network protocol definitions with 90-95% coverage (new!)
- **Multi-Version Support**: Works with both Yarn-mapped (1.21.x) and non-obfuscated (26.1+) versions

It automatically queries Fabric Meta and the Fabric Maven to pick the
appropriate `minecraft_version`, `yarn_mappings` (for versions < 26.1), 
`loader_version`, and `fabric_api_version` for each game version, similar to 
using the https://fabricmc.net/develop/ UI.
  file utilities, and JSON loader.
- `fabric-template/` – minimal Fabric mod project used as a template for each version.
- `mc-data-gen.yaml` – configuration file listing versions and paths.

## Usage (high level)

1. Ensure you have Java + a Gradle-compatible environment.
2. Edit `mc-data-gen.yaml` to list the Minecraft versions you care about.
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
   - Collect `run/data/blocks.json` into `<cfg.output_dir>/<version>/blocks/minecraft/<block files>`.
   - Collect `run/data/items.json` into `<cfg.output_dir>/<version>/items/items.json`.
   - Collect `run/data/protocol.json` into `<cfg.output_dir>/<version>/protocol/protocol.json`.
## Using the loader
You can consume generated data via the separate `loader` module.

Install:

```bash
go get github.com/reallyoldfogie/mc-data-gen/loader@latest
```

Example:

```go
package main

import (
    "fmt"
    mdl "github.com/reallyoldfogie/mc-data-gen/loader"
)

func main() {
    // Point to a generated version directory
    m, err := mdl.LoadBlocksDir("./data/1.21.5/blocks")
    if err != nil { panic(err) }

    key := mdl.StateKey{BlockID: "minecraft:stone", PropsKey: mdl.MakePropsKey(nil)}
    info := m[key]
    fmt.Println("passable?", info.IsPassable())
}
```

Using the loader presumes you have generated the data (see steps above) or downloaded a release. Adjust import paths if you fork/rename the module.
