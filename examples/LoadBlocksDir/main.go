package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/reallyoldfogie/mc-data-gen/internal/mcgen"
)

func main() {
	path := filepath.Join("data", "1.21.1", "blocks")
	shapes, err := mcgen.LoadBlocksDir(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("loaded information for %d block shapes\n", len(shapes))
	// Example lookup for stone with no properties.
	key := mcgen.StateKey{
		BlockID:  "minecraft:stone",
		PropsKey: mcgen.MakePropsKey(map[string]string{}),
	}
	info, ok := shapes[key]
	if !ok {
		fmt.Println("no shape info for stone")
		return
	}

	fmt.Printf("Stone has %d collision boxes\n", len(info.Collision))
}
