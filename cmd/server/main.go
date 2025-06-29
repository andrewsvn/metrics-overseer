package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/config"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
)

func main() {
	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	mhandlers := handler.NewMetricsHandlers(msrv)

	http.HandleFunc("/update/", mhandlers.UpdateHandler())
	log.Printf("Starting server on port %d\n", config.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.ServerPort), nil))
}
