import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '@/types/api'

interface AuthState {
  accessToken: string | null
  user: User | null
  login: (username: string, password: string) => Promise<void>
  refresh: () => Promise<boolean>
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, _get) => ({
      accessToken: null,
      user: null,

      login: async (username, password) => {
        const res = await fetch('/api/v1/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password }),
          credentials: 'include', // receive HttpOnly refresh cookie
        })
        if (!res.ok) throw new Error('Invalid credentials')
        const data = await res.json()
        set({ accessToken: data.access_token, user: data.user })
      },

      refresh: async () => {
        try {
          const res = await fetch('/api/v1/auth/refresh', {
            method: 'POST',
            credentials: 'include',
          })
          if (!res.ok) return false
          const data = await res.json()
          set({ accessToken: data.access_token })
          return true
        } catch {
          return false
        }
      },

      logout: () => {
        fetch('/api/v1/auth/logout', { method: 'POST', credentials: 'include' })
        set({ accessToken: null, user: null })
      },
    }),
    {
      name: 'sv-auth',
      partialize: (s) => ({ accessToken: s.accessToken, user: s.user }),
    }
  )
)
