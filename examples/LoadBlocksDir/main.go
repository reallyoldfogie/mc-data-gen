package main

import (
    "fmt"
    "log"
    "path/filepath"

    loader "github.com/reallyoldfogie/mc-data-gen/loader"
)

func main() {
    // Update version to one you have generated locally
    path := filepath.Join("data", "1.21.5", "blocks")
    shapes, err := loader.LoadBlocksDir(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("loaded information for %d block shapes\n", len(shapes))
	// Example lookup for stone with no properties.
    key := loader.StateKey{
        BlockID:  "minecraft:stone",
        PropsKey: loader.MakePropsKey(map[string]string{}),
    }
	info, ok := shapes[key]
	if !ok {
		fmt.Println("no shape info for stone")
		return
	}

	fmt.Printf("Stone has %d collision boxes\n", len(info.Collision))
}
