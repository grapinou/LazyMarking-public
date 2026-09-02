package httpsecurity

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/csrf"
)

const (
	csrfAuthKeyLength  = 32
	maxUnsafeBodyBytes = 100 << 20
)

type requestResponseWriter struct {
	http.ResponseWriter
	request *http.Request
}

func (w *requestResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func NewCSRFMiddlewareFromEnvironment() (func(http.Handler) http.Handler, error) {
	authKey := []byte(os.Getenv("CSRF_AUTH_KEY"))
	if len(authKey) != csrfAuthKeyLength {
		return nil, errors.New("CSRF_AUTH_KEY must contain exactly 32 bytes")
	}
	if sessionKey := os.Getenv("SESSION_KEY"); sessionKey != "" && sessionKey == string(authKey) {
		return nil, errors.New("CSRF_AUTH_KEY must be distinct from SESSION_KEY")
	}
	secureValue := os.Getenv("SESSION_SECURE")
	if secureValue == "" {
		return nil, errors.New("SESSION_SECURE must be explicitly set to true or false")
	}
	secure, err := strconv.ParseBool(secureValue)
	if err != nil {
		return nil, errors.New("SESSION_SECURE must be a boolean")
	}
	return NewCSRFMiddleware(authKey, secure), nil
}

func NewCSRFMiddleware(authKey []byte, secure bool) func(http.Handler) http.Handler {
	protect := csrf.Protect(
		authKey,
		csrf.Path("/"),
		csrf.HttpOnly(true),
		csrf.Secure(secure),
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.CookieName("_lazymarking_csrf"),
		csrf.MaxAge(7200),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Cette requête de sécurité n'est plus valide. Rechargez la page puis réessayez.", http.StatusForbidden)
		})),
	)
	return func(next http.Handler) http.Handler {
		withRequestWriter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&requestResponseWriter{ResponseWriter: w, request: r}, r)
		})
		protected := protect(withRequestWriter)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Gorilla CSRF reads form values before the application handler. Cap
			// unsafe request bodies here so multipart parsing cannot bypass the
			// upload admission limits installed by downstream handlers.
			if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && r.Method != http.MethodTrace {
				r.Body = http.MaxBytesReader(w, r.Body, maxUnsafeBodyBytes)
			}
			if !secure {
				r = csrf.PlaintextHTTPRequest(r)
			}
			protected.ServeHTTP(w, r)
		})
	}
}

func TemplateFuncs(w http.ResponseWriter) template.FuncMap {
	return template.FuncMap{
		csrf.TemplateTag: func() template.HTML {
			carrier, ok := w.(*requestResponseWriter)
			if !ok || carrier.request == nil {
				return ""
			}
			return csrf.TemplateField(carrier.request)
		},
	}
}
