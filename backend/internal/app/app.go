package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Swek09/lct-fcp-2026/backend/internal/assessment"
	"github.com/Swek09/lct-fcp-2026/backend/internal/db"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	DatabaseURL, RedisURL, AppURL, Env, SMTPHost, SMTPPort, SMTPFrom, SMTPUser, SMTPPassword string
	Demo                                                                                     bool
}

func EnvConfig() Config {
	get := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return Config{DatabaseURL: get("DATABASE_URL", "postgres://fsp:fsp_local_change_me@localhost:5432/fsp?sslmode=disable"), RedisURL: get("REDIS_URL", "redis://localhost:6379/0"), AppURL: strings.TrimRight(get("APP_URL", "http://localhost:8080"), "/"), Env: get("APP_ENV", "development"), SMTPHost: get("SMTP_HOST", "localhost"), SMTPPort: get("SMTP_PORT", "1025"), SMTPFrom: get("SMTP_FROM", "platform@fsp.local"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), Demo: os.Getenv("DEMO_SEED") == "true"}
}

type App struct {
	DB     *pgxpool.Pool
	Q      *db.Queries
	Redis  *redis.Client
	Config Config
	API    huma.API
	// Integration runs isolate counters without clearing shared Redis data.
	rateLimitPrefix string
}
type Response[T any] struct{ Body T }

func ok[T any](v T) *Response[T] { return &Response[T]{Body: v} }

type Message struct {
	Message string `json:"message"`
}

func msg(s string) *Response[Message] { return ok(Message{s}) }
func internal(err error) error {
	slog.Error("request failed", "error", err)
	return huma.Error500InternalServerError("Не удалось выполнить операцию. Попробуйте ещё раз.")
}
func New(ctx context.Context, cfg Config) (*App, error) {
	u, err := url.Parse(cfg.AppURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid APP_URL")
	}
	if cfg.Env == "production" && (cfg.Demo || u.Scheme != "https") {
		return nil, fmt.Errorf("production requires HTTPS and DEMO_SEED=false")
	}
	p, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		p.Close()
		return nil, err
	}
	r := redis.NewClient(opts)
	if err = r.Ping(ctx).Err(); err != nil {
		p.Close()
		_ = r.Close()
		return nil, err
	}
	return &App{DB: p, Q: db.New(p), Redis: r, Config: cfg}, nil
}
func (a *App) Close() { a.DB.Close(); _ = a.Redis.Close() }
func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
				if r.Header.Get("X-Requested-With") != "fsp" || !a.acceptsOrigin(r.Header.Get("Origin")) {
					http.Error(w, "Недопустимый источник запроса", 403)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Use(a.sessionMiddleware)
	cfg := huma.DefaultConfig("ФСП · Карьера API", "0.2.0")
	cfg.OpenAPIPath = "/api/openapi"
	cfg.DocsPath = "/api/docs"
	cfg.SchemasPath = "/api/schemas"
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{"session": {Type: "apiKey", In: "cookie", Name: "fsp_session"}}
	api := humachi.New(r, cfg)
	a.API = api
	huma.Get(api, "/api/health", func(ctx context.Context, in *struct{}) (*Response[map[string]string], error) {
		if err := a.DB.Ping(ctx); err != nil {
			return nil, huma.Error503ServiceUnavailable("База недоступна")
		}
		if err := a.Redis.Ping(ctx).Err(); err != nil {
			return nil, huma.Error503ServiceUnavailable("Сессии недоступны")
		}
		return ok(map[string]string{"status": "ok"}), nil
	})
	huma.Get(api, "/api/config", func(ctx context.Context, in *struct{}) (*Response[map[string]any], error) {
		return ok(map[string]any{"demo": a.Config.Demo, "grade_cooldown_days": 90, "retry_hours": 24, "test_minutes": 20, "test_questions": assessment.QuestionCount, "test_pass_score": assessment.PassScore, "test_version": assessment.Version, "salary_period": "month", "salary_tax": "gross"}), nil
	})
	a.authRoutes(api)
	a.profileRoutes(api)
	a.assessmentRoutes(api)
	a.matchingRoutes(api)
	a.invitationRoutes(api)
	a.challengeRoutes(api)
	a.vacancyRoutes(api)
	a.evidenceRoutes(api)
	a.overviewRoutes(api)
	return r
}

func (a *App) acceptsOrigin(origin string) bool {
	if origin == "" || origin == a.Config.AppURL {
		return true
	}
	if a.Config.Env != "development" {
		return false
	}
	// Local browser URLs may use different loopback names. Keep this exception
	// limited to development and the configured scheme and port.
	parseLocal := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Opaque != "" || u.Path != "" || u.RawQuery != "" || u.ForceQuery || strings.Contains(raw, "#") {
			return nil
		}
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return u
		default:
			return nil
		}
	}
	configured, requested := parseLocal(a.Config.AppURL), parseLocal(origin)
	return configured != nil && requested != nil && configured.Scheme == requested.Scheme && configured.Port() == requested.Port()
}

func register[I, O any](api huma.API, method, path, summary string, fn func(context.Context, *I) (*O, error)) {
	huma.Register(api, huma.Operation{OperationID: strings.ToLower(method) + "-" + strings.NewReplacer("/", "-", "{", "", "}", "").Replace(path), Method: method, Path: path, Summary: summary, Errors: []int{400, 401, 403, 404, 409, 422, 429, 500, 503}, Security: []map[string][]string{{"session": {}}}}, fn)
}
func readJSON[T any](ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, query string, args ...any) (T, error) {
	var v T
	var b []byte
	err := db.QueryRow(ctx, query, args...).Scan(&b)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(b, &v)
	return v, err
}
func isMissing(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
func (a *App) MailLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.deliverMail(ctx); err != nil {
				slog.Warn("email delivery pending", "error", err)
			}
		}
	}
}
