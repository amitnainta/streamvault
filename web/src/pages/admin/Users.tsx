import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { UserPlus, Shield, Eye } from 'lucide-react'
import type { User } from '@/types/api'

export default function AdminUsers() {
  const { data: users } = useQuery({
    queryKey: ['users'],
    queryFn: () => api.get<User[]>('/users'),
  })

  return (
    <div className="p-8 space-y-6 max-w-2xl">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Users</h1>
        <button className="flex items-center gap-2 bg-accent text-accent-foreground text-sm font-medium px-3 py-2 rounded-lg hover:bg-accent/90 transition-colors">
          <UserPlus size={15} /> Invite User
        </button>
      </div>

      <div className="space-y-2">
        {users?.map((u) => (
          <div key={u.id} className="bg-card border border-border rounded-xl p-4 flex items-center gap-4">
            <div className="w-9 h-9 rounded-full bg-muted flex items-center justify-center text-sm font-medium uppercase">
              {u.username[0]}
            </div>
            <div className="flex-1">
              <p className="font-medium">{u.username}</p>
              <p className="text-xs text-muted-foreground">
                Joined {new Date(u.created_at).toLocaleDateString()}
                {u.last_login_at && ` · Last seen ${new Date(u.last_login_at).toLocaleDateString()}`}
              </p>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              {u.role === 'admin' ? <Shield size={13} /> : <Eye size={13} />}
              <span className="capitalize">{u.role}</span>
            </div>
            <div className={`w-2 h-2 rounded-full ${u.is_enabled ? 'bg-green-500' : 'bg-muted'}`} title={u.is_enabled ? 'Active' : 'Disabled'} />
          </div>
        ))}
      </div>
    </div>
  )
}
