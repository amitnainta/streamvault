import { NavLink, useNavigate, useLocation } from 'react-router-dom'
import {
  Home, Film, Tv, Music, Shield,
  Library, Users, LogOut, Server, Search
} from 'lucide-react'
import { useAuthStore } from '@/store/authStore'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { Library as LibraryType } from '@/types/api'
import { useState, useEffect } from 'react'

const iconSize = 18

export default function Sidebar() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchVal, setSearchVal] = useState('')

  // Sync search box when navigating away from /search
  useEffect(() => {
    if (!location.pathname.startsWith('/search')) setSearchVal('')
  }, [location.pathname])

  // Debounce navigate to /search
  useEffect(() => {
    if (!searchVal.trim()) return
    const t = setTimeout(() => {
      navigate(`/search?q=${encodeURIComponent(searchVal.trim())}`)
    }, 300)
    return () => clearTimeout(t)
  }, [searchVal, navigate])

  const { data: libraries } = useQuery({
    queryKey: ['libraries'],
    queryFn: () => api.get<LibraryType[]>('/libraries'),
  })

  const libraryIcon = (type: string) => {
    if (type === 'movie') return <Film size={iconSize} />
    if (type === 'show') return <Tv size={iconSize} />
    return <Music size={iconSize} />
  }

  const link = 'flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors'
  const activeLink = 'bg-muted text-foreground'

  return (
    <aside className="w-56 shrink-0 flex flex-col border-r border-border h-full">
      {/* Logo */}
      <div className="px-4 py-5 border-b border-border">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded bg-accent flex items-center justify-center">
            <span className="text-white font-bold text-xs">SV</span>
          </div>
          <span className="font-semibold text-foreground">StreamVault</span>
        </div>
      </div>

      {/* Search */}
      <div className="px-3 py-2 border-b border-border">
        <div className="flex items-center gap-2 px-2 py-1.5 rounded-md bg-muted text-muted-foreground">
          <Search size={14} className="shrink-0" />
          <input
            type="text"
            placeholder="Search…"
            value={searchVal}
            onChange={e => setSearchVal(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && searchVal.trim())
                navigate(`/search?q=${encodeURIComponent(searchVal.trim())}`)
            }}
            className="bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none w-full"
          />
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto p-3 space-y-1">
        <NavLink to="/" end className={({ isActive }) => `${link} ${isActive ? activeLink : ''}`}>
          <Home size={iconSize} /> Home
        </NavLink>

        {libraries && libraries.length > 0 && (
          <div className="pt-3">
            <p className="px-3 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">Libraries</p>
            {libraries.map((lib) => (
              <NavLink
                key={lib.id}
                to={`/library/${lib.id}`}
                className={({ isActive }) => `${link} ${isActive ? activeLink : ''}`}
              >
                {libraryIcon(lib.type)}
                <span className="truncate">{lib.name}</span>
              </NavLink>
            ))}
          </div>
        )}

        {user?.role === 'admin' && (
          <div className="pt-3">
            <p className="px-3 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">Admin</p>
            <NavLink to="/admin" end className={({ isActive }) => `${link} ${isActive ? activeLink : ''}`}>
              <Server size={iconSize} /> Dashboard
            </NavLink>
            <NavLink to="/admin/privacy" className={({ isActive }) => `${link} ${isActive ? activeLink : ''}`}>
              <Shield size={iconSize} /> Privacy & Internet
            </NavLink>
            <NavLink to="/admin/libraries" className={({ isActive }) => `${link} ${isActive ? activeLink : ''}`}>
              <Library size={iconSize} /> Libraries
            </NavLink>
            <NavLink to="/admin/users" className={({ isActive }) => `${link} ${isActive ? activeLink : ''}`}>
              <Users size={iconSize} /> Users
            </NavLink>
          </div>
        )}
      </nav>

      {/* User footer */}
      <div className="p-3 border-t border-border">
        <div className="flex items-center gap-2 px-2 py-1">
          <div className="w-7 h-7 rounded-full bg-muted flex items-center justify-center text-xs font-medium uppercase">
            {user?.username?.[0] ?? '?'}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">{user?.username}</p>
            <p className="text-xs text-muted-foreground capitalize">{user?.role}</p>
          </div>
          <button
            onClick={() => { logout(); navigate('/login') }}
            className="text-muted-foreground hover:text-red-400 transition-colors p-1 rounded"
            title="Sign out"
          >
            <LogOut size={16} />
          </button>
        </div>
      </div>
    </aside>
  )
}
