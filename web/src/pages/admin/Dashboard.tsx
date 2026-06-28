import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { Cpu, HardDrive, Film, Users, Activity } from 'lucide-react'
import type { ServerStats } from '@/types/api'

export default function AdminDashboard() {
  const { data: stats } = useQuery({
    queryKey: ['server-stats'],
    queryFn: () => api.get<ServerStats>('/server/stats'),
    refetchInterval: 5000,
  })

  const cards = [
    { icon: <Cpu size={18} />, label: 'CPU', value: stats ? `${stats.cpu_pct.toFixed(1)}%` : '—' },
    { icon: <HardDrive size={18} />, label: 'Memory', value: stats ? `${stats.memory_mb} MB` : '—' },
    { icon: <Activity size={18} />, label: 'Active Streams', value: stats?.active_sessions ?? '—' },
    { icon: <Film size={18} />, label: 'Media Items', value: stats?.item_count?.toLocaleString() ?? '—' },
    { icon: <Users size={18} />, label: 'Libraries', value: stats?.library_count ?? '—' },
  ]

  return (
    <div className="p-8 space-y-6">
      <h1 className="text-xl font-semibold">Server Dashboard</h1>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
        {cards.map((c) => (
          <div key={c.label} className="bg-card border border-border rounded-xl p-4 space-y-2">
            <div className="flex items-center gap-2 text-muted-foreground">
              {c.icon}
              <span className="text-xs font-medium">{c.label}</span>
            </div>
            <p className="text-2xl font-semibold">{c.value}</p>
          </div>
        ))}
      </div>
    </div>
  )
}
