package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	appdb "github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/about"
	altanswers "github.com/grapinou/LazyMarking/internal/handlers/altAnswers"
	altimages "github.com/grapinou/LazyMarking/internal/handlers/altImages"
	altpreview "github.com/grapinou/LazyMarking/internal/handlers/altPreview"
	altquestions "github.com/grapinou/LazyMarking/internal/handlers/altQuestions"
	"github.com/grapinou/LazyMarking/internal/handlers/answers"
	classcodes "github.com/grapinou/LazyMarking/internal/handlers/classCodes"
	"github.com/grapinou/LazyMarking/internal/handlers/dashboard"
	"github.com/grapinou/LazyMarking/internal/handlers/difficulties"
	"github.com/grapinou/LazyMarking/internal/handlers/errorsmessages"
	"github.com/grapinou/LazyMarking/internal/handlers/exams"
	generateexams "github.com/grapinou/LazyMarking/internal/handlers/generateExams"
	"github.com/grapinou/LazyMarking/internal/handlers/home"
	"github.com/grapinou/LazyMarking/internal/handlers/images"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/logout"
	"github.com/grapinou/LazyMarking/internal/handlers/marking"
	"github.com/grapinou/LazyMarking/internal/handlers/periods"
	"github.com/grapinou/LazyMarking/internal/handlers/points"
	"github.com/grapinou/LazyMarking/internal/handlers/preview"
	"github.com/grapinou/LazyMarking/internal/handlers/qcm"
	qcmpreview "github.com/grapinou/LazyMarking/internal/handlers/qcmPreview"
	qcmquestions "github.com/grapinou/LazyMarking/internal/handlers/qcmQuestions"
	"github.com/grapinou/LazyMarking/internal/handlers/questions"
	"github.com/grapinou/LazyMarking/internal/handlers/register"
	"github.com/grapinou/LazyMarking/internal/handlers/resetpassword"
	"github.com/grapinou/LazyMarking/internal/handlers/skills"
	studentclasscode "github.com/grapinou/LazyMarking/internal/handlers/studentClassCode"
	"github.com/grapinou/LazyMarking/internal/handlers/students"
	"github.com/grapinou/LazyMarking/internal/handlers/subjects"
	"github.com/grapinou/LazyMarking/internal/handlers/themes"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/handlers/yearlevels"
	"github.com/grapinou/LazyMarking/internal/handlers/years"
	"github.com/grapinou/LazyMarking/internal/httpsecurity"
	"github.com/grapinou/LazyMarking/internal/task"
	"github.com/joho/godotenv"
)

