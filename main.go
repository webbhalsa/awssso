package main

import "github.com/webbhalsa/awssso/cmd"

var version = "dev" // overridden by GoReleaser: -X main.version={{.Version}}

func main() {
	cmd.Execute(version)
}
