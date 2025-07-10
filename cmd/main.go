package main

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/home"
)

func main() {

	mux := http.NewServeMux()
	home.RegisterRoutes(mux)

	// Starting server
	const port = ":8080"
	log.Println("Starting Server at port ", port)
	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Error Server", err)
	}

}
