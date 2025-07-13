package main

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/dashboard"
	appdb "github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/about"
	"github.com/grapinou/LazyMarking/internal/handlers/home"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/register"
	"github.com/joho/godotenv"
)

const dbPath = "./db/data/app.db"

func main() {
	// .env load
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found.")
	}

	// cookie init
	login.InitSessionStore()

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
	login.RegisterRoutes(mux, queries)

	// dashboard
	dashboard.RegisterRoutes(mux)

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
