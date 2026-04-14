import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, ChevronLeft, ClipboardList, Pencil, BarChart2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { getProject, updateProject, getProjectStats } from '../api'
import { PageLoader, EmptyState, Modal, Spinner, StatusBadge, FieldError } from '../components/ui'
import { TaskModal } from '../components/ui/TaskModal'
import { TaskCard } from '../components/ui/TaskCard'
import { useAuth } from '../context/AuthContext'
import type { TaskStatus, ApiError } from '../types'

const STATUS_COLUMNS: { key: TaskStatus; label: string }[] = [
  { key: 'todo', label: 'To Do' },
  { key: 'in_progress', label: 'In Progress' },
  { key: 'done', label: 'Done' },
]

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { user } = useAuth()
  const queryClient = useQueryClient()

  const [showCreate, setShowCreate] = useState(false)
  const [showEdit, setShowEdit] = useState(false)
  const [showStats, setShowStats] = useState(false)
  const [filterStatus, setFilterStatus] = useState<string>('')
  const [editName, setEditName] = useState('')
  const [editDesc, setEditDesc] = useState('')
  const [editErrors, setEditErrors] = useState<Record<string, string>>({})

  const { data: project, isLoading, isError } = useQuery({
    queryKey: ['project', id],
    queryFn: () => getProject(id!),
    enabled: !!id,
  })

  const { data: stats } = useQuery({
    queryKey: ['stats', id],
    queryFn: () => getProjectStats(id!),
    enabled: showStats && !!id,
  })

  const editMutation = useMutation({
    mutationFn: (data: { name?: string; description?: string }) => updateProject(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', id] })
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      setShowEdit(false)
      toast.success('Project updated!')
    },
    onError: (err: unknown) => {
      const body = (err as { response?: { data?: ApiError } })?.response?.data
      if (body?.fields) setEditErrors(body.fields)
      else toast.error('Failed to update project')
    },
  })

  const openEdit = () => {
    setEditName(project?.name ?? '')
    setEditDesc(project?.description ?? '')
    setEditErrors({})
    setShowEdit(true)
  }

  if (isLoading) return <PageLoader />
  if (isError || !project) return (
    <div className="text-center py-20 text-red-500 dark:text-red-400">
      Project not found or failed to load.
    </div>
  )

  const isOwner = project.owner_id === user?.id
  const tasks = project.tasks ?? []
  const filtered = filterStatus ? tasks.filter((t) => t.status === filterStatus) : tasks

  // Build member list: current user + anyone who is an assignee on a task
  const memberMap = new Map<string, string>()
  if (user) memberMap.set(user.id, user.name)
  tasks.forEach((t) => {
    if (t.assignee_id && !memberMap.has(t.assignee_id)) {
      // We don't have their name from the API unless they're the current user.
      // Show "Team Member" as placeholder — a /users endpoint would fix this properly.
      memberMap.set(t.assignee_id, 'Team Member')
    }
  })
  const projectMembers = Array.from(memberMap.entries()).map(([mid, name]) => ({ id: mid, name }))

  return (
    <div>
      {/* Breadcrumb */}
      <div className="mb-5">
        <Link
          to="/projects"
          className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-sky-600 dark:hover:text-sky-400 transition-colors"
        >
          <ChevronLeft className="w-4 h-4" />
          All Projects
        </Link>
      </div>

      {/* Project header */}
      <div className="flex items-start justify-between mb-6 gap-4 flex-wrap">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white truncate">
            {project.name}
          </h1>
          {project.description && (
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{project.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2 flex-shrink-0 flex-wrap">
          <button onClick={() => setShowStats(true)} className="btn-secondary">
            <BarChart2 className="w-4 h-4" />
            <span className="hidden sm:inline">Stats</span>
          </button>
          {isOwner && (
            <button onClick={openEdit} className="btn-secondary">
              <Pencil className="w-4 h-4" />
              <span className="hidden sm:inline">Edit</span>
            </button>
          )}
          <button onClick={() => setShowCreate(true)} className="btn-primary">
            <Plus className="w-4 h-4" />
            Add Task
          </button>
        </div>
      </div>

      {/* Status filter pills */}
      <div className="flex items-center gap-2 mb-5 flex-wrap">
        <span className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide">
          Filter:
        </span>
        {(['', 'todo', 'in_progress', 'done'] as const).map((s) => (
          <button
            key={s}
            onClick={() => setFilterStatus(s)}
            className={`px-3 py-1 rounded-full text-xs font-medium transition-all ${
              filterStatus === s
                ? 'bg-sky-600 text-white shadow-sm'
                : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-600'
            }`}
          >
            {s === ''
              ? 'All'
              : s === 'in_progress'
              ? 'In Progress'
              : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
        <span className="ml-auto text-xs text-slate-400 dark:text-slate-500">
          {filtered.length} task{filtered.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Kanban board or empty state */}
      {tasks.length === 0 ? (
        <EmptyState
          icon={<ClipboardList className="w-16 h-16" />}
          title="No tasks yet"
          description="Add your first task to get started on this project."
          action={
            <button onClick={() => setShowCreate(true)} className="btn-primary">
              <Plus className="w-4 h-4" />
              Add Task
            </button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {STATUS_COLUMNS.map(({ key, label }) => {
            const col = filtered.filter((t) => t.status === key)
            return (
              <div
                key={key}
                className="bg-slate-100 dark:bg-slate-800/50 rounded-xl p-3 min-h-[120px]"
              >
                <div className="flex items-center justify-between mb-3 px-1">
                  <div className="flex items-center gap-2">
                    <StatusBadge status={key} />
                    <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                      {label}
                    </span>
                  </div>
                  <span className="text-xs font-semibold text-slate-400 dark:text-slate-500 bg-slate-200 dark:bg-slate-700 px-1.5 py-0.5 rounded-full">
                    {col.length}
                  </span>
                </div>

                <div className="space-y-2">
                  {col.length === 0 ? (
                    <div className="text-center py-6 text-xs text-slate-400 dark:text-slate-500 italic">
                      No tasks
                    </div>
                  ) : (
                    col.map((task) => (
                      <TaskCard
                        key={task.id}
                        task={task}
                        projectId={id!}
                        projectMembers={projectMembers}
                      />
                    ))
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* ── Modals ── */}

      {/* Create task */}
      <TaskModal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        projectId={id!}
        projectMembers={projectMembers}
      />

      {/* Edit project */}
      <Modal open={showEdit} onClose={() => setShowEdit(false)} title="Edit Project">
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (!editName.trim()) {
              setEditErrors({ name: 'is required' })
              return
            }
            editMutation.mutate({ name: editName.trim(), description: editDesc.trim() || undefined })
          }}
          className="space-y-4"
        >
          <div>
            <label className="label">Project name *</label>
            <input
              className="input"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              autoFocus
            />
            <FieldError message={editErrors.name} />
          </div>
          <div>
            <label className="label">Description</label>
            <textarea
              className="input resize-none"
              rows={3}
              value={editDesc}
              onChange={(e) => setEditDesc(e.target.value)}
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={() => setShowEdit(false)} className="btn-secondary">
              Cancel
            </button>
            <button type="submit" disabled={editMutation.isPending} className="btn-primary">
              {editMutation.isPending ? <Spinner className="h-4 w-4" /> : 'Save changes'}
            </button>
          </div>
        </form>
      </Modal>

      {/* Stats */}
      <Modal open={showStats} onClose={() => setShowStats(false)} title="Project Stats">
        {!stats ? (
          <div className="flex justify-center py-8">
            <Spinner className="h-8 w-8" />
          </div>
        ) : (
          <div className="space-y-5">
            <div>
              <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">
                Tasks by status
              </h3>
              <div className="grid grid-cols-3 gap-3">
                {Object.entries(stats.by_status).map(([status, count]) => (
                  <div key={status} className="card p-3 text-center">
                    <div className="text-2xl font-bold text-slate-900 dark:text-white mb-1">
                      {count}
                    </div>
                    <StatusBadge status={status} />
                  </div>
                ))}
              </div>
            </div>

            {Object.keys(stats.by_assignee).length > 0 && (
              <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">
                  By assignee
                </h3>
                <div className="space-y-2">
                  {Object.entries(stats.by_assignee).map(([aid, { name, count }]) => (
                    <div
                      key={aid}
                      className="flex items-center justify-between py-2 px-3 bg-slate-50 dark:bg-slate-700/50 rounded-lg"
                    >
                      <span className="text-sm text-slate-700 dark:text-slate-300">{name}</span>
                      <span className="text-sm font-semibold text-slate-900 dark:text-white">
                        {count} task{count !== 1 ? 's' : ''}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}
