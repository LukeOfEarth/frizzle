package main

import (
	"os"

	"github.com/LukeOfEarth/frizzle/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
