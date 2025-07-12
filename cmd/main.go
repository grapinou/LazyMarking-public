package main

import (
	"log"
	"net/http"

	appdb "github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/about"
	"github.com/grapinou/LazyMarking/internal/handlers/home"
	"github.com/grapinou/LazyMarking/internal/handlers/register"
)

const dbPath = "./db/data/app.db"

func main() {
	// db initialization
	conn, err := appdb.InitDB(dbPath)
	if err != nil {
		log.Fatal("Failed connect to db :", err)
	}
	defer conn.Close()

	// client sqlc
	queries := appdb.New(conn)

	mux := http.NewServeMux()

	// home
	home.RegisterRoutes(mux)
	about.RegisterRoutes(mux)
	register.RegisterRoutes(mux, queries)

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
