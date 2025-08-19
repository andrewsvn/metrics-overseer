package main

import (
	"github.com/andrewsvn/metrics-overseer/internal/server"
	"log"
)

func main() {
	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
