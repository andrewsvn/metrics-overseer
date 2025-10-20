package main

import (
	"log"

	"github.com/andrewsvn/metrics-overseer/internal/server"
)

func main() {
	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