func main() {
	// .env load
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found.")
	}

	// cookie init
	if err := login.InitSessionStore(); err != nil {
		log.Fatal("Failed to initialize session store: ", err)
	}
	csrfMiddleware, err := httpsecurity.NewCSRFMiddlewareFromEnvironment()
	if err != nil {
		log.Fatal("Failed to initialize CSRF protection: ", err)
	}

	// db initialization
	conn, err := appdb.InitDB(config.DatabasePath)
	if err != nil {
		log.Fatal("Failed connect to db :", err)
	}
	defer conn.Close()

	// client sqlc
	queries := appdb.New(conn)

	// token clearer

	appCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	markingRecovery, err := tools.RecoverRunningMarkingJobs(appCtx, queries)
	if err != nil {
		log.Fatal("Failed to recover interrupted marking jobs: ", err)
	}
	log.Printf(
		"Marking recovery: %d jobs found, %d recovered, %d cleanup failures, %d transition failures",
		markingRecovery.Found,
		markingRecovery.Recovered,
		markingRecovery.CleanupFailures,
		markingRecovery.TransitionFailures,
	)
	if err := tools.RecoverRunningExamGenerations(appCtx, queries); err != nil {
		log.Fatal("Failed to recover interrupted exam generations: ", err)
	}
	if err := tools.PurgeExpiredMarkingJobs(appCtx, queries, time.Now()); err != nil {
		log.Printf("Failed to purge expired marking jobs: %v", err)
	}
	if err := tools.PurgeExpiredEphemeralWorkspaces(time.Now()); err != nil {
		log.Printf("Failed to purge expired preview workspaces: %v", err)
	}
	task.StartTokenCleaner(appCtx, queries, 24*time.Hour)

	mux := http.NewServeMux()

	// home
	home.RegisterRoutes(mux)
	about.RegisterRoutes(mux)
	register.RegisterRoutes(mux, queries)
	login.RegisterRoutes(mux, queries)
	logout.RegisterRoutes(mux)
	resetpassword.RegisterRoutes(mux, conn, queries)

	// User images require both authentication and database ownership.
	mux.Handle("GET "+config.PublicImageBaseURL+"{filename}", login.CheckAuth(
		tools.HandlerWithDB(tools.ServeUserImageHandler, queries)))

	// dashboard
	dashboard.RegisterRoutes(mux)
	questions.RegisterRoutes(mux, queries)
	subjects.RegisterRoutes(mux, queries)
	themes.RegisterRoutes(mux, queries)
	yearlevels.RegisterRoutes(mux, queries)
	skills.RegisterRoutes(mux, queries)
	difficulties.RegisterRoutes(mux, queries)
	points.RegisterRoutes(mux, queries)
	errorsmessages.RegisterRoutes(mux)
	answers.RegisterRoutes(mux, queries)
	altquestions.RegisterRoutes(mux, queries)
	altanswers.RegisterRoutes(mux, queries)
	images.RegisterRoutes(mux, queries)
	altimages.RegisterRoutes(mux, queries)
	preview.RegisterRoutes(mux, queries)
	altpreview.RegisterRoutes(mux, queries)
	classcodes.RegisterRoutes(mux, queries)
	students.RegisterRoutes(mux, queries, conn)
	studentclasscode.RegisterRoutes(mux, queries)
	qcm.RegisterRoutes(mux, queries)
	qcmquestions.RegisterRoutes(mux, queries, conn)
	qcmpreview.RegisterRoutes(mux, queries)
	exams.RegisterRoutes(mux, queries)
	years.RegisterRoutes(mux, queries)
	periods.RegisterRoutes(mux, queries)
	var backgroundJobs sync.WaitGroup
	generateexams.RegisterRoutes(mux, queries, appCtx, &backgroundJobs)
	marking.RegisterRoutes(mux, queries, appCtx, &backgroundJobs)

	// Starting server
	const port = ":8080"
	log.Println("Starting Server at port ", port)
	server := &http.Server{
		Addr:              port,
		Handler:           tools.SecurityHeaders(csrfMiddleware(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	var serverErr error
	serverStopped := false
	select {
	case <-appCtx.Done():
		log.Println("Server shutdown requested")
	case serverErr = <-serverErrors:
		serverStopped = true
		stop()
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		log.Printf("Server shutdown error: %v", shutdownErr)
		if err := server.Close(); err != nil {
			log.Printf("Forced server close error: %v", err)
		}
	}
	cancelShutdown()

	if !serverStopped {
		serverErr = <-serverErrors
	}
	unexpectedServerErr := serverErr != nil && serverErr != http.ErrServerClosed
	if unexpectedServerErr {
		log.Printf("Server error: %v", serverErr)
	}

	if shutdownErr == nil {
		backgroundCtx, cancelBackground := context.WithTimeout(context.Background(), 2*time.Minute)
		if err := waitForJobs(backgroundCtx, &backgroundJobs); err != nil {
			log.Printf("Timed out waiting for background jobs: %v", err)
		} else {
			log.Println("All background jobs stopped")
		}
		cancelBackground()
	} else {
		log.Println("Skipping background job wait because HTTP shutdown did not complete")
	}
	if unexpectedServerErr {
		log.Fatal("Server stopped unexpectedly: ", serverErr)
	}
}
