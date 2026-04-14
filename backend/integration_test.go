package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/saikiran/taskflow/internal/auth"
	"github.com/saikiran/taskflow/internal/db"
	appMiddleware "github.com/saikiran/taskflow/internal/middleware"
	"github.com/saikiran/taskflow/internal/projects"
	"github.com/saikiran/taskflow/internal/tasks"
)

var testDB *sqlx.DB
var testServer *httptest.Server

func TestMain(m *testing.M) {
	os.Setenv("DB_HOST", getEnv("TEST_DB_HOST", "localhost"))
	os.Setenv("DB_PORT", getEnv("TEST_DB_PORT", "5432"))
	os.Setenv("DB_USER", getEnv("TEST_DB_USER", "taskflow"))
	os.Setenv("DB_PASSWORD", getEnv("TEST_DB_PASSWORD", "taskflow_secret"))
	os.Setenv("DB_NAME", getEnv("TEST_DB_NAME", "taskflow_test"))
	os.Setenv("JWT_SECRET", "test_secret_key_32_chars_minimum!")

	var err error
	testDB, err = db.Connect()
	if err != nil {
		fmt.Printf("SKIP: could not connect to test DB: %v\n", err)
		os.Exit(0)
	}

	if err := db.RunMigrations(testDB.DB); err != nil {
		fmt.Printf("SKIP: migrations failed: %v\n", err)
		os.Exit(0)
	}

	testServer = httptest.NewServer(buildRouter(testDB))
	defer testServer.Close()

	code := m.Run()

	// Cleanup test data
	testDB.Exec("DELETE FROM tasks")
	testDB.Exec("DELETE FROM projects")
	testDB.Exec("DELETE FROM users")

	os.Exit(code)
}

func buildRouter(database *sqlx.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}))

	authSvc := auth.NewService(database)
	projectSvc := projects.NewService(database)
	taskSvc := tasks.NewService(database)

	authH := auth.NewHandler(authSvc)
	projectH := projects.NewHandler(projectSvc)
	taskH := tasks.NewHandler(taskSvc)

	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)

	r.Group(func(r chi.Router) {
		r.Use(appMiddleware.Auth)
		r.Get("/projects", projectH.List)
		r.Post("/projects", projectH.Create)
		r.Get("/projects/{id}", projectH.Get)
		r.Patch("/projects/{id}", projectH.Update)
		r.Delete("/projects/{id}", projectH.Delete)
		r.Get("/projects/{id}/tasks", taskH.List)
		r.Post("/projects/{id}/tasks", taskH.Create)
		r.Patch("/tasks/{id}", taskH.Update)
		r.Delete("/tasks/{id}", taskH.Delete)
	})

	return r
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func doPost(t *testing.T, path string, body any, token string) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", testServer.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func doGet(t *testing.T, path, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", testServer.URL+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var v map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func uniqueEmail() string {
	return fmt.Sprintf("user_%d@test.com", time.Now().UnixNano())
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func registerAndGetToken(t *testing.T, email string) string {
	t.Helper()
	resp := doPost(t, "/auth/register", map[string]string{
		"name": "Test", "email": email, "password": "password123",
	}, "")
	body := decodeBody(t, resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: %d %v", resp.StatusCode, body)
	}
	return body["token"].(string)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestAuth(t *testing.T) {
	email := uniqueEmail()

	t.Run("register_success", func(t *testing.T) {
		resp := doPost(t, "/auth/register", map[string]string{
			"name": "Test User", "email": email, "password": "password123",
		}, "")
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["token"] == nil {
			t.Fatal("expected token in response")
		}
		if body["user"] == nil {
			t.Fatal("expected user in response")
		}
	})

	t.Run("register_duplicate_email_returns_400", func(t *testing.T) {
		resp := doPost(t, "/auth/register", map[string]string{
			"name": "Test User", "email": email, "password": "password123",
		}, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for duplicate, got %d", resp.StatusCode)
		}
	})

	t.Run("register_missing_fields_returns_validation_error", func(t *testing.T) {
		resp := doPost(t, "/auth/register", map[string]string{
			"name": "", "email": "", "password": "short",
		}, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["fields"] == nil {
			t.Fatal("expected fields map in error response")
		}
	})

	t.Run("login_success", func(t *testing.T) {
		resp := doPost(t, "/auth/login", map[string]string{
			"email": email, "password": "password123",
		}, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["token"] == nil {
			t.Fatal("expected token")
		}
	})

	t.Run("login_wrong_password_returns_400", func(t *testing.T) {
		resp := doPost(t, "/auth/login", map[string]string{
			"email": email, "password": "wrongpassword",
		}, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("protected_without_token_returns_401", func(t *testing.T) {
		resp := doGet(t, "/projects", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestProjectsAndTasks(t *testing.T) {
	token := registerAndGetToken(t, uniqueEmail())
	var projectID string

	t.Run("create_project", func(t *testing.T) {
		resp := doPost(t, "/projects", map[string]string{
			"name": "Test Project", "description": "A test project",
		}, token)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		projectID = fmt.Sprintf("%v", body["id"])
		if projectID == "" || projectID == "<nil>" {
			t.Fatal("expected project id")
		}
	})

	t.Run("list_projects_returns_created_project", func(t *testing.T) {
		resp := doGet(t, "/projects", token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		list, ok := body["projects"].([]any)
		if !ok || len(list) == 0 {
			t.Fatal("expected at least one project")
		}
	})

	t.Run("get_project_detail", func(t *testing.T) {
		resp := doGet(t, "/projects/"+projectID, token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("create_task_in_project", func(t *testing.T) {
		resp := doPost(t, "/projects/"+projectID+"/tasks", map[string]string{
			"title": "Test Task", "priority": "high",
		}, token)
		if resp.StatusCode != http.StatusCreated {
			body := decodeBody(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}
		body := decodeBody(t, resp)
		if body["id"] == nil {
			t.Fatal("expected task id in response")
		}
	})

	t.Run("task_missing_title_returns_validation_error", func(t *testing.T) {
		resp := doPost(t, "/projects/"+projectID+"/tasks", map[string]string{
			"priority": "medium",
		}, token)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["error"] != "validation failed" {
			t.Fatalf("expected 'validation failed', got: %v", body["error"])
		}
	})

	t.Run("get_nonexistent_project_returns_404", func(t *testing.T) {
		resp := doGet(t, "/projects/00000000-0000-0000-0000-000000000000", token)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})
}
