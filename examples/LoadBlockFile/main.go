package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/reallyoldfogie/mc-data-gen/internal/mcgen"
)

func main() {
	path := filepath.Join("data", "1.21.1", "blocks", "minecraft", "stone.json")
	shapes, err := mcgen.LoadBlocksFile(path)
	if err != nil {
		log.Fatal(err)
	}

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
