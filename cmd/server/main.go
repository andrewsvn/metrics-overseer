package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/config/server"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
)

func main() {
	run()
}

func run() {
	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	mhandlers := handler.NewMetricsHandlers(msrv)

	r := mhandlers.GetRouter()

	log.Printf("Starting server on port %d\n", server.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", server.ServerPort), r))
}
