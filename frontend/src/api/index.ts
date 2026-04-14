import api from './client'
import type { AuthResponse, Project, ProjectWithTasks, Task, ProjectStats } from '../types'

// ── Auth ──────────────────────────────────────────────────────────────────────
export const register = (data: { name: string; email: string; password: string }) =>
  api.post<AuthResponse>('/auth/register', data).then((r) => r.data)

export const login = (data: { email: string; password: string }) =>
  api.post<AuthResponse>('/auth/login', data).then((r) => r.data)

// ── Projects ──────────────────────────────────────────────────────────────────
export const getProjects = () =>
  api.get<{ projects: Project[] }>('/projects').then((r) => r.data.projects ?? [])

export const getProject = (id: string) =>
  api.get<ProjectWithTasks>(`/projects/${id}`).then((r) => r.data)

export const createProject = (data: { name: string; description?: string }) =>
  api.post<Project>('/projects', data).then((r) => r.data)

export const updateProject = (id: string, data: { name?: string; description?: string }) =>
  api.patch<Project>(`/projects/${id}`, data).then((r) => r.data)

export const deleteProject = (id: string) => api.delete(`/projects/${id}`)

export const getProjectStats = (id: string) =>
  api.get<ProjectStats>(`/projects/${id}/stats`).then((r) => r.data)

// ── Tasks ─────────────────────────────────────────────────────────────────────
export const getTasks = (
  projectId: string,
  params?: { status?: string; assignee?: string }
) =>
  api
    .get<{ tasks: Task[] }>(`/projects/${projectId}/tasks`, { params })
    .then((r) => r.data.tasks ?? [])

export interface CreateTaskPayload {
  title: string
  description?: string
  priority?: string
  assignee_id?: string | null
  due_date?: string | null
}

export interface UpdateTaskPayload {
  title?: string
  description?: string
  status?: string
  priority?: string
  assignee_id?: string | null
  due_date?: string | null
}

export const createTask = (projectId: string, data: CreateTaskPayload) =>
  api.post<Task>(`/projects/${projectId}/tasks`, data).then((r) => r.data)

export const updateTask = (id: string, data: UpdateTaskPayload) =>
  api.patch<Task>(`/tasks/${id}`, data).then((r) => r.data)

export const deleteTask = (id: string) => api.delete(`/tasks/${id}`)
