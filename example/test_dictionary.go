package main

import (
	"github.com/giorgisio/goav/avcodec"
	"fmt"
)

func main() {
	var dictionary *avcodec.Dictionary
	fmt.Print("Setting value in dictionary\n")
	if rc := avcodec.AvDictSet(&dictionary, "Key0", "Value0", 0); rc < 0 {
		fmt.Print("Failed to set dictionary value: %d\n", rc)
		return
	}

	fmt.Printf("dictionary pointer: %p\n", dictionary)

	entry := avcodec.AvDictGet(dictionary, "Key0", nil, 0)
	if entry == nil {
		fmt.Print("Entry Key0 not found\n")
		return
	}

	fmt.Printf("Found value for Key0: %s\n", entry.Value())
}
