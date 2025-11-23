package mcgen

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "sort"
    "strings"
    "time"
)

// FabricMeta holds the resolved versions for a given Minecraft version.
type FabricMeta struct {
    MinecraftVersion  string
    YarnVersion       string // e.g. "1.21.1+build.1"
    LoaderVersion     string // e.g. "0.16.0"
    FabricAPIVersion  string // e.g. "0.103.0+1.21.1"
}

// ResolveFabricMeta queries Fabric Meta and Maven to resolve
// Yarn, loader, and fabric-api versions for a given MC version.
func ResolveFabricMeta(mcVersion string) (*FabricMeta, error) {
    client := &http.Client{Timeout: 20 * time.Second}

    yarn, err := fetchYarnForGame(client, mcVersion)
    if err != nil {
        return nil, err
    }
    loader, err := fetchLoaderForGame(client, mcVersion)
    if err != nil {
        return nil, err
    }
    fabricAPI, err := fetchFabricAPIVersion(client, mcVersion)
    if err != nil {
        return nil, err
    }

    return &FabricMeta{
        MinecraftVersion: mcVersion,
        YarnVersion:      yarn,
        LoaderVersion:    loader,
        FabricAPIVersion: fabricAPI,
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

    suffix := "+" + mcVersion
    var candidates []string
    for _, v := range meta.Versioning.Versions {
        if strings.HasSuffix(v, suffix) {
            candidates = append(candidates, v)
        }
    }
    if len(candidates) == 0 {
        return "", fmt.Errorf("no fabric-api versions for minecraft %s", mcVersion)
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
