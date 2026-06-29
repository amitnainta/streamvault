import { useEffect, useRef, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import Hls from 'hls.js'
import { ArrowLeft, Play, Pause, Volume2, VolumeX, Maximize, Settings } from 'lucide-react'
import { api } from '@/api/client'
import { useAuthStore } from '@/store/authStore'
import type { MediaItem, PlaybackInfo } from '@/types/api'

type Quality = 'auto' | 'high' | 'medium' | 'low'

const QUALITY_LABELS: Record<Quality, string> = {
  auto:   'Auto (Direct)',
  high:   'High (8 Mbps)',
  medium: 'Medium (4 Mbps)',
  low:    'Low (2 Mbps)',
}

export default function PlayerPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  useAuthStore()
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)

  const [playing, setPlaying] = useState(false)
  const [muted, setMuted] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [quality, setQuality] = useState<Quality>('auto')
  const [showQuality, setShowQuality] = useState(false)
  const savedTimeRef = useRef(0)

  const { data: item } = useQuery({
    queryKey: ['item', id],
    queryFn: () => api.get<MediaItem>(`/items/${id}`),
    enabled: !!id,
  })

  const { data: playback, refetch: refetchPlayback } = useQuery({
    queryKey: ['playback', id, quality],
    queryFn: () => api.get<PlaybackInfo>(`/items/${id}/playback?quality=${quality}`),
    enabled: !!id,
  })

  const progressMutation = useMutation({
    mutationFn: (posMs: number) =>
      api.put(`/progress/${id}`, {
        position_ms: posMs,
        duration_ms: duration * 1000,
        completed: duration > 0 && posMs / (duration * 1000) > 0.9,
      }),
  })

  const stopHls = useCallback(() => {
    hlsRef.current?.destroy()
    hlsRef.current = null
  }, [])

  // Set up HLS or direct play
  useEffect(() => {
    if (!playback || !videoRef.current) return
    const video = videoRef.current
    const resumeAt = savedTimeRef.current

    stopHls()

    const tryPlay = () => {
      if (resumeAt > 0) video.currentTime = resumeAt
      video.play().catch(() => {})
    }

    if (playback.type === 'direct') {
      video.src = playback.url
      video.load()
      video.addEventListener('canplay', tryPlay, { once: true })
    } else if (Hls.isSupported()) {
      const token = useAuthStore.getState().accessToken
      const hls = new Hls({ xhrSetup: (xhr) => xhr.setRequestHeader('Authorization', `Bearer ${token}`) })
      hls.loadSource(playback.url)
      hls.attachMedia(video)
      hls.on(Hls.Events.MANIFEST_PARSED, tryPlay)
      hlsRef.current = hls
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = playback.url
      video.addEventListener('canplay', tryPlay, { once: true })
    }

    return () => stopHls()
  }, [playback, stopHls])

  // Save progress every 10 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      if (videoRef.current && playing) {
        progressMutation.mutate(Math.floor(videoRef.current.currentTime * 1000))
      }
    }, 10_000)
    return () => clearInterval(interval)
  }, [playing])

  const changeQuality = (q: Quality) => {
    // Save current position so new stream resumes from same point
    savedTimeRef.current = videoRef.current?.currentTime ?? 0
    setShowQuality(false)
    setQuality(q)
    // Stop current HLS session before refetch starts a new one
    if (playback?.type === 'hls' && playback.session_id) {
      api.delete(`/stream/sessions/${playback.session_id}`).catch(() => {})
    }
    refetchPlayback()
  }

  const togglePlay = () => {
    if (!videoRef.current) return
    if (playing) videoRef.current.pause()
    else videoRef.current.play()
  }

  const seek = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!videoRef.current) return
    videoRef.current.currentTime = Number(e.target.value)
  }

  const fmt = (s: number) => {
    const h = Math.floor(s / 3600)
    const m = Math.floor((s % 3600) / 60)
    const sec = Math.floor(s % 60)
    return h > 0
      ? `${h}:${m.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`
      : `${m}:${sec.toString().padStart(2, '0')}`
  }

  return (
    <div className="fixed inset-0 bg-black flex flex-col" onClick={() => setShowQuality(false)}>
      {/* Video */}
      <video
        ref={videoRef}
        className="flex-1 w-full"
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onTimeUpdate={() => setCurrentTime(videoRef.current?.currentTime ?? 0)}
        onLoadedMetadata={() => setDuration(videoRef.current?.duration ?? 0)}
        onClick={(e) => { e.stopPropagation(); togglePlay() }}
      />

      {/* Controls overlay */}
      <div className={`absolute inset-0 flex flex-col justify-between p-4 bg-gradient-to-b from-black/60 via-transparent to-black/80 transition-opacity ${playing && !showQuality ? 'opacity-0 hover:opacity-100' : 'opacity-100'}`}>
        {/* Top bar */}
        <div className="flex items-center gap-3">
          <button onClick={() => navigate(-1)} className="text-white hover:text-white/80">
            <ArrowLeft size={20} />
          </button>
          <span className="text-white font-medium">{item?.metadata?.title}</span>
        </div>

        {/* Bottom bar */}
        <div className="space-y-2">
          {/* Seek bar */}
          <input
            type="range"
            min={0}
            max={duration}
            value={currentTime}
            onChange={seek}
            className="w-full h-1 accent-accent cursor-pointer"
          />
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <button onClick={togglePlay} className="text-white hover:text-white/80">
                {playing ? <Pause size={20} fill="white" /> : <Play size={20} fill="white" />}
              </button>
              <button
                onClick={() => { setMuted(!muted); if (videoRef.current) videoRef.current.muted = !muted }}
                className="text-white hover:text-white/80"
              >
                {muted ? <VolumeX size={18} /> : <Volume2 size={18} />}
              </button>
              <span className="text-white text-sm">{fmt(currentTime)} / {fmt(duration)}</span>
            </div>

            <div className="flex items-center gap-3">
              {/* Quality selector */}
              <div className="relative" onClick={(e) => e.stopPropagation()}>
                <button
                  onClick={() => setShowQuality(!showQuality)}
                  className="flex items-center gap-1.5 text-white hover:text-white/80 text-xs font-medium"
                  title="Quality"
                >
                  <Settings size={16} />
                  <span className="hidden sm:inline">{QUALITY_LABELS[quality]}</span>
                </button>
                {showQuality && (
                  <div className="absolute bottom-8 right-0 bg-black/90 border border-white/20 rounded-lg overflow-hidden min-w-[160px]">
                    {(Object.keys(QUALITY_LABELS) as Quality[]).map((q) => (
                      <button
                        key={q}
                        onClick={() => changeQuality(q)}
                        className={`w-full text-left px-4 py-2 text-sm transition-colors
                          ${quality === q ? 'text-accent bg-white/10' : 'text-white hover:bg-white/10'}`}
                      >
                        {QUALITY_LABELS[q]}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              <button onClick={() => videoRef.current?.requestFullscreen()} className="text-white hover:text-white/80">
                <Maximize size={18} />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
