import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, FolderOpen, Trash2, ChevronRight, BarChart2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { getProjects, createProject, deleteProject } from '../api'
import { Modal, PageLoader, EmptyState, Spinner, FieldError } from '../components/ui'
import type { ApiError } from '../types'

export default function ProjectsPage() {
  const [showCreate, setShowCreate] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: projects, isLoading, isError } = useQuery({
    queryKey: ['projects'],
    queryFn: getProjects,
  })

  const createMutation = useMutation({
    mutationFn: createProject,
    onSuccess: (project) => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      setShowCreate(false)
      setName('')
      setDescription('')
      setFieldErrors({})
      toast.success('Project created!')
      navigate(`/projects/${project.id}`)
    },
    onError: (err: unknown) => {
      const body = (err as { response?: { data?: ApiError } })?.response?.data
      if (body?.fields) setFieldErrors(body.fields)
      else toast.error(body?.error ?? 'Failed to create project')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteProject,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      toast.success('Project deleted')
      setDeletingId(null)
    },
    onError: () => toast.error('Failed to delete project'),
  })

  const handleCreate = (e: FormEvent) => {
    e.preventDefault()
    setFieldErrors({})
    createMutation.mutate({ name, description: description || undefined })
  }

  if (isLoading) return <PageLoader />

  if (isError) return (
    <div className="text-center py-20 text-red-500">Failed to load projects. Please refresh.</div>
  )

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Projects</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
            {projects?.length ?? 0} project{projects?.length !== 1 ? 's' : ''}
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className="btn-primary">
          <Plus className="w-4 h-4" /> New Project
        </button>
      </div>

      {/* Projects grid */}
      {projects?.length === 0 ? (
        <EmptyState
          icon={<FolderOpen className="w-16 h-16" />}
          title="No projects yet"
          description="Create your first project to start tracking tasks."
          action={
            <button onClick={() => setShowCreate(true)} className="btn-primary">
              <Plus className="w-4 h-4" /> New Project
            </button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {projects?.map((project) => (
            <div
              key={project.id}
              className="card p-5 hover:shadow-md transition-shadow cursor-pointer group"
              onClick={() => navigate(`/projects/${project.id}`)}
            >
              <div className="flex items-start justify-between mb-3">
                <div className="w-9 h-9 rounded-lg bg-sky-100 dark:bg-sky-900/40 flex items-center justify-center flex-shrink-0">
                  <FolderOpen className="w-5 h-5 text-sky-600 dark:text-sky-400" />
                </div>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onClick={(e) => { e.stopPropagation(); navigate(`/projects/${project.id}`) }}
                    className="btn-ghost p-1.5 rounded-lg"
                    title="View stats"
                  >
                    <BarChart2 className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={(e) => { e.stopPropagation(); setDeletingId(project.id) }}
                    className="btn-ghost p-1.5 rounded-lg text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                    title="Delete"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>

              <h3 className="font-semibold text-slate-900 dark:text-white mb-1 truncate">{project.name}</h3>
              {project.description && (
                <p className="text-sm text-slate-500 dark:text-slate-400 line-clamp-2 mb-3">{project.description}</p>
              )}

              <div className="flex items-center justify-between mt-auto pt-2 border-t border-slate-100 dark:border-slate-700">
                <span className="text-xs text-slate-400 dark:text-slate-500">
                  {new Date(project.created_at).toLocaleDateString()}
                </span>
                <ChevronRight className="w-4 h-4 text-slate-400 group-hover:text-sky-500 transition-colors" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create modal */}
      <Modal open={showCreate} onClose={() => { setShowCreate(false); setFieldErrors({}) }} title="New Project">
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="label">Project name *</label>
            <input
              className="input"
              placeholder="e.g. Website Redesign"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
            <FieldError message={fieldErrors.name} />
          </div>
          <div>
            <label className="label">Description</label>
            <textarea
              className="input resize-none"
              rows={3}
              placeholder="What's this project about?"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={() => setShowCreate(false)} className="btn-secondary">
              Cancel
            </button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">
              {createMutation.isPending ? <Spinner className="h-4 w-4" /> : 'Create Project'}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete confirm modal */}
      <Modal open={!!deletingId} onClose={() => setDeletingId(null)} title="Delete Project" size="sm">
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-5">
          Are you sure? This will permanently delete the project and all its tasks.
        </p>
        <div className="flex justify-end gap-2">
          <button onClick={() => setDeletingId(null)} className="btn-secondary">Cancel</button>
          <button
            onClick={() => deletingId && deleteMutation.mutate(deletingId)}
            disabled={deleteMutation.isPending}
            className="btn-danger"
          >
            {deleteMutation.isPending ? <Spinner className="h-4 w-4" /> : 'Delete'}
          </button>
        </div>
      </Modal>
    </div>
  )
}
