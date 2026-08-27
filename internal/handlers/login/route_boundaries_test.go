package login_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/grapinou/LazyMarking/internal/handlers/dashboard"
	"github.com/grapinou/LazyMarking/internal/handlers/generateExams"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/marking"
	"github.com/grapinou/LazyMarking/internal/handlers/subjects"
)

func TestRepresentativeUserRoutesRejectAnonymousRequests(t *testing.T) {
	initRouteSessionStore(t)
	for _, tc := range []struct {
		name     string
		path     string
		method   string
		register func(*http.ServeMux)
	}{
		{"dashboard", "/dashboard", http.MethodGet, dashboard.RegisterRoutes},
		{"CRUD", "/dashboard/questions/subjects", http.MethodGet, func(mux *http.ServeMux) { subjects.RegisterRoutes(mux, nil) }},
		{"marking", "/dashboard/marking", http.MethodGet, func(mux *http.ServeMux) {
			var jobs sync.WaitGroup
			marking.RegisterRoutes(mux, nil, context.Background(), &jobs)
		}},
		{"generation", "/dashboard/exams/generate", http.MethodPost, func(mux *http.ServeMux) {
			var jobs sync.WaitGroup
			generateexams.RegisterRoutes(mux, nil, context.Background(), &jobs)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			tc.register(mux)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != http.StatusFound || response.Header().Get("Location") != "/login" {
				t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
			}
		})
	}
}

func TestAuthenticatedDashboardRoutePassesMiddleware(t *testing.T) {
	t.Chdir("../../..")
	initRouteSessionStore(t)
	mux := http.NewServeMux()
	dashboard.RegisterRoutes(mux)
	seedRequest := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	session, err := login.GetStore().Get(seedRequest, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "alice"
	cookieResponse := httptest.NewRecorder()
	if err := session.Save(seedRequest, cookieResponse); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	request.AddCookie(cookieResponse.Result().Cookies()[0])
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestGenerationMutationsRejectGET(t *testing.T) {
	initRouteSessionStore(t)
	mux := http.NewServeMux()
	var jobs sync.WaitGroup
	generateexams.RegisterRoutes(mux, nil, context.Background(), &jobs)

	for _, path := range []string{"/dashboard/exams/generate?exam_id=1", "/dashboard/exams/generatemini?exam_id=1"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("GET %s status=%d, want 405", path, response.Code)
		}
	}
}

func initRouteSessionStore(t *testing.T) {
	t.Helper()
	t.Setenv("SESSION_KEY", "route-boundary-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
}
