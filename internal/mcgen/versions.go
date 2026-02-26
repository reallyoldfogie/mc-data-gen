package mcgen

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "sort"
    "strconv"
    "strings"
    "time"
)

// FabricMeta holds the resolved versions for a given Minecraft version.
type FabricMeta struct {
    MinecraftVersion  string
    YarnVersion       string // e.g. "1.21.1+build.1" (empty for 26.1+)
    LoaderVersion     string // e.g. "0.16.0"
    FabricAPIVersion  string // e.g. "0.103.0+1.21.1"
    LoomVersion       string // e.g. "1.11-SNAPSHOT" or "1.14-SNAPSHOT"
}

// minecraftVersion represents a parsed Minecraft version for comparison.
type minecraftVersion struct {
    major int
    minor int
    patch int
}

// parseMinecraftVersion parses a version string like "1.21.1" or "26.1-snapshot-1".
func parseMinecraftVersion(version string) (minecraftVersion, error) {
    // Remove snapshot suffix if present
    version = strings.Split(version, "-")[0]
    
    parts := strings.Split(version, ".")
    if len(parts) < 2 {
        return minecraftVersion{}, fmt.Errorf("invalid version format: %s", version)
    }
    
    major, err := strconv.Atoi(parts[0])
    if err != nil {
        return minecraftVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
    }
    
    minor, err := strconv.Atoi(parts[1])
    if err != nil {
        return minecraftVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
    }
    
    patch := 0
    if len(parts) >= 3 {
        patch, err = strconv.Atoi(parts[2])
        if err != nil {
            return minecraftVersion{}, fmt.Errorf("invalid patch version: %s", parts[2])
        }
    }
    
    return minecraftVersion{major: major, minor: minor, patch: patch}, nil
}

// needsYarnMappings determines if a Minecraft version needs Yarn mappings.
// Versions >= 26.1 ship with non-obfuscated code and don't need Yarn.
func needsYarnMappings(mcVersion string) bool {
    v, err := parseMinecraftVersion(mcVersion)
    if err != nil {
        // If we can't parse, assume it needs Yarn (safer default)
        return true
    }
    
    // Versions before 26.1 need Yarn mappings
    if v.major < 26 {
        return true
    }
    if v.major == 26 && v.minor < 1 {
        return true
    }
    
    // 26.1+ doesn't need Yarn
    return false
}

// ResolveFabricMeta queries Fabric Meta and Maven to resolve
// Yarn, loader, and fabric-api versions for a given MC version.
func ResolveFabricMeta(mcVersion string) (*FabricMeta, error) {
    client := &http.Client{Timeout: 20 * time.Second}

    // Versions >= 26.1 don't need Yarn mappings (non-obfuscated)
    yarn := ""
    if needsYarnMappings(mcVersion) {
        var err error
        yarn, err = fetchYarnForGame(client, mcVersion)
        if err != nil {
            return nil, err
        }
    }
    
    loader, err := fetchLoaderForGame(client, mcVersion)
    if err != nil {
        return nil, err
    }
    fabricAPI, err := fetchFabricAPIVersion(client, mcVersion)
    if err != nil {
        return nil, err
    }

    // Select Loom version based on Minecraft version
    // 26.1+ uses Loom 1.14-SNAPSHOT (supports Java 25)
    // 1.21.11+ requires Loom >= 1.13.3 (some Fabric API modules are built with newer Loom)
    // Older versions use Loom 1.11-SNAPSHOT
    loomVersion := "1.11-SNAPSHOT"
    v, err := parseMinecraftVersion(mcVersion)
    if err == nil {
        if v.major >= 26 {
            loomVersion = "1.14-SNAPSHOT"
        } else if v.major == 1 && v.minor == 21 && v.patch >= 11 {
            loomVersion = "1.13.3"
        }
    }

    return &FabricMeta{
        MinecraftVersion: mcVersion,
        YarnVersion:      yarn,
        LoaderVersion:    loader,
        FabricAPIVersion: fabricAPI,
        LoomVersion:      loomVersion,
    }, nil
}

// --- Yarn from Fabric Meta ---

type yarnVersion struct {
    GameVersion string `json:"gameVersion"`
    Separator   string `json:"separator"`
    Build       int    `json:"build"`
    Maven       string `json:"maven"`
    Version     string `json:"version"`
    Stable      bool   `json:"stable"`
}

func fetchYarnForGame(client *http.Client, gameVersion string) (string, error) {
    url := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/yarn/%s", gameVersion)
    body, err := httpGetAll(client, url)
    if err != nil {
        return "", err
    }
    var all []yarnVersion
    if err := json.Unmarshal(body, &all); err != nil {
        return "", fmt.Errorf("decode yarn for %s: %w", gameVersion, err)
    }
    if len(all) == 0 {
        return "", fmt.Errorf("no yarn versions for %s", gameVersion)
    }
    // Prefer first stable, else first entry (list is newest-first).
    for _, v := range all {
        if v.Stable {
            return v.Version, nil
        }
    }
    return all[0].Version, nil
}

// --- Loader from Fabric Meta ---

type loaderVersion struct {
    Separator string `json:"separator"`
    Build     int    `json:"build"`
    Maven     string `json:"maven"`
    Version   string `json:"version"`
    Stable    bool   `json:"stable"`
}

type loaderEntry struct {
    Loader loaderVersion `json:"loader"`
    // intermediary + launcherMeta omitted
}

func fetchLoaderForGame(client *http.Client, gameVersion string) (string, error) {
    url := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s", gameVersion)
    body, err := httpGetAll(client, url)
    if err != nil {
        return "", err
    }
    var all []loaderEntry
    if err := json.Unmarshal(body, &all); err != nil {
        return "", fmt.Errorf("decode loader for %s: %w", gameVersion, err)
    }
    if len(all) == 0 {
        return "", fmt.Errorf("no loader versions for %s", gameVersion)
    }
    for _, e := range all {
        if e.Loader.Stable {
            return e.Loader.Version, nil
        }
    }
    return all[0].Loader.Version, nil
}

// --- Fabric API from Maven metadata ---

type mavenMetadata struct {
    Versioning struct {
        Versions []string `xml:"versions>version"`
    } `xml:"versioning"`
}

func fetchFabricAPIVersion(client *http.Client, mcVersion string) (string, error) {
    const url = "https://maven.fabricmc.net/net/fabricmc/fabric-api/fabric-api/maven-metadata.xml"
    body, err := httpGetAll(client, url)
    if err != nil {
        return "", err
    }
    var meta mavenMetadata
    if err := xml.Unmarshal(body, &meta); err != nil {
        return "", fmt.Errorf("decode fabric-api metadata: %w", err)
    }

    // Strip snapshot suffix for Fabric API lookup (e.g., 26.1-snapshot-1 -> 26.1)
    // Fabric API versions use base version numbers only
    baseVersion := strings.Split(mcVersion, "-")[0]
    suffix := "+" + baseVersion
    
    var candidates []string
    for _, v := range meta.Versioning.Versions {
        if strings.HasSuffix(v, suffix) {
            candidates = append(candidates, v)
        }
    }
    if len(candidates) == 0 {
        return "", fmt.Errorf("no fabric-api versions for minecraft %s (base: %s)", mcVersion, baseVersion)
    }
    sort.Strings(candidates)
    return candidates[len(candidates)-1], nil
}

// --- HTTP helper ---

func httpGetAll(client *http.Client, url string) ([]byte, error) {
    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("GET %s: %w", url, err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("GET %s: status %s, body=%s", url, resp.Status, string(b))
    }
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read %s: %w", url, err)
    }
    return data, nil
}
