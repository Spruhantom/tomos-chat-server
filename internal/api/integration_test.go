package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthenticationFlowWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("TOMCHAT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TOMCHAT_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	defer adminPool.Close()

	schema := "tomchat_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	}()

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	testPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect to isolated test schema: %v", err)
	}
	defer testPool.Close()

	handler := New(testPool, true)
	if err := handler.Migrate(); err != nil {
		t.Fatalf("run schema migrations: %v", err)
	}
	server := handler.Handler()

	registerBody := []byte(`{"username":"tomtest","password":"a-long-test-password"}`)
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	server.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", registerResponse.Code, registerResponse.Body.String())
	}

	loginBody := []byte(`{"username":"tomtest","password":"a-long-test-password"}`)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	server.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
	var session authResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if session.Token == "" || session.User.Username != "tomtest" {
		t.Fatalf("unexpected login response: %+v", session)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+session.Token)
	meResponse := httptest.NewRecorder()
	server.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("current user: expected 200, got %d", meResponse.Code)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+session.Token)
	logoutResponse := httptest.NewRecorder()
	server.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d", logoutResponse.Code)
	}

	reuseRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	reuseRequest.Header.Set("Authorization", "Bearer "+session.Token)
	reuseResponse := httptest.NewRecorder()
	server.ServeHTTP(reuseResponse, reuseRequest)
	if reuseResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session: expected 401, got %d", reuseResponse.Code)
	}
}
