package main

import (
	"github.com/boyter/scc/processor"
)

//go:generate go run scripts/include.go
func main() {
	processor.Process()
}
