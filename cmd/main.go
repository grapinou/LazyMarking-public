package main

import (
	"context"
	"log"
	"net/http"
	"time"

	appdb "github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/about"
	"github.com/grapinou/LazyMarking/internal/handlers/dashboard"
	"github.com/grapinou/LazyMarking/internal/handlers/difficulties"
	"github.com/grapinou/LazyMarking/internal/handlers/home"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/logout"
	"github.com/grapinou/LazyMarking/internal/handlers/questions"
	"github.com/grapinou/LazyMarking/internal/handlers/register"
	"github.com/grapinou/LazyMarking/internal/handlers/resetpassword"
	"github.com/grapinou/LazyMarking/internal/handlers/skills"
	"github.com/grapinou/LazyMarking/internal/handlers/subjects"
	"github.com/grapinou/LazyMarking/internal/handlers/themes"
	"github.com/grapinou/LazyMarking/internal/handlers/yearlevels"
	"github.com/grapinou/LazyMarking/internal/task"
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

	// token clearer

	ctx := context.Background()
	task.StartTokenCleaner(ctx, queries, 24*time.Hour)

	mux := http.NewServeMux()

	// home
	home.RegisterRoutes(mux)
	about.RegisterRoutes(mux)
	register.RegisterRoutes(mux, queries)
	login.RegisterRoutes(mux, queries)
	logout.RegisterRoutes(mux)
	resetpassword.RegisterRoutes(mux, conn, queries)

	// dashboard
	dashboard.RegisterRoutes(mux)
	questions.RegisterRoutes(mux, queries)
	subjects.RegisterRoutes(mux, queries)
	themes.RegisterRoutes(mux, queries)
	yearlevels.RegisterRoutes(mux, queries)
	skills.RegisterRoutes(mux, queries)
	difficulties.RegisterRoutes(mux, queries)

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
