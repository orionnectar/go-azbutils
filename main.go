package main

import "github.com/orionnectar/go-azbutils/cmd"

var version = "dev" // default, replaced by GoReleaser via ldflags

func main() {
	cmd.Execute(version)
}
