-- Seed data for TaskFlow
-- Test credentials: test@example.com / password123
-- Hash generated with bcrypt cost 12 (Go-compatible $2a$ prefix)

INSERT INTO users (id, name, email, password) VALUES
  ('00000000-0000-0000-0000-000000000001',
   'Test User',
   'test@example.com',
   '$2a$12$5lKkna9Ut8gKQFwTwSAMGenklUaaZrxN7kZRMtM03VpBm2d16wb2O')
ON CONFLICT (email) DO NOTHING;

INSERT INTO projects (id, name, description, owner_id) VALUES
  ('00000000-0000-0000-0000-000000000010',
   'TaskFlow Launch',
   'Getting the TaskFlow product ready for launch',
   '00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, creator_id, due_date) VALUES
  ('00000000-0000-0000-0000-000000000100',
   'Set up CI/CD pipeline',
   'Configure GitHub Actions for automated testing and deployment',
   'done', 'high',
   '00000000-0000-0000-0000-000000000010',
   '00000000-0000-0000-0000-000000000001',
   '00000000-0000-0000-0000-000000000001',
   '2026-04-10'),
  ('00000000-0000-0000-0000-000000000101',
   'Write API documentation',
   'Document all REST endpoints with request/response examples',
   'in_progress', 'medium',
   '00000000-0000-0000-0000-000000000010',
   '00000000-0000-0000-0000-000000000001',
   '00000000-0000-0000-0000-000000000001',
   '2026-04-15'),
  ('00000000-0000-0000-0000-000000000102',
   'Design landing page',
   'Create mockups and implement the marketing landing page',
   'todo', 'low',
   '00000000-0000-0000-0000-000000000010',
   NULL,
   '00000000-0000-0000-0000-000000000001',
   '2026-04-20')
ON CONFLICT (id) DO NOTHING;
