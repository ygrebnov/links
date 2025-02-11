package main

import (
	"log"

	"github.com/ygrebnov/links/cmd/links"
)

func main() {
	if err := links.Execute(); err != nil {
		log.Fatal(err)
	}
}
