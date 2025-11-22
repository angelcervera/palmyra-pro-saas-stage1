package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
)

func TestRequestTraceWithAuth(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(platformauth.JWT(platformauth.UnsignedTokenVerifier(), nil))
	r.Use(RequestTrace)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		audit, ok := requesttrace.FromContext(req.Context())
		require.True(t, ok)
		require.Equal(t, requesttrace.ActorKindUser, audit.ActorKind)
		require.NotNil(t, audit.UserID)
		require.Equal(t, "user-123", *audit.UserID)
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer header.eyJ1aWQiOiJ1c2VyLTEyMyIsImZpcmViYXNlIjp7InRlbmFudCI6ImRldi1hY21lIn19.signature")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestRequestTraceAnonymous(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(RequestTrace)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		audit, ok := requesttrace.FromContext(req.Context())
		require.True(t, ok)
		require.Equal(t, requesttrace.ActorKindAnonymous, audit.ActorKind)
		require.Nil(t, audit.UserID)
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}
