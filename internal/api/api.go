package api

import (
	"crypto/rand"
	"crypto/sha256"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

//go:embed schema.sql
var schemaFS embed.FS

const sessionLifetime = 24 * time.Hour

type API struct {
	db *pgxpool.Pool
	registrationEnabled bool
	loginLimiter *loginRateLimiter
}

type user struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Role string `json:"role"`
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	User user `json:"user"`
	Token string `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

func New(db *pgxpool.Pool, registrationEnabled bool) *API {
	return &API{db: db, registrationEnabled: registrationEnabled, loginLimiter: newLoginRateLimiter(10, time.Minute)}
}

func (a *API) Migrate() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil { return err }
	_, err = a.db.Exec(aContext(), string(schema))
	return err
}

func (a *API) BootstrapAdmin(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) || len(password) < 12 || len(password) > 256 {
		return errors.New("bootstrap administrator credentials do not meet policy")
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Serialize bootstrap attempts so simultaneous server starts cannot create
	// more than one initial administrator.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(72190420261106)"); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return tx.Commit(ctx)
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO users (id, username, password_hash, role) VALUES ($1, $2, $3, 'admin')",
		uuid.New(), username, hash); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /api/v1/version", a.version)
	mux.HandleFunc("GET /api/v1/registration", a.registrationStatus)
	mux.HandleFunc("POST /api/v1/auth/register", a.register)
	mux.Handle("POST /api/v1/auth/login", a.loginLimiter.middleware(http.HandlerFunc(a.login)))
	mux.Handle("GET /api/v1/users/me", a.requireAuth(http.HandlerFunc(a.me)))
	mux.Handle("POST /api/v1/auth/logout", a.requireAuth(http.HandlerFunc(a.logout)))
	return securityHeaders(mux)
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status":"ok","service":"tomchat-server"})
}

func (a *API) version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"name":"TomChat Server","version":"0.2.0-dev"})
}

func (a *API) registrationStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"registration_enabled":a.registrationEnabled})
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	if !a.registrationEnabled {
		writeError(w, http.StatusForbidden, "registration_disabled")
		return
	}
	var input credentials
	if !decodeJSON(w, r, &input) { return }
	input.Username = strings.TrimSpace(input.Username)
	if !usernamePattern.MatchString(input.Username) {
		writeError(w, http.StatusBadRequest, "invalid_username")
		return
	}
	if len(input.Password) < 12 || len(input.Password) > 256 {
		writeError(w, http.StatusBadRequest, "password_must_be_12_to_256_characters")
		return
	}
	hash, err := hashPassword(input.Password)
	if err != nil {
		log.Printf("hash registration password: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	id := uuid.New()
	_, err = a.db.Exec(r.Context(),
		"INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)",
		id, input.Username, hash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "username_unavailable")
			return
		}
		log.Printf("register user: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusCreated, user{ID:id.String(), Username:input.Username, Role:"user"})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if !decodeJSON(w, r, &input) { return }
	if len(input.Password) > 256 { writeError(w, http.StatusUnauthorized, "invalid_credentials"); return }
	var u user
	var passwordHash string
	err := a.db.QueryRow(r.Context(),
		"SELECT id::text, username, role, password_hash FROM users WHERE username=$1",
		strings.TrimSpace(input.Username)).Scan(&u.ID, &u.Username, &u.Role, &passwordHash)
	if err != nil || !verifyPassword(input.Password, passwordHash) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	tokenHash := sha256.Sum256([]byte(token))
	expires := time.Now().UTC().Add(sessionLifetime)
	_, err = a.db.Exec(r.Context(),
		"INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)",
		tokenHash[:], u.ID, expires)
	if err != nil {
		log.Printf("create session: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{User:u, Token:token, ExpiresAt:expires})
}

func (a *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "authentication_required")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || len(token) > 256 {
			writeError(w, http.StatusUnauthorized, "invalid_session")
			return
		}
		sum := sha256.Sum256([]byte(token))
		var u user
		err := a.db.QueryRow(r.Context(), `
			SELECT users.id::text, users.username, users.role
			FROM sessions JOIN users ON users.id=sessions.user_id
			WHERE sessions.token_hash=$1 AND sessions.expires_at > NOW()
		`, sum[:]).Scan(&u.ID, &u.Username, &u.Role)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_session")
			return
		}
		ctx := contextWithUser(r, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r)
	if !ok { writeError(w, http.StatusUnauthorized, "authentication_required"); return }
	writeJSON(w, http.StatusOK, u)
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	sum := sha256.Sum256([]byte(token))
	_, err := a.db.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", sum[:])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return false
	}
	return true
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return "argon2id$" + base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 || parts[0] != "argon2id" { return false }
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[1])
	expected, err2 := base64.RawStdEncoding.DecodeString(parts[2])
	if err1 != nil || err2 != nil || len(expected) != 32 { return false }
	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, uint32(len(expected)))
	if len(actual) != len(expected) { return false }
	var diff byte
	for i := range actual { diff |= actual[i] ^ expected[i] }
	return diff == 0
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil { log.Printf("write JSON: %v", err) }
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, errorResponse{Error:code})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// These tiny context wrappers keep the authenticated identity out of request parameters.
type contextKey struct{}
func contextWithUser(r *http.Request, u user) context.Context {
	return context.WithValue(r.Context(), contextKey{}, u)
}
func userFromContext(r *http.Request) (user, bool) {
	u, ok := r.Context().Value(contextKey{}).(user)
	return u, ok
}

func aContext() context.Context { return context.Background() }

