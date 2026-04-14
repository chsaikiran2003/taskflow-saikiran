package tasks

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Task struct {
	ID          uuid.UUID  `db:"id"          json:"id"`
	Title       string     `db:"title"        json:"title"`
	Description *string    `db:"description"  json:"description"`
	Status      string     `db:"status"       json:"status"`
	Priority    string     `db:"priority"     json:"priority"`
	ProjectID   uuid.UUID  `db:"project_id"   json:"project_id"`
	AssigneeID  *uuid.UUID `db:"assignee_id"  json:"assignee_id"`
	CreatorID   uuid.UUID  `db:"creator_id"   json:"creator_id"`
	DueDate     *string    `db:"due_date"     json:"due_date"`
	CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"   json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Priority    string     `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	DueDate     *string    `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	Priority    *string    `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	DueDate     *string    `json:"due_date"`
}

var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")

var validStatuses = map[string]bool{"todo": true, "in_progress": true, "done": true}
var validPriorities = map[string]bool{"low": true, "medium": true, "high": true}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID, statusFilter, assigneeFilter string) ([]Task, error) {
	query := "SELECT * FROM tasks WHERE project_id=$1"
	args := []any{projectID}

	argIdx := 2
	if statusFilter != "" {
		query += " AND status=$" + itoa(argIdx)
		args = append(args, statusFilter)
		argIdx++
	}
	if assigneeFilter != "" {
		query += " AND assignee_id=$" + itoa(argIdx)
		args = append(args, assigneeFilter)
	}
	query += " ORDER BY created_at DESC"

	var taskList []Task
	if err := s.db.SelectContext(ctx, &taskList, query, args...); err != nil {
		slog.Error("listing tasks", "err", err)
		return nil, err
	}
	if taskList == nil {
		taskList = []Task{}
	}
	return taskList, nil
}

func (s *Service) Create(ctx context.Context, projectID, creatorID uuid.UUID, req CreateTaskRequest) (*Task, map[string]string) {
	fields := map[string]string{}
	if req.Title == "" {
		fields["title"] = "is required"
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	} else if !validPriorities[priority] {
		fields["priority"] = "must be low, medium, or high"
	}
	if len(fields) > 0 {
		return nil, fields
	}

	// Cast due_date to text so PostgreSQL accepts ISO string or null
	task := &Task{}
	err := s.db.QueryRowxContext(ctx,
		`INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, creator_id, due_date)
		 VALUES (gen_random_uuid(), $1, $2, 'todo', $3, $4, $5, $6, $7::date)
		 RETURNING *`,
		req.Title, req.Description, priority, projectID, req.AssigneeID, creatorID, req.DueDate,
	).StructScan(task)
	if err != nil {
		slog.Error("creating task", "err", err)
		return nil, map[string]string{"_": "could not create task"}
	}

	return task, nil
}

func (s *Service) Update(ctx context.Context, taskID, userID uuid.UUID, req UpdateTaskRequest) (*Task, error) {
	// Fetch task + project owner in a single query for auth check
	var row struct {
		Task
		ProjectOwnerID uuid.UUID `db:"project_owner_id"`
	}
	err := s.db.QueryRowxContext(ctx, `
		SELECT t.*, p.owner_id AS project_owner_id
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1`, taskID).StructScan(&row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Authorization: only the task's project owner or creator may update
	if row.CreatorID != userID && row.ProjectOwnerID != userID {
		return nil, ErrForbidden
	}

	task := row.Task
	if req.Title != nil && *req.Title != "" {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Status != nil && validStatuses[*req.Status] {
		task.Status = *req.Status
	}
	if req.Priority != nil && validPriorities[*req.Priority] {
		task.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		task.AssigneeID = req.AssigneeID
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	updated := &Task{}
	err = s.db.QueryRowxContext(ctx,
		`UPDATE tasks
		 SET title=$1, description=$2, status=$3, priority=$4,
		     assignee_id=$5, due_date=$6::date, updated_at=now()
		 WHERE id=$7
		 RETURNING *`,
		task.Title, task.Description, task.Status, task.Priority,
		task.AssigneeID, task.DueDate, taskID,
	).StructScan(updated)
	if err != nil {
		slog.Error("updating task", "err", err)
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, taskID, userID uuid.UUID) error {
	var row struct {
		CreatorID uuid.UUID `db:"creator_id"`
		OwnerID   uuid.UUID `db:"owner_id"`
	}
	err := s.db.GetContext(ctx, &row, `
		SELECT t.creator_id, p.owner_id
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1`, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if row.CreatorID != userID && row.OwnerID != userID {
		return ErrForbidden
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id=$1", taskID)
	return err
}

// itoa converts an int to string without importing strconv at call sites
func itoa(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return string(digits[i])
	}
	return string(digits[i/10]) + string(digits[i%10])
}
