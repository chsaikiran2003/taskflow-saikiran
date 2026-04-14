import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil, Trash2, Calendar, User } from 'lucide-react'
import toast from 'react-hot-toast'
import { updateTask, deleteTask } from '../../api'
import { StatusBadge, PriorityBadge, Modal, Spinner } from './index'
import { TaskModal } from './TaskModal'
import type { Task } from '../../types'

interface TaskCardProps {
  task: Task
  projectId: string
  projectMembers?: Array<{ id: string; name: string }>
}

// Parse "YYYY-MM-DD" safely without timezone shift
function parseLocalDate(dateStr: string): string {
  const [year, month, day] = dateStr.split('-')
  const d = new Date(Number(year), Number(month) - 1, Number(day))
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

export function TaskCard({ task, projectId, projectMembers = [] }: TaskCardProps) {
  const [editing, setEditing] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const queryClient = useQueryClient()

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['project', projectId] })
    queryClient.invalidateQueries({ queryKey: ['tasks', projectId] })
  }

  const nextStatus: Record<string, string> = {
    todo: 'in_progress',
    in_progress: 'done',
    done: 'todo',
  }

  // Optimistic status update
  const statusMutation = useMutation({
    mutationFn: (status: string) => updateTask(task.id, { status }),
    onMutate: async (newStatus) => {
      await queryClient.cancelQueries({ queryKey: ['project', projectId] })
      const prev = queryClient.getQueryData(['project', projectId])
      queryClient.setQueryData(['project', projectId], (old: unknown) => {
        if (!old || typeof old !== 'object') return old
        const o = old as { tasks?: Task[] }
        return {
          ...o,
          tasks: (o.tasks ?? []).map((t: Task) =>
            t.id === task.id ? { ...t, status: newStatus } : t
          ),
        }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(['project', projectId], ctx.prev)
      toast.error('Failed to update status')
    },
    onSettled: invalidate,
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteTask(task.id),
    onSuccess: () => {
      invalidate()
      setConfirming(false)
      toast.success('Task deleted')
    },
    onError: () => toast.error('Failed to delete task'),
  })

  const assignee = projectMembers.find((m) => m.id === task.assignee_id)

  return (
    <>
      <div className="card p-4 hover:shadow-md transition-shadow group">
        <div className="flex items-start gap-3">
          {/* Cycle-status button */}
          <button
            onClick={() => statusMutation.mutate(nextStatus[task.status])}
            disabled={statusMutation.isPending}
            className={`mt-0.5 w-4 h-4 rounded-full border-2 flex-shrink-0 transition-all ${
              task.status === 'done'
                ? 'bg-green-500 border-green-500'
                : task.status === 'in_progress'
                ? 'border-blue-500 bg-blue-100 dark:bg-blue-900/30'
                : 'border-slate-300 dark:border-slate-600'
            }`}
            title={`Mark as ${nextStatus[task.status].replace('_', ' ')}`}
            aria-label={`Mark as ${nextStatus[task.status].replace('_', ' ')}`}
          />

          <div className="flex-1 min-w-0">
            <p
              className={`text-sm font-medium leading-snug mb-2 ${
                task.status === 'done'
                  ? 'line-through text-slate-400'
                  : 'text-slate-900 dark:text-white'
              }`}
            >
              {task.title}
            </p>

            {task.description && (
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-2 line-clamp-2">
                {task.description}
              </p>
            )}

            <div className="flex flex-wrap items-center gap-2">
              <StatusBadge status={task.status} />
              <PriorityBadge priority={task.priority} />

              {task.due_date && (
                <span className="flex items-center gap-1 text-xs text-slate-500 dark:text-slate-400">
                  <Calendar className="w-3 h-3" />
                  {parseLocalDate(task.due_date)}
                </span>
              )}

              {assignee && (
                <span className="flex items-center gap-1 text-xs text-slate-500 dark:text-slate-400">
                  <User className="w-3 h-3" />
                  {assignee.name}
                </span>
              )}
            </div>
          </div>

          {/* Actions — visible on hover */}
          <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
            <button
              onClick={() => setEditing(true)}
              className="btn-ghost p-1 rounded"
              title="Edit task"
              aria-label="Edit task"
            >
              <Pencil className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => setConfirming(true)}
              className="btn-ghost p-1 rounded text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
              title="Delete task"
              aria-label="Delete task"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>

      <TaskModal
        open={editing}
        onClose={() => setEditing(false)}
        projectId={projectId}
        task={task}
        projectMembers={projectMembers}
      />

      <Modal open={confirming} onClose={() => setConfirming(false)} title="Delete Task" size="sm">
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-5">
          Delete "<strong>{task.title}</strong>"? This cannot be undone.
        </p>
        <div className="flex justify-end gap-2">
          <button onClick={() => setConfirming(false)} className="btn-secondary">
            Cancel
          </button>
          <button
            onClick={() => deleteMutation.mutate()}
            disabled={deleteMutation.isPending}
            className="btn-danger"
          >
            {deleteMutation.isPending ? <Spinner className="h-4 w-4" /> : 'Delete'}
          </button>
        </div>
      </Modal>
    </>
  )
}
