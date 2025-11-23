# mc-data-gen: Minecraft collision shape generator

This project automates running a Fabric-based collision exporter for multiple
Minecraft versions, and provides Go structs + loader code for consuming the
generated `blocks.json` files in your RL environment or other agents.

It automatically queries Fabric Meta and the Fabric Maven to pick the
appropriate `minecraft_version`, `yarn_mappings`, `loader_version`, and
`fabric_api_version` for each game version, similar to using the
https://fabricmc.net/develop/ UI.

## Layout

- `cmd/mc-data-gen/` – CLI tool that drives the generation.
- `internal/mcgen/` – config, HTTP version resolution, Gradle orchestration,
  file utilities, and JSON loader.
- `fabric-template/` – minimal Fabric mod project used as a template for each version.
- `mc-data-gen.yaml` – configuration file listing versions and paths.

## Usage (high level)

1. Ensure you have Java + a Gradle-compatible environment.
2. Edit `mc-data-gen.yaml` to list the Minecraft versions you care about.
3. Run:

   ```bash
   go run ./cmd/mcgen -config mc-data-gen.yaml -work-dir ./work
   ```

4. For each version, the tool will:
   - Resolve Yarn, loader, and fabric-api versions via Fabric Meta + Maven.
   - Copy the Fabric template into `work/<version>`.
   - Set `minecraft_version`, `yarn_mappings`, `loader_version`,
     and `fabric_api_version` in `gradle.properties`.
   - Run `./gradlew migrateMappings --mappings "<yarn_version>"`.
   - Run `./gradlew runServer` (or your configured Gradle task).
   - Collect `run/collision-data/blocks.json` into `<cfg.output_dir>/<version>/blocks/minecraft/<block files>`.

## Using the loader

See the `examples` directory for loader usage.  Using the loader presumes you have either generated the data, or downloaded it from github.

Adjust module path (`github.com/reallyoldfogie/mc-data-gen`) to match your fork if you rename it.
