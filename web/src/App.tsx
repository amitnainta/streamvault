import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import AppShell from '@/components/layout/AppShell'
import LoginPage from '@/pages/Login'
import HomePage from '@/pages/Home'
import LibraryPage from '@/pages/Library'
import ItemPage from '@/pages/Item'
import PlayerPage from '@/pages/Player'
import AdminDashboard from '@/pages/admin/Dashboard'
import AdminPrivacy from '@/pages/admin/Privacy'
import AdminLibraries from '@/pages/admin/Libraries'
import AdminUsers from '@/pages/admin/Users'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.accessToken)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />

      <Route
        path="/*"
        element={
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        }
      >
        <Route index element={<HomePage />} />
        <Route path="library/:id" element={<LibraryPage />} />
        <Route path="item/:id" element={<ItemPage />} />
        <Route path="player/:id" element={<PlayerPage />} />
        <Route path="admin" element={<AdminDashboard />} />
        <Route path="admin/privacy" element={<AdminPrivacy />} />
        <Route path="admin/libraries" element={<AdminLibraries />} />
        <Route path="admin/users" element={<AdminUsers />} />
      </Route>
    </Routes>
  )
}
