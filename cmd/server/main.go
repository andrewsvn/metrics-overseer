package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/config/server"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	run()
}

func run() {
	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	mhandlers := handler.NewMetricsHandlers(msrv)

	r := chi.NewRouter()
	r.Post("/update/{mtype}/{id}/{value}", mhandlers.UpdateHandler())
	r.Get("/value/{mtype}/{id}", mhandlers.GetValueHandler())
	r.Get("/", mhandlers.ShowMetricsPage())

	log.Printf("Starting server on port %d\n", server.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", server.ServerPort), r))
}
