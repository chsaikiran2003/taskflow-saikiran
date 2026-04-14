import { Link, useNavigate } from 'react-router-dom'
import { LogOut, Moon, Sun, CheckSquare } from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import { useTheme } from '../../context/ThemeContext'

export function Navbar() {
  const { user, logout } = useAuth()
  const { dark, toggle } = useTheme()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <header className="sticky top-0 z-40 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
        <Link to="/projects" className="flex items-center gap-2 font-semibold text-sky-600 dark:text-sky-400">
          <CheckSquare className="w-5 h-5" />
          <span className="text-slate-900 dark:text-white">TaskFlow</span>
        </Link>

        <div className="flex items-center gap-3">
          <button onClick={toggle} className="btn-ghost p-2 rounded-lg" aria-label="Toggle theme">
            {dark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
          </button>

          {user && (
            <>
              <span className="hidden sm:block text-sm text-slate-600 dark:text-slate-400">
                {user.name}
              </span>
              <button onClick={handleLogout} className="btn-ghost p-2 rounded-lg" aria-label="Logout">
                <LogOut className="w-4 h-4" />
              </button>
            </>
          )}
        </div>
      </div>
    </header>
  )
}
