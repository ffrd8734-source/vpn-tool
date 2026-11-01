package main

import (
	"vpn-tool/pkg/cmd/root"
)

var (
	version   = "1.0.0"
	buildDate = "2026-03-30"
)

func main() {
	root.Execute(version, buildDate)
}
