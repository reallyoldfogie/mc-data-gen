package main

import (
    "fmt"
    "log"
    "path/filepath"

    loader "github.com/reallyoldfogie/mc-data-gen/loader"
)

func main() {
    // Update version to one you have generated locally
    path := filepath.Join("data", "1.21.5", "blocks", "minecraft", "stone.json")
    shapes, err := loader.LoadBlocksFile(path)
	if err != nil {
		log.Fatal(err)
	}

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
