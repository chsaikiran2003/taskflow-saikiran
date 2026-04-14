package projects

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Project struct {
	ID          uuid.UUID `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description"`
	OwnerID     uuid.UUID `db:"owner_id"    json:"owner_id"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// Task mirrors tasks.Task — kept here so projects package has no import cycle
type Task struct {
	ID          uuid.UUID  `db:"id"          json:"id"`
	Title       string     `db:"title"       json:"title"`
	Description *string    `db:"description" json:"description"`
	Status      string     `db:"status"      json:"status"`
	Priority    string     `db:"priority"    json:"priority"`
	ProjectID   uuid.UUID  `db:"project_id"  json:"project_id"`
	AssigneeID  *uuid.UUID `db:"assignee_id" json:"assignee_id"`
	CreatorID   uuid.UUID  `db:"creator_id"  json:"creator_id"`
	DueDate     *string    `db:"due_date"    json:"due_date"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"  json:"updated_at"`
}

type ProjectWithTasks struct {
	Project
	Tasks []Task `json:"tasks"`
}

type CreateProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type ProjectStats struct {
	ByStatus   map[string]int         `json:"by_status"`
	ByAssignee map[string]AssigneeStats `json:"by_assignee"`
}

type AssigneeStats struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	var projectList []Project
	err := s.db.SelectContext(ctx, &projectList, `
		SELECT DISTINCT p.*
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		slog.Error("listing projects", "err", err)
		return nil, err
	}
	if projectList == nil {
		projectList = []Project{}
	}
	return projectList, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateProjectRequest) (*Project, map[string]string) {
	fields := map[string]string{}
	if req.Name == "" {
		fields["name"] = "is required"
	}
	if len(fields) > 0 {
		return nil, fields
	}

	project := &Project{}
	err := s.db.QueryRowxContext(ctx,
		`INSERT INTO projects (id, name, description, owner_id)
		 VALUES (gen_random_uuid(), $1, $2, $3)
		 RETURNING *`,
		req.Name, req.Description, userID,
	).StructScan(project)
	if err != nil {
		slog.Error("creating project", "err", err)
		return nil, map[string]string{"_": "could not create project"}
	}
	return project, nil
}

func (s *Service) Get(ctx context.Context, projectID uuid.UUID) (*ProjectWithTasks, error) {
	project := &Project{}
	if err := s.db.GetContext(ctx, project, "SELECT * FROM projects WHERE id=$1", projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var taskList []Task
	if err := s.db.SelectContext(ctx, &taskList,
		"SELECT * FROM tasks WHERE project_id=$1 ORDER BY created_at DESC", projectID); err != nil {
		slog.Error("fetching tasks for project", "err", err)
		taskList = []Task{}
	}
	if taskList == nil {
		taskList = []Task{}
	}

	return &ProjectWithTasks{Project: *project, Tasks: taskList}, nil
}

func (s *Service) Update(ctx context.Context, projectID, userID uuid.UUID, req UpdateProjectRequest) (*Project, error) {
	project := &Project{}
	if err := s.db.GetContext(ctx, project, "SELECT * FROM projects WHERE id=$1", projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if project.OwnerID != userID {
		return nil, ErrForbidden
	}

	if req.Name != nil && *req.Name != "" {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = req.Description
	}

	updated := &Project{}
	err := s.db.QueryRowxContext(ctx,
		`UPDATE projects SET name=$1, description=$2 WHERE id=$3 RETURNING *`,
		project.Name, project.Description, projectID,
	).StructScan(updated)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, projectID, userID uuid.UUID) error {
	project := &Project{}
	if err := s.db.GetContext(ctx, project, "SELECT * FROM projects WHERE id=$1", projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if project.OwnerID != userID {
		return ErrForbidden
	}

	_, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id=$1", projectID)
	return err
}

func (s *Service) Stats(ctx context.Context, projectID uuid.UUID) (*ProjectStats, error) {
	// Task counts by status
	rows, err := s.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM tasks WHERE project_id=$1 GROUP BY status`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byStatus := map[string]int{"todo": 0, "in_progress": 0, "done": 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			byStatus[status] = count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Task counts by assignee
	arows, err := s.db.QueryContext(ctx, `
		SELECT u.id::text, u.name, COUNT(t.id)
		FROM tasks t
		JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id=$1 AND t.assignee_id IS NOT NULL
		GROUP BY u.id, u.name
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer arows.Close()

	byAssignee := map[string]AssigneeStats{}
	for arows.Next() {
		var id, name string
		var count int
		if err := arows.Scan(&id, &name, &count); err == nil {
			byAssignee[id] = AssigneeStats{Name: name, Count: count}
		}
	}
	if err := arows.Err(); err != nil {
		return nil, err
	}

	return &ProjectStats{ByStatus: byStatus, ByAssignee: byAssignee}, nil
}
