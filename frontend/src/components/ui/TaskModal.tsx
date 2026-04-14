import { useState, useEffect, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { createTask, updateTask, type CreateTaskPayload, type UpdateTaskPayload } from '../../api'
import { Modal, Spinner, FieldError } from './index'
import type { Task, ApiError } from '../../types'

interface TaskModalProps {
  open: boolean
  onClose: () => void
  projectId: string
  task?: Task | null
  projectMembers?: Array<{ id: string; name: string }>
}

export function TaskModal({ open, onClose, projectId, task, projectMembers = [] }: TaskModalProps) {
  const isEdit = !!task
  const queryClient = useQueryClient()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState('todo')
  const [priority, setPriority] = useState('medium')
  const [assigneeId, setAssigneeId] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  // Reset form whenever modal opens or the task being edited changes
  useEffect(() => {
    if (open) {
      setTitle(task?.title ?? '')
      setDescription(task?.description ?? '')
      setStatus(task?.status ?? 'todo')
      setPriority(task?.priority ?? 'medium')
      setAssigneeId(task?.assignee_id ?? '')
      setDueDate(task?.due_date ?? '')
      setFieldErrors({})
    }
  }, [open, task])

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['project', projectId] })
    queryClient.invalidateQueries({ queryKey: ['tasks', projectId] })
  }

  const createMutation = useMutation({
    mutationFn: (data: CreateTaskPayload) => createTask(projectId, data),
    onSuccess: () => { invalidate(); onClose(); toast.success('Task created!') },
    onError: (err: unknown) => {
      const body = (err as { response?: { data?: ApiError } })?.response?.data
      if (body?.fields) setFieldErrors(body.fields)
      else toast.error(body?.error ?? 'Failed to create task')
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: UpdateTaskPayload) => updateTask(task!.id, data),
    onSuccess: () => { invalidate(); onClose(); toast.success('Task updated!') },
    onError: (err: unknown) => {
      const body = (err as { response?: { data?: ApiError } })?.response?.data
      if (body?.fields) setFieldErrors(body.fields)
      else toast.error('Failed to update task')
    },
  })

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    setFieldErrors({})

    if (!title.trim()) {
      setFieldErrors({ title: 'is required' })
      return
    }

    if (isEdit) {
      const payload: UpdateTaskPayload = {
        title: title.trim(),
        description: description.trim() || undefined,
        status,
        priority,
        assignee_id: assigneeId || null,
        due_date: dueDate || null,
      }
      updateMutation.mutate(payload)
    } else {
      const payload: CreateTaskPayload = {
        title: title.trim(),
        description: description.trim() || undefined,
        priority,
        assignee_id: assigneeId || null,
        due_date: dueDate || null,
      }
      createMutation.mutate(payload)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Modal open={open} onClose={onClose} title={isEdit ? 'Edit Task' : 'New Task'} size="lg">
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Title */}
        <div>
          <label className="label">Title *</label>
          <input
            className="input"
            placeholder="What needs to be done?"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            autoFocus
          />
          <FieldError message={fieldErrors.title} />
        </div>

        {/* Description */}
        <div>
          <label className="label">Description</label>
          <textarea
            className="input resize-none"
            rows={3}
            placeholder="Add more context..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        {/* Priority + Due date */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="label">Priority</label>
            <select className="input" value={priority} onChange={(e) => setPriority(e.target.value)}>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </div>
          <div>
            <label className="label">Due date</label>
            <input
              className="input"
              type="date"
              value={dueDate}
              onChange={(e) => setDueDate(e.target.value)}
            />
          </div>
        </div>

        {/* Status — only shown when editing an existing task */}
        {isEdit && (
          <div>
            <label className="label">Status</label>
            <select className="input" value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="todo">To Do</option>
              <option value="in_progress">In Progress</option>
              <option value="done">Done</option>
            </select>
          </div>
        )}

        {/* Assignee */}
        <div>
          <label className="label">Assignee</label>
          <select className="input" value={assigneeId} onChange={(e) => setAssigneeId(e.target.value)}>
            <option value="">Unassigned</option>
            {projectMembers.map((m) => (
              <option key={m.id} value={m.id}>{m.name}</option>
            ))}
          </select>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2">
          <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
          <button type="submit" disabled={isPending} className="btn-primary">
            {isPending ? <Spinner className="h-4 w-4" /> : isEdit ? 'Save changes' : 'Create Task'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
