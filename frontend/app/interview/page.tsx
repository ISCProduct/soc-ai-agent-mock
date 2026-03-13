'use client'

import { useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  Box,
  Button,
  Chip,
  Divider,
  Drawer,
  IconButton,
  InputBase,
  LinearProgress,
  Paper,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import MicIcon from '@mui/icons-material/Mic'
import MicOffIcon from '@mui/icons-material/MicOff'
import VideocamIcon from '@mui/icons-material/Videocam'
import VideocamOffIcon from '@mui/icons-material/VideocamOff'
import CallEndIcon from '@mui/icons-material/CallEnd'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import RefreshIcon from '@mui/icons-material/Refresh'
import ClosedCaptionIcon from '@mui/icons-material/ClosedCaption'
import PsychologyIcon from '@mui/icons-material/Psychology'
import LightbulbIcon from '@mui/icons-material/Lightbulb'
import SendIcon from '@mui/icons-material/Send'
import SearchIcon from '@mui/icons-material/Search'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ApartmentIcon from '@mui/icons-material/Apartment'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import { authService, User } from '@/lib/auth'
import { interviewApi, interviewLimits, InterviewReport, InterviewSession } from '@/lib/interview'
import ThreeAvatar from './components/ThreeAvatar'

const PRIMARY = '#ec5b13'
const BG_LIGHT = '#f8f6f6'
const BG_DARK = '#221610'

type Utterance = {
  role: 'user' | 'ai'
  text: string
}

type InterviewCompany = {
  id: number
  name: string
  description?: string
  main_business?: string
  industry?: string
  location?: string
  employee_count?: number
  culture?: string
  work_style?: string
  welfare_details?: string
}

type Position = {
  id: string
  title: string
  department: string
  icon: string
  questions: number
}

const POSITIONS: Position[] = [
  { id: 'engineer', title: 'ソフトウェアエンジニア', department: 'Engineering', icon: '💻', questions: 8 },
  { id: 'designer', title: 'プロダクトデザイナー', department: 'Design', icon: '🎨', questions: 7 },
  { id: 'sales', title: '営業職', department: 'Sales', icon: '📈', questions: 7 },
  { id: 'marketing', title: 'マーケティング', department: 'Growth', icon: '📣', questions: 6 },
  { id: 'pm', title: 'プロダクトマネージャー', department: 'Product', icon: '🧭', questions: 9 },
  { id: 'data', title: 'データアナリスト', department: 'Data', icon: '📊', questions: 7 },
]

export default function InterviewPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [status, setStatus] = useState<'selection' | 'lobby' | 'connecting' | 'connected' | 'error' | 'finished'>('selection')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [utterances, setUtterances] = useState<Utterance[]>([])
  const [partialUser, setPartialUser] = useState('')
  const [partialAi, setPartialAi] = useState('')
  const [remainingSeconds, setRemainingSeconds] = useState(interviewLimits.maxMinutes * 60)
  const [estimatedCost, setEstimatedCost] = useState(0)
  const [session, setSession] = useState<InterviewSession | null>(null)
  const [report, setReport] = useState<InterviewReport | null>(null)
  const [reportStatus, setReportStatus] = useState<'idle' | 'pending' | 'ready' | 'error'>('idle')
  const [aiLevel, setAiLevel] = useState(0)
  const [aiSpeaking, setAiSpeaking] = useState(false)
  const [avatarGender, setAvatarGender] = useState<'male' | 'female'>('male')
  const [interviewCompany, setInterviewCompany] = useState<InterviewCompany | null>(null)
  const [micEnabled, setMicEnabled] = useState(true)
  const [cameraEnabled, setCameraEnabled] = useState(true)
  const [noteInput, setNoteInput] = useState('')
  const [lobbyPermissionError, setLobbyPermissionError] = useState<string | null>(null)
  const [captionsVisible, setCaptionsVisible] = useState(true)
  // Selection screen state
  const [allCompanies, setAllCompanies] = useState<InterviewCompany[]>([])
  const [companiesLoading, setCompaniesLoading] = useState(false)
  const [companySearch, setCompanySearch] = useState('')
  const [selectedPosition, setSelectedPosition] = useState<Position>(POSITIONS[0])

  const pcRef = useRef<RTCPeerConnection | null>(null)
  const dcRef = useRef<RTCDataChannel | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const lobbyVideoRef = useRef<HTMLVideoElement | null>(null)
  const sessionVideoRef = useRef<HTMLVideoElement | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const aiAudioStreamRef = useRef<MediaStream | null>(null)
  const aiAudioCtxRef = useRef<AudioContext | null>(null)
  const aiAnalyserRef = useRef<AnalyserNode | null>(null)
  const aiAnimationRef = useRef<number | null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const sessionStartRef = useRef<number | null>(null)
  const transcriptEndRef = useRef<HTMLDivElement | null>(null)

  // Auth check
  useEffect(() => {
    const storedUser = authService.getStoredUser()
    if (!storedUser) { router.replace('/login'); return }
    if (storedUser.target_level !== '新卒' && storedUser.target_level !== '中途') {
      router.replace('/onboarding'); return
    }
    setUser(storedUser)
    setLoading(false)
  }, [router])

  // Load company list for selection screen (initial fetch + debounced search)
  useEffect(() => {
    if (loading) return
    let cancelled = false
    const timer = setTimeout(() => {
      setCompaniesLoading(true)
      const params = new URLSearchParams({ limit: '50', offset: '0' })
      if (companySearch.trim()) params.set('name', companySearch.trim())
      fetch(`/api/companies?${params}`, { cache: 'no-store' })
        .then(r => r.ok ? r.json() : null)
        .then(data => {
          if (cancelled) return
          const list: InterviewCompany[] = Array.isArray(data?.companies) ? data.companies : []
          setAllCompanies(list)
          if (list.length > 0 && !interviewCompany) setInterviewCompany(list[0])
        })
        .catch(() => { /* ignore */ })
        .finally(() => { if (!cancelled) setCompaniesLoading(false) })
    }, companySearch ? 400 : 0)
    return () => { cancelled = true; clearTimeout(timer) }
  }, [loading, companySearch]) // eslint-disable-line react-hooks/exhaustive-deps

  // Lobby camera preview
  useEffect(() => {
    if (loading) return
    let stream: MediaStream | null = null
    const startPreview = async () => {
      try {
        stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
        streamRef.current = stream
        if (lobbyVideoRef.current) {
          lobbyVideoRef.current.srcObject = stream
          lobbyVideoRef.current.play().catch(() => undefined)
        }
      } catch (err: any) {
        if (err.name === 'NotAllowedError' || err.name === 'PermissionDeniedError') {
          setLobbyPermissionError('マイクとカメラへのアクセスが拒否されました。ブラウザの設定から許可してください。')
        } else if (err.name === 'NotFoundError') {
          setLobbyPermissionError('マイクまたはカメラが見つかりません。デバイスを確認してください。')
        } else {
          setLobbyPermissionError('カメラの起動に失敗しました。')
        }
      }
    }
    startPreview()
    return () => {
      // stream is kept in streamRef for reuse during interview
    }
  }, [loading])

  // Cleanup on unmount
  useEffect(() => () => cleanupConnection(), [])

  // Auto-scroll transcript
  useEffect(() => {
    transcriptEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [utterances, partialAi])

  // Attach stream to session video when connected
  useEffect(() => {
    if (status === 'connected' && sessionVideoRef.current && streamRef.current) {
      sessionVideoRef.current.srcObject = streamRef.current
      sessionVideoRef.current.play().catch(() => undefined)
    }
  }, [status])

  const formatSeconds = (s: number) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
  const parseJsonSafe = (v?: string) => { try { return v ? JSON.parse(v) : null } catch { return null } }

  const cleanupConnection = () => {
    ;[timerRef, pollRef].forEach(r => { if (r.current) { clearInterval(r.current); r.current = null } })
    if (aiAnimationRef.current) { cancelAnimationFrame(aiAnimationRef.current); aiAnimationRef.current = null }
    if (aiAudioCtxRef.current) { aiAudioCtxRef.current.close().catch(() => undefined); aiAudioCtxRef.current = null; aiAnalyserRef.current = null }
    if (dcRef.current) { dcRef.current.close(); dcRef.current = null }
    if (pcRef.current) { pcRef.current.close(); pcRef.current = null }
    if (streamRef.current) { streamRef.current.getTracks().forEach(t => t.stop()); streamRef.current = null }
  }

  const startTimer = () => {
    sessionStartRef.current = Date.now()
    timerRef.current = setInterval(() => {
      if (!sessionStartRef.current) return
      const elapsed = Math.floor((Date.now() - sessionStartRef.current) / 1000)
      const remaining = Math.max(0, interviewLimits.maxMinutes * 60 - elapsed)
      setRemainingSeconds(remaining)
      setEstimatedCost((elapsed / 60) * interviewLimits.costPerMinuteUSD)
      if (remaining <= 0) handleStop(true)
    }, 1000)
  }

  const parseStartError = (error: any): string => {
    const msg: string = error?.message || ''
    if (msg.includes('NotAllowedError') || msg.toLowerCase().includes('denied'))
      return 'マイクとカメラへのアクセスが拒否されました。ブラウザのアドレスバー横から権限を許可してください。'
    if (msg.includes('NotFoundError'))
      return 'マイクまたはカメラが見つかりません。デバイスが正しく接続されているか確認してください。'
    if (msg.toLowerCase().includes('unauthorized') || msg.includes('401'))
      return 'AIサービスへの接続に失敗しました。（OpenAI APIキーを確認してください）'
    return msg || '接続に失敗しました。ネットワークを確認して再試行してください。'
  }

  const handleJoin = async () => {
    if (!user) return
    setErrorMessage(null)
    setUtterances([])
    setPartialUser(''); setPartialAi('')
    setReport(null); setReportStatus('idle')
    setRemainingSeconds(interviewLimits.maxMinutes * 60)
    setEstimatedCost(0)
    setMicEnabled(true); setCameraEnabled(true)

    try {
      setStatus('connecting')
      const nextGender = getNextAvatarGender()
      setAvatarGender(nextGender)
      const created = await interviewApi.createSession(user.user_id)
      setSession(created)
      await interviewApi.startSession(created.id, user.user_id)
      const token = await interviewApi.createRealtimeToken(user.user_id, created.id)
      await startConnection(token, created.id)
      setStatus('connected')
      startTimer()
    } catch (error: any) {
      setStatus('error')
      setErrorMessage(parseStartError(error))
      cleanupConnection()
    }
  }

  const handleStop = async (forced = false) => {
    cleanupConnection()
    if (user && session) {
      try { await interviewApi.finishSession(session.id, user.user_id) } catch { /* ignore */ }
    }
    setStatus('finished')
    setReportStatus('pending')
    if (forced) setErrorMessage('時間上限に達したため面接を終了しました。')
    if (session && user) startReportPolling(session.id, user.user_id)
  }

  const startReportPolling = (sessionId: number, userId: number) => {
    if (pollRef.current) clearInterval(pollRef.current)
    pollRef.current = setInterval(async () => {
      try {
        const detail = await interviewApi.getDetail(sessionId, userId)
        if (detail.report) {
          setReport(detail.report); setReportStatus('ready')
          clearInterval(pollRef.current!); pollRef.current = null
        }
      } catch { setReportStatus('error') }
    }, 3000)
  }

  const startConnection = async (token: string, sessionId: number) => {
    const pc = new RTCPeerConnection()
    pcRef.current = pc
    const dc = pc.createDataChannel('oai-events')
    dcRef.current = dc
    dc.onmessage = (e) => {
      try { handleRealtimeEvent(JSON.parse(e.data), sessionId) } catch { /* ignore */ }
    }
    pc.ontrack = (event) => {
      if (audioRef.current) {
        audioRef.current.srcObject = event.streams[0]
        audioRef.current.play().catch(() => undefined)
        setupAiAudioAnalyser(event.streams[0])
      }
    }

    // Reuse lobby stream if available, otherwise acquire new
    let stream = streamRef.current
    if (!stream) {
      try {
        stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
      } catch (err: any) {
        if (err.name === 'NotAllowedError' || err.name === 'PermissionDeniedError')
          throw new Error('NotAllowedError')
        if (err.name === 'NotFoundError')
          throw new Error('NotFoundError')
        throw err
      }
      streamRef.current = stream
    }

    stream.getTracks().forEach(track => pc.addTrack(track, stream!))
    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)

    const response = await fetch('https://api.openai.com/v1/realtime/calls', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/sdp' },
      body: offer.sdp,
    })
    if (!response.ok) throw new Error(await response.text())
    await pc.setRemoteDescription({ type: 'answer', sdp: await response.text() })

    dc.onopen = () => {
      dc.send(JSON.stringify({
        type: 'conversation.item.create',
        item: { type: 'message', role: 'user', content: [{ type: 'input_text', text: '面接を開始してください。最初の質問をお願いします。' }] },
      }))
      dc.send(JSON.stringify({ type: 'response.create', response: { modalities: ['audio'] } }))
    }
  }

  const setupAiAudioAnalyser = (stream: MediaStream) => {
    if (aiAudioCtxRef.current) return
    aiAudioStreamRef.current = stream
    const ctx = new AudioContext()
    aiAudioCtxRef.current = ctx
    const source = ctx.createMediaStreamSource(stream)
    const analyser = ctx.createAnalyser()
    analyser.fftSize = 512
    aiAnalyserRef.current = analyser
    source.connect(analyser)
    const data = new Uint8Array(analyser.frequencyBinCount)
    const tick = () => {
      analyser.getByteTimeDomainData(data)
      let sum = 0
      data.forEach(v => { const n = (v - 128) / 128; sum += n * n })
      const level = Math.min(1, Math.sqrt(sum / data.length) * 2.5)
      setAiLevel(level)
      setAiSpeaking(level > 0.08)
      aiAnimationRef.current = requestAnimationFrame(tick)
    }
    tick()
  }

  const handleRealtimeEvent = async (event: any, sessionId: number) => {
    if (!user) return
    switch (event.type) {
      case 'conversation.item.input_audio_transcription.delta':
        if (event.delta) setPartialUser(p => p + event.delta); break
      case 'conversation.item.input_audio_transcription.completed': {
        const text = (event.transcript || event.text || partialUser).trim()
        if (text) {
          setUtterances(p => [...p, { role: 'user', text }]); setPartialUser('')
          try { await interviewApi.saveUtterance(sessionId, user.user_id, 'user', text) } catch { /* ignore */ }
        }
        break
      }
      case 'response.audio_transcript.delta':
        if (event.delta) setPartialAi(p => p + event.delta); break
      case 'response.audio_transcript.done': {
        const text = (event.transcript || partialAi).trim()
        if (text) {
          setUtterances(p => [...p, { role: 'ai', text }]); setPartialAi('')
          try { await interviewApi.saveUtterance(sessionId, user.user_id, 'ai', text) } catch { /* ignore */ }
        }
        break
      }
    }
  }

  const toggleMic = () => {
    if (!streamRef.current) return
    const next = !micEnabled
    streamRef.current.getAudioTracks().forEach(t => { t.enabled = next })
    setMicEnabled(next)
  }

  const toggleCamera = () => {
    if (!streamRef.current) return
    const next = !cameraEnabled
    streamRef.current.getVideoTracks().forEach(t => { t.enabled = next })
    setCameraEnabled(next)
  }

  if (loading || !user) return null

  const isActive = status === 'connecting' || status === 'connected'
  const isConnected = status === 'connected'
  const progress = Math.min(100, Math.round(((interviewLimits.maxMinutes * 60 - remainingSeconds) / (interviewLimits.maxMinutes * 60)) * 100))
  const scores = report ? parseJsonSafe(report.scores_json) : null
  const evidence = report ? parseJsonSafe(report.evidence_json) : null
  const isFemale = avatarGender === 'female'
  const companyName = interviewCompany?.name || 'AI面接練習'
  const latestAiText = partialAi || (utterances.filter(u => u.role === 'ai').slice(-1)[0]?.text ?? '')
  const recruitingText = [
    '【募集背景】', interviewCompany?.description || '企業情報の取得後に表示されます。', '',
    '【仕事内容】', interviewCompany?.main_business || interviewCompany?.industry || '詳細は面接内でご案内します。', '',
    '【職場環境】', interviewCompany?.work_style || '勤務形態は選考でご説明します。', '',
    '【企業文化・福利厚生】',
    `${interviewCompany?.culture || 'チームで成果を重視する文化'} / ${interviewCompany?.welfare_details || '情報準備中'}`,
    '', `【勤務地・人数】 ${interviewCompany?.location || '未設定'} / ${interviewCompany?.employee_count ? interviewCompany.employee_count + '名' : '非公開'}`,
  ].join('\n')

  // allCompanies is already filtered by server-side search, no client-side filter needed
  const filteredCompanies = allCompanies

  // ─────────────────────────────────────────────
  // SELECTION SCREEN  (Step 1 of 3)
  // ─────────────────────────────────────────────
  if (status === 'selection') {
    return (
      <Box sx={{ minHeight: '100vh', bgcolor: BG_LIGHT }}>
        {/* Header */}
        <Box component="header" sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: { xs: 3, lg: 10 }, py: 2, bgcolor: '#fff', borderBottom: '1px solid #e2e8f0' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box sx={{ color: PRIMARY, display: 'flex', alignItems: 'center' }}>
              <PsychologyIcon sx={{ fontSize: 32 }} />
            </Box>
            <Typography sx={{ fontWeight: 700, fontSize: 20, color: '#0f172a' }}>InterviewAI</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <IconButton sx={{ bgcolor: '#f1f5f9', color: '#475569' }} size="small" onClick={() => router.push('/')}>
              <ArrowBackIcon fontSize="small" />
            </IconButton>
            <Box sx={{ width: 40, height: 40, borderRadius: '50%', bgcolor: `${PRIMARY}30`, border: `1px solid ${PRIMARY}50`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <Typography sx={{ fontWeight: 700, color: PRIMARY, fontSize: 14 }}>
                {(user.name || 'U').charAt(0).toUpperCase()}
              </Typography>
            </Box>
          </Box>
        </Box>

        {/* Main */}
        <Box component="main" sx={{ display: 'flex', justifyContent: 'center', py: 5, px: { xs: 3, lg: 10 } }}>
          <Box sx={{ maxWidth: 896, width: '100%', display: 'flex', flexDirection: 'column', gap: 4 }}>

            {/* Step indicator + Title */}
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <Typography sx={{ color: PRIMARY, fontWeight: 600, fontSize: 13, textTransform: 'uppercase', letterSpacing: 1 }}>
                  Step 1 / 3
                </Typography>
                <Box sx={{ height: 4, width: 96, bgcolor: '#e2e8f0', borderRadius: 9999, overflow: 'hidden' }}>
                  <Box sx={{ height: '100%', width: '33%', bgcolor: PRIMARY }} />
                </Box>
              </Box>
              <Typography variant="h4" sx={{ fontWeight: 700, color: '#0f172a' }}>練習する企業・職種を選ぶ</Typography>
              <Typography sx={{ color: '#64748b', fontSize: 15 }}>
                志望企業と職種を選択して、AIが面接内容をカスタマイズします。
              </Typography>
            </Box>

            {/* 3-col grid */}
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '2fr 1fr' }, gap: 4, alignItems: 'start' }}>

              {/* Left: Company + Position */}
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>

                {/* Company section */}
                <Paper elevation={0} sx={{ p: 3, borderRadius: 2, border: '1px solid #e2e8f0' }}>
                  <Typography sx={{ fontWeight: 700, fontSize: 17, mb: 2 }}>志望企業</Typography>
                  {/* Search */}
                  <Box sx={{ position: 'relative', mb: 3 }}>
                    <SearchIcon sx={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: '#94a3b8', fontSize: 20 }} />
                    <Box
                      component="input"
                      value={companySearch}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => setCompanySearch(e.target.value)}
                      placeholder="企業名・業種で検索（例：テック、製造、金融）"
                      sx={{
                        width: '100%', pl: '40px', pr: 2, py: 1.5,
                        bgcolor: '#f8fafc', border: '1px solid #e2e8f0',
                        borderRadius: 2, fontSize: 14, color: '#0f172a',
                        outline: 'none', boxSizing: 'border-box',
                        '&:focus': { borderColor: PRIMARY, boxShadow: `0 0 0 2px ${PRIMARY}20` },
                        fontFamily: 'inherit',
                      }}
                    />
                  </Box>

                  {/* Company chips */}
                  {companiesLoading ? (
                    <LinearProgress sx={{ borderRadius: 1 }} />
                  ) : (
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5 }}>
                      {filteredCompanies.map(c => {
                        const isSelected = interviewCompany?.id === c.id
                        return (
                          <Button
                            key={c.id}
                            size="small"
                            onClick={() => setInterviewCompany(c)}
                            startIcon={isSelected ? <CheckCircleIcon sx={{ fontSize: '16px !important' }} /> : undefined}
                            sx={{
                              px: 2, py: 0.8, borderRadius: 2, fontWeight: 500, fontSize: 13,
                              textTransform: 'none',
                              bgcolor: isSelected ? PRIMARY : '#f1f5f9',
                              color: isSelected ? '#fff' : '#475569',
                              '&:hover': { bgcolor: isSelected ? `${PRIMARY}e0` : '#e2e8f0' },
                            }}
                          >
                            {c.name}
                          </Button>
                        )
                      })}
                      {filteredCompanies.length === 0 && !companiesLoading && (
                        <Typography sx={{ color: '#94a3b8', fontSize: 13 }}>企業が見つかりません</Typography>
                      )}
                    </Box>
                  )}
                </Paper>

                {/* Position section */}
                <Paper elevation={0} sx={{ p: 3, borderRadius: 2, border: '1px solid #e2e8f0' }}>
                  <Typography sx={{ fontWeight: 700, fontSize: 17, mb: 2 }}>応募職種</Typography>
                  <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 1.5 }}>
                    {POSITIONS.map(pos => {
                      const isSelected = selectedPosition.id === pos.id
                      return (
                        <Box
                          key={pos.id}
                          onClick={() => setSelectedPosition(pos)}
                          sx={{
                            position: 'relative', display: 'flex', alignItems: 'center', gap: 1.5,
                            p: 2, borderRadius: 2, cursor: 'pointer',
                            border: `2px solid ${isSelected ? PRIMARY : 'transparent'}`,
                            bgcolor: isSelected ? `${PRIMARY}08` : '#f8fafc',
                            transition: 'all 0.15s',
                            '&:hover': { borderColor: isSelected ? PRIMARY : '#cbd5e1' },
                          }}
                        >
                          <Typography sx={{ fontSize: 22 }}>{pos.icon}</Typography>
                          <Box sx={{ flex: 1 }}>
                            <Typography sx={{ fontWeight: 700, fontSize: 14, color: '#0f172a' }}>{pos.title}</Typography>
                            <Typography sx={{ fontSize: 12, color: '#94a3b8' }}>{pos.department}</Typography>
                          </Box>
                          <Box sx={{ position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)' }}>
                            {isSelected
                              ? <CheckCircleIcon sx={{ color: PRIMARY, fontSize: 20 }} />
                              : <Box sx={{ width: 18, height: 18, borderRadius: '50%', border: '2px solid #cbd5e1' }} />
                            }
                          </Box>
                        </Box>
                      )
                    })}
                  </Box>
                </Paper>
              </Box>

              {/* Right: Summary + CTA */}
              <Box sx={{ position: { md: 'sticky' }, top: 32, display: 'flex', flexDirection: 'column', gap: 3 }}>
                <Paper elevation={0} sx={{ p: 3, borderRadius: 2, border: `1px solid ${PRIMARY}30`, bgcolor: `${PRIMARY}05` }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 3 }}>
                    <Box sx={{ width: 48, height: 48, borderRadius: 2, bgcolor: '#fff', border: '1px solid #e2e8f0', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      <ApartmentIcon sx={{ color: PRIMARY }} />
                    </Box>
                    <Box>
                      <Typography sx={{ fontWeight: 700, fontSize: 15 }}>{interviewCompany?.name || '企業未選択'}</Typography>
                      <Typography sx={{ fontSize: 13, color: '#64748b' }}>{interviewCompany?.industry || '業種未設定'}</Typography>
                    </Box>
                  </Box>

                  <Stack spacing={2.5}>
                    <Box>
                      <Typography sx={{ fontSize: 11, fontWeight: 700, color: '#94a3b8', textTransform: 'uppercase', letterSpacing: 1, mb: 0.5 }}>応募ポジション</Typography>
                      <Typography sx={{ fontWeight: 600, fontSize: 15 }}>{selectedPosition.title}</Typography>
                    </Box>
                    <Box>
                      <Typography sx={{ fontSize: 11, fontWeight: 700, color: '#94a3b8', textTransform: 'uppercase', letterSpacing: 1, mb: 0.5 }}>企業概要</Typography>
                      <Typography sx={{ fontSize: 13, color: '#64748b', lineHeight: 1.7 }}>
                        {interviewCompany?.description
                          ? interviewCompany.description.slice(0, 120) + (interviewCompany.description.length > 120 ? '...' : '')
                          : '企業を選択すると詳細が表示されます。'}
                      </Typography>
                    </Box>
                    <Box sx={{ pt: 2, borderTop: `1px solid ${PRIMARY}15` }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                        <Typography sx={{ fontSize: 14 }}>⏱</Typography>
                        <Typography sx={{ fontSize: 13, color: '#475569' }}>所要時間: {interviewLimits.maxMinutes}分</Typography>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography sx={{ fontSize: 14 }}>❓</Typography>
                        <Typography sx={{ fontSize: 13, color: '#475569' }}>{selectedPosition.questions}問 技術・行動面接</Typography>
                      </Box>
                    </Box>
                  </Stack>
                </Paper>

                <Button
                  variant="contained"
                  fullWidth
                  endIcon={<ArrowForwardIcon />}
                  disabled={!interviewCompany}
                  onClick={() => setStatus('lobby')}
                  sx={{
                    bgcolor: PRIMARY, '&:hover': { bgcolor: `${PRIMARY}e0` },
                    borderRadius: 2, py: 1.8, fontWeight: 700, fontSize: 16,
                    textTransform: 'none',
                    boxShadow: `0 8px 24px ${PRIMARY}30`,
                    '&:disabled': { bgcolor: '#e2e8f0', color: '#94a3b8', boxShadow: 'none' },
                  }}
                >
                  面接を開始する
                </Button>
                <Typography sx={{ fontSize: 12, textAlign: 'center', color: '#94a3b8' }}>
                  開始すると<Box component="a" href="#" sx={{ textDecoration: 'underline', color: 'inherit' }}>利用規約</Box>に同意したことになります
                </Typography>
              </Box>
            </Box>
          </Box>
        </Box>
      </Box>
    )
  }

  // ─────────────────────────────────────────────
  // LOBBY SCREEN
  // ─────────────────────────────────────────────
  if (status === 'lobby') {
    return (
      <Box sx={{ minHeight: '100vh', bgcolor: BG_LIGHT, display: 'flex', flexDirection: 'column' }}>
        {/* Header */}
        <Box component="header" sx={{ px: 3, py: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box sx={{ width: 40, height: 40, borderRadius: 2, bgcolor: PRIMARY, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <PsychologyIcon sx={{ color: '#fff', fontSize: 24 }} />
            </Box>
            <Box>
              <Typography sx={{ fontWeight: 700, fontSize: 16, lineHeight: 1.2 }}>AI面接練習</Typography>
              <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>セッション: {companyName}</Typography>
            </Box>
          </Box>
          <IconButton size="small" onClick={() => router.push('/')}>
            <ArrowBackIcon fontSize="small" />
          </IconButton>
        </Box>

        {/* Content */}
        <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', p: { xs: 2, md: 6 } }}>
          <Box sx={{ maxWidth: 960, width: '100%', display: 'grid', gridTemplateColumns: { xs: '1fr', lg: '7fr 5fr' }, gap: { xs: 4, lg: 8 }, alignItems: 'center' }}>

            {/* Camera preview */}
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
              <Box sx={{ position: 'relative', width: '100%', aspectRatio: '16/9', bgcolor: '#202124', borderRadius: 2, overflow: 'hidden', boxShadow: '0 4px 20px rgba(0,0,0,0.25)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                {lobbyPermissionError ? (
                  <Box sx={{ textAlign: 'center', p: 3 }}>
                    <Typography sx={{ color: '#f28b82', mb: 2, fontSize: 14 }}>{lobbyPermissionError}</Typography>
                    <Button size="small" startIcon={<RefreshIcon />} onClick={() => { setLobbyPermissionError(null); window.location.reload() }} sx={{ color: '#8ab4f8' }}>
                      再試行
                    </Button>
                  </Box>
                ) : (
                  <video
                    ref={lobbyVideoRef}
                    muted
                    playsInline
                    style={{ width: '100%', height: '100%', objectFit: 'cover', transform: 'scaleX(-1)', display: cameraEnabled ? 'block' : 'none' }}
                  />
                )}
                {!lobbyPermissionError && !cameraEnabled && (
                  <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <VideocamOffIcon sx={{ color: '#9aa0a6', fontSize: 48 }} />
                  </Box>
                )}

                {/* Camera label */}
                <Box sx={{ position: 'absolute', top: 12, left: 12, bgcolor: 'rgba(0,0,0,0.5)', px: 1.5, py: 0.5, borderRadius: 1 }}>
                  <Typography sx={{ color: '#fff', fontSize: 13 }}>{user.name || 'あなた'}</Typography>
                </Box>

                {/* Controls overlay */}
                <Box sx={{ position: 'absolute', bottom: 16, left: 0, right: 0, display: 'flex', justifyContent: 'center', gap: 2 }}>
                  <Tooltip title={micEnabled ? 'マイクをオフ' : 'マイクをオン'}>
                    <IconButton
                      onClick={() => {
                        if (!streamRef.current) return
                        const next = !micEnabled
                        streamRef.current.getAudioTracks().forEach(t => { t.enabled = next })
                        setMicEnabled(next)
                      }}
                      sx={{ bgcolor: micEnabled ? 'rgba(255,255,255,0.15)' : '#ea4335', border: '1px solid rgba(255,255,255,0.3)', '&:hover': { bgcolor: micEnabled ? 'rgba(255,255,255,0.25)' : '#c5221f' } }}
                    >
                      {micEnabled ? <MicIcon sx={{ color: '#fff' }} /> : <MicOffIcon sx={{ color: '#fff' }} />}
                    </IconButton>
                  </Tooltip>
                  <Tooltip title={cameraEnabled ? 'カメラをオフ' : 'カメラをオン'}>
                    <IconButton
                      onClick={() => {
                        if (!streamRef.current) return
                        const next = !cameraEnabled
                        streamRef.current.getVideoTracks().forEach(t => { t.enabled = next })
                        setCameraEnabled(next)
                      }}
                      sx={{ bgcolor: cameraEnabled ? 'rgba(255,255,255,0.15)' : '#ea4335', border: '1px solid rgba(255,255,255,0.3)', '&:hover': { bgcolor: cameraEnabled ? 'rgba(255,255,255,0.25)' : '#c5221f' } }}
                    >
                      {cameraEnabled ? <VideocamIcon sx={{ color: '#fff' }} /> : <VideocamOffIcon sx={{ color: '#fff' }} />}
                    </IconButton>
                  </Tooltip>
                </Box>
              </Box>

              <Typography variant="body2" sx={{ color: 'text.secondary', fontSize: 13 }}>
                カメラとマイクの確認が完了したら「面接に参加」を押してください
              </Typography>
            </Box>

            {/* Join panel */}
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: { xs: 'center', lg: 'flex-start' }, textAlign: { xs: 'center', lg: 'left' } }}>
              <Typography variant="h4" sx={{ fontWeight: 400, color: '#202124', mb: 1 }}>
                準備はできましたか？
              </Typography>
              <Typography sx={{ color: 'text.secondary', mb: 4, fontSize: 15 }}>
                {companyName}
              </Typography>

              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ width: '100%', maxWidth: 340 }}>
                <Button
                  variant="contained"
                  onClick={handleJoin}
                  sx={{
                    flex: 1,
                    bgcolor: '#1a73e8',
                    '&:hover': { bgcolor: '#1557b0' },
                    borderRadius: 9999,
                    py: 1.2,
                    fontWeight: 500,
                    fontSize: 15,
                    textTransform: 'none',
                    boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
                  }}
                >
                  面接に参加
                </Button>
              </Stack>

              <Box sx={{ mt: 5 }}>
                <Typography variant="body2" sx={{ color: 'text.secondary', mb: 1.5 }}>募集要項を確認する</Typography>
                <Button
                  size="small"
                  onClick={() => {/* scroll to info */ }}
                  sx={{ color: '#1a73e8', textTransform: 'none', fontWeight: 500, p: 0, '&:hover': { textDecoration: 'underline', bgcolor: 'transparent' } }}
                >
                  求人詳細を見る →
                </Button>
              </Box>
            </Box>
          </Box>
        </Box>

        {/* Footer */}
        <Box component="footer" sx={{ py: 2.5, display: 'flex', justifyContent: 'center', gap: 4 }}>
          {['プライバシー', '利用規約', 'ヘルプ'].map(label => (
            <Typography key={label} variant="body2" sx={{ color: 'text.secondary', cursor: 'pointer', '&:hover': { textDecoration: 'underline' } }}>
              {label}
            </Typography>
          ))}
        </Box>
      </Box>
    )
  }

  // ─────────────────────────────────────────────
  // REPORT SCREEN (finished)
  // ─────────────────────────────────────────────
  if (status === 'finished') {
    return (
      <Box sx={{ minHeight: '100vh', bgcolor: BG_DARK, color: '#e8eaed', overflowY: 'auto', p: { xs: 2, md: 4 } }}>
        <Box sx={{ maxWidth: 720, mx: 'auto' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 4 }}>
            <IconButton sx={{ color: '#bdc1c6' }} onClick={() => router.push('/')}>
              <ArrowBackIcon />
            </IconButton>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#e8eaed' }}>面接レポート</Typography>
          </Box>

          {errorMessage && (
            <Paper sx={{ bgcolor: 'rgba(234,67,53,0.15)', border: '1px solid rgba(234,67,53,0.4)', p: 2, mb: 2, borderRadius: 2 }}>
              <Typography variant="body2" sx={{ color: '#f28b82' }}>{errorMessage}</Typography>
            </Paper>
          )}

          {reportStatus === 'pending' && (
            <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
              <Typography sx={{ color: '#e8eaed', mb: 1.5 }}>レポートを生成中です...</Typography>
              <LinearProgress sx={{ bgcolor: '#3c4043', '& .MuiLinearProgress-bar': { bgcolor: '#4285f4' } }} />
            </Paper>
          )}

          {reportStatus === 'ready' && report && (
            <Stack spacing={2}>
              <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                <Typography sx={{ color: PRIMARY, fontWeight: 700, mb: 1 }}>要約</Typography>
                <Typography variant="body2" sx={{ color: '#bdc1c6', lineHeight: 1.8, whiteSpace: 'pre-line' }}>
                  {report.summary_text || '要約がありません'}
                </Typography>
              </Paper>
              <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                <Typography sx={{ color: PRIMARY, fontWeight: 700, mb: 1.5 }}>評価スコア</Typography>
                {scores ? (
                  <Stack spacing={1}>
                    {Object.entries(scores).map(([k, v]) => (
                      <Box key={k} sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Typography variant="body2" sx={{ color: '#bdc1c6' }}>{k}</Typography>
                        <Chip label={String(v)} size="small" sx={{ bgcolor: '#3c4043', color: '#e8eaed', fontWeight: 700 }} />
                      </Box>
                    ))}
                  </Stack>
                ) : (
                  <Typography variant="body2" sx={{ color: '#bdc1c6', whiteSpace: 'pre-line' }}>{report.scores_json}</Typography>
                )}
              </Paper>
              {evidence && (
                <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                  <Typography sx={{ color: PRIMARY, fontWeight: 700, mb: 1.5 }}>根拠</Typography>
                  <Stack spacing={1.5}>
                    {Object.entries(evidence).map(([k, v]) => (
                      <Box key={k}>
                        <Typography variant="body2" sx={{ color: '#9aa0a6', fontWeight: 600 }}>{k}</Typography>
                        <Typography variant="body2" sx={{ color: '#bdc1c6', lineHeight: 1.6 }}>{String(v)}</Typography>
                        <Divider sx={{ mt: 1, borderColor: 'rgba(255,255,255,0.06)' }} />
                      </Box>
                    ))}
                  </Stack>
                </Paper>
              )}
            </Stack>
          )}

          {reportStatus === 'error' && (
            <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(234,67,53,0.3)', p: 3, borderRadius: 2 }}>
              <Typography variant="body2" sx={{ color: '#f28b82' }}>レポート生成に失敗しました。</Typography>
            </Paper>
          )}
        </Box>
      </Box>
    )
  }

  // ─────────────────────────────────────────────
  // SESSION SCREEN (connecting / connected / error)
  // ─────────────────────────────────────────────
  return (
    <Box sx={{ height: '100vh', width: '100vw', bgcolor: BG_DARK, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>

      {/* ── Header ── */}
      <Box component="header" sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 3, py: 1.5, flexShrink: 0, borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box sx={{ width: 36, height: 36, borderRadius: 2, bgcolor: PRIMARY, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <PsychologyIcon sx={{ color: '#fff', fontSize: 20 }} />
          </Box>
          <Box>
            <Typography sx={{ fontWeight: 700, fontSize: 15, color: '#e8eaed', lineHeight: 1.2 }}>AI面接練習</Typography>
            <Typography sx={{ fontSize: 12, color: '#9aa0a6' }}>セッション: {companyName}</Typography>
          </Box>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {isActive && (
            <Box sx={{ display: 'flex', alignItems: 'center', px: 1.5, py: 0.6, borderRadius: 9999, bgcolor: 'rgba(255,255,255,0.08)', gap: 1 }}>
              <Box component="span" sx={{ fontSize: 16 }}>⏱</Box>
              <Typography sx={{ fontSize: 13, color: '#e8eaed', fontWeight: 500 }}>{formatSeconds(remainingSeconds)}</Typography>
            </Box>
          )}
          {isConnected && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: '#34a853', animation: 'pulse 2s infinite', '@keyframes pulse': { '0%,100%': { opacity: 1 }, '50%': { opacity: 0.4 } } }} />
              <Typography sx={{ fontSize: 12, color: '#34a853', fontWeight: 600 }}>接続中</Typography>
            </Box>
          )}
        </Box>
      </Box>

      {/* ── Main ── */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: { xs: 'column', lg: 'row' }, gap: 2, px: 2, pb: '88px', pt: 2, overflow: 'hidden', minHeight: 0 }}>

        {/* Video grid */}
        <Box sx={{ flex: 1, display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2, alignContent: 'center', minHeight: 0 }}>

          {/* AI interviewer tile */}
          <Box sx={{ position: 'relative', aspectRatio: '16/9', borderRadius: 2, overflow: 'hidden', bgcolor: '#303134', boxShadow: '0 8px 32px rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Box sx={{
              width: '100%',
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'radial-gradient(circle at 50% 40%, #3c4043 0%, #202124 100%)',
            }}>
              <Box sx={{
                width: { xs: 120, md: 180 },
                height: { xs: 120, md: 180 },
                borderRadius: '50%',
                boxShadow: aiSpeaking ? `0 0 0 16px rgba(236,91,19,0.15), 0 0 40px rgba(236,91,19,0.1)` : 'none',
                transition: 'box-shadow 0.3s',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <ThreeAvatar gender={avatarGender} audioStream={aiAudioStreamRef.current} level={aiLevel} speaking={aiSpeaking} />
              </Box>
            </Box>

            {/* Speaking indicator */}
            {aiSpeaking && (
              <Box sx={{ position: 'absolute', top: 12, right: 12, width: 10, height: 10, borderRadius: '50%', bgcolor: PRIMARY, animation: 'pulse 1s infinite' }} />
            )}

            {/* Label */}
            <Box sx={{ position: 'absolute', bottom: 12, left: 12, display: 'flex', alignItems: 'center', gap: 1, bgcolor: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(8px)', px: 1.5, py: 0.6, borderRadius: 1.5 }}>
              <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: PRIMARY }} />
              <Typography sx={{ color: '#fff', fontSize: 13, fontWeight: 500 }}>
                面接官AI（{isFemale ? '女性' : '男性'}）
              </Typography>
            </Box>

            {/* Subtitle overlay */}
            {captionsVisible && latestAiText && (
              <Box sx={{ position: 'absolute', bottom: 48, left: '50%', transform: 'translateX(-50%)', maxWidth: '85%', bgcolor: 'rgba(0,0,0,0.72)', borderRadius: 1.5, px: 2, py: 0.8, textAlign: 'center' }}>
                <Typography sx={{ color: '#fff', fontSize: 13, lineHeight: 1.5 }}>{latestAiText}</Typography>
              </Box>
            )}

            {/* Error overlay */}
            {status === 'error' && (
              <Box sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(0,0,0,0.8)', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', p: 3, gap: 2 }}>
                <Typography sx={{ color: '#f28b82', textAlign: 'center', fontSize: 14, lineHeight: 1.6 }}>{errorMessage}</Typography>
                <Button variant="contained" startIcon={<RefreshIcon />} onClick={handleJoin}
                  sx={{ bgcolor: '#4285f4', '&:hover': { bgcolor: '#3367d6' }, textTransform: 'none' }}>
                  再接続する
                </Button>
              </Box>
            )}

            {/* Connecting overlay */}
            {status === 'connecting' && (
              <Box sx={{ position: 'absolute', inset: 0, bgcolor: 'rgba(0,0,0,0.6)', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 2 }}>
                <Typography sx={{ color: '#e8eaed' }}>接続中...</Typography>
                <LinearProgress sx={{ width: 160, bgcolor: '#3c4043', '& .MuiLinearProgress-bar': { bgcolor: PRIMARY } }} />
              </Box>
            )}
          </Box>

          {/* User camera tile */}
          <Box sx={{ position: 'relative', aspectRatio: '16/9', borderRadius: 2, overflow: 'hidden', bgcolor: '#3c4043', boxShadow: `0 0 0 2px rgba(236,91,19,0.3), 0 8px 32px rgba(0,0,0,0.4)`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <video
              ref={sessionVideoRef}
              muted
              playsInline
              style={{ width: '100%', height: '100%', objectFit: 'cover', transform: 'scaleX(-1)', display: cameraEnabled ? 'block' : 'none' }}
            />
            {!cameraEnabled && (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1 }}>
                <VideocamOffIcon sx={{ color: '#9aa0a6', fontSize: 40 }} />
                <Typography sx={{ color: '#9aa0a6', fontSize: 13 }}>カメラオフ</Typography>
              </Box>
            )}
            <Box sx={{ position: 'absolute', bottom: 12, left: 12, display: 'flex', alignItems: 'center', gap: 1, bgcolor: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(8px)', px: 1.5, py: 0.6, borderRadius: 1.5 }}>
              {micEnabled ? <MicIcon sx={{ color: '#fff', fontSize: 16 }} /> : <MicOffIcon sx={{ color: '#ea4335', fontSize: 16 }} />}
              <Typography sx={{ color: '#fff', fontSize: 13, fontWeight: 500 }}>あなた（候補者）</Typography>
            </Box>
            {/* Highlight border overlay */}
            <Box sx={{ position: 'absolute', inset: 0, border: `2px solid rgba(236,91,19,0.25)`, borderRadius: 2, pointerEvents: 'none' }} />
          </Box>
        </Box>

        {/* ── Right sidebar: Transcript ── */}
        <Box sx={{ width: { xs: '100%', lg: 360 }, display: 'flex', flexDirection: 'column', bgcolor: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, overflow: 'hidden', flexShrink: 0 }}>
          {/* Sidebar header */}
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 2, py: 1.5, borderBottom: '1px solid rgba(255,255,255,0.08)' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Box component="span" sx={{ fontSize: 20 }}>💬</Box>
              <Typography sx={{ fontWeight: 700, color: '#e8eaed', fontSize: 14 }}>リアルタイム字幕</Typography>
            </Box>
            {isConnected && (
              <Box sx={{ bgcolor: `${PRIMARY}20`, border: `1px solid ${PRIMARY}40`, px: 1, py: 0.3, borderRadius: 1 }}>
                <Typography sx={{ color: PRIMARY, fontSize: 10, fontWeight: 700, letterSpacing: 1 }}>LIVE</Typography>
              </Box>
            )}
          </Box>

          {/* Transcript list */}
          <Box sx={{ flex: 1, overflowY: 'auto', p: 2, display: 'flex', flexDirection: 'column', gap: 2.5 }}>
            {utterances.length === 0 && !partialAi && (
              <Typography sx={{ color: '#5f6368', fontSize: 13, textAlign: 'center', mt: 4 }}>
                面接が始まると字幕がここに表示されます
              </Typography>
            )}
            {utterances.map((u, i) => (
              <Box key={i} sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: u.role === 'ai' ? 'flex-start' : 'flex-end' }}>
                <Typography sx={{ fontSize: 10, fontWeight: 700, color: u.role === 'ai' ? '#9aa0a6' : PRIMARY, letterSpacing: 1, textTransform: 'uppercase' }}>
                  {u.role === 'ai' ? `面接官AI（${isFemale ? '女性' : '男性'}）` : 'あなた'}
                </Typography>
                <Box sx={{
                  bgcolor: u.role === 'ai' ? 'rgba(255,255,255,0.06)' : PRIMARY,
                  px: 1.5, py: 1, borderRadius: u.role === 'ai' ? '0 12px 12px 12px' : '12px 0 12px 12px',
                  maxWidth: '90%',
                }}>
                  <Typography sx={{ fontSize: 13, color: '#e8eaed', lineHeight: 1.6 }}>{u.text}</Typography>
                </Box>
              </Box>
            ))}

            {/* Partial AI (typing) */}
            {partialAi && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                <Typography sx={{ fontSize: 10, fontWeight: 700, color: '#9aa0a6', letterSpacing: 1, textTransform: 'uppercase' }}>
                  面接官AI
                </Typography>
                <Box sx={{ bgcolor: 'rgba(255,255,255,0.06)', px: 1.5, py: 1, borderRadius: '0 12px 12px 12px', maxWidth: '90%', opacity: 0.7 }}>
                  <Typography sx={{ fontSize: 13, color: '#e8eaed', lineHeight: 1.6 }}>{partialAi}</Typography>
                </Box>
              </Box>
            )}

            {/* Partial user (typing) */}
            {partialUser && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-end' }}>
                <Typography sx={{ fontSize: 10, fontWeight: 700, color: PRIMARY, letterSpacing: 1, textTransform: 'uppercase' }}>あなた</Typography>
                <Box sx={{ bgcolor: `${PRIMARY}80`, px: 1.5, py: 1, borderRadius: '12px 0 12px 12px', maxWidth: '90%', opacity: 0.7 }}>
                  <Typography sx={{ fontSize: 13, color: '#fff', lineHeight: 1.6 }}>{partialUser}</Typography>
                </Box>
              </Box>
            )}

            {/* AI tip */}
            {utterances.length >= 2 && (
              <Box sx={{ p: 1.5, bgcolor: `${PRIMARY}10`, border: `1px solid ${PRIMARY}30`, borderRadius: 2, display: 'flex', gap: 1.5 }}>
                <LightbulbIcon sx={{ color: PRIMARY, fontSize: 20, flexShrink: 0 }} />
                <Box>
                  <Typography sx={{ fontSize: 11, fontWeight: 700, color: PRIMARY, mb: 0.3 }}>AIヒント</Typography>
                  <Typography sx={{ fontSize: 12, color: '#9aa0a6', lineHeight: 1.5 }}>
                    具体的なエピソードを交えて回答すると、より説得力が増します。
                  </Typography>
                </Box>
              </Box>
            )}

            <div ref={transcriptEndRef} />
          </Box>

          {/* Note input */}
          <Box sx={{ p: 1.5, borderTop: '1px solid rgba(255,255,255,0.08)' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', bgcolor: 'rgba(255,255,255,0.06)', borderRadius: 2, px: 1.5, py: 0.5 }}>
              <InputBase
                value={noteInput}
                onChange={e => setNoteInput(e.target.value)}
                placeholder="メモやヒントのリクエストを入力..."
                sx={{ flex: 1, color: '#e8eaed', fontSize: 13, '& ::placeholder': { color: '#5f6368' } }}
              />
              <IconButton size="small" sx={{ color: noteInput ? PRIMARY : '#5f6368' }}>
                <SendIcon fontSize="small" />
              </IconButton>
            </Box>
          </Box>
        </Box>
      </Box>

      {/* ── Bottom control bar ── */}
      <Box sx={{
        position: 'fixed', bottom: 0, left: 0, right: 0,
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        px: { xs: 2, md: 4 }, py: 1.5,
        bgcolor: 'rgba(32,33,36,0.97)',
        borderTop: '1px solid rgba(255,255,255,0.06)',
        zIndex: 100,
      }}>
        {/* Left: session info */}
        <Box sx={{ display: { xs: 'none', md: 'flex' }, alignItems: 'center', gap: 1 }}>
          <Typography sx={{ fontSize: 13, color: '#9aa0a6', fontWeight: 500 }}>
            {companyName} 面接
          </Typography>
          {isActive && (
            <LinearProgress
              variant="determinate"
              value={progress}
              sx={{ width: 80, height: 4, borderRadius: 2, bgcolor: '#3c4043', '& .MuiLinearProgress-bar': { bgcolor: PRIMARY } }}
            />
          )}
        </Box>

        {/* Center: controls */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: { xs: 1, md: 1.5 }, mx: 'auto' }}>
          <Tooltip title={micEnabled ? 'マイクをオフ' : 'マイクをオン'}>
            <span>
              <IconButton onClick={toggleMic} disabled={!isConnected} sx={{ bgcolor: micEnabled ? 'rgba(255,255,255,0.08)' : '#ea4335', width: 48, height: 48, '&:hover': { bgcolor: micEnabled ? 'rgba(255,255,255,0.15)' : '#c5221f' }, '&:disabled': { bgcolor: 'rgba(255,255,255,0.04)' } }}>
                {micEnabled ? <MicIcon sx={{ color: '#e8eaed' }} /> : <MicOffIcon sx={{ color: '#fff' }} />}
              </IconButton>
            </span>
          </Tooltip>

          <Tooltip title={cameraEnabled ? 'カメラをオフ' : 'カメラをオン'}>
            <span>
              <IconButton onClick={toggleCamera} disabled={!isConnected} sx={{ bgcolor: cameraEnabled ? 'rgba(255,255,255,0.08)' : '#ea4335', width: 48, height: 48, '&:hover': { bgcolor: cameraEnabled ? 'rgba(255,255,255,0.15)' : '#c5221f' }, '&:disabled': { bgcolor: 'rgba(255,255,255,0.04)' } }}>
                {cameraEnabled ? <VideocamIcon sx={{ color: '#e8eaed' }} /> : <VideocamOffIcon sx={{ color: '#fff' }} />}
              </IconButton>
            </span>
          </Tooltip>

          <Tooltip title={captionsVisible ? '字幕をオフ' : '字幕をオン'}>
            <IconButton onClick={() => setCaptionsVisible(p => !p)} sx={{ bgcolor: captionsVisible ? `${PRIMARY}30` : 'rgba(255,255,255,0.08)', width: 48, height: 48, '&:hover': { bgcolor: captionsVisible ? `${PRIMARY}40` : 'rgba(255,255,255,0.15)' } }}>
              <ClosedCaptionIcon sx={{ color: captionsVisible ? PRIMARY : '#9aa0a6' }} />
            </IconButton>
          </Tooltip>

          {/* End call / Join button */}
          {!isActive ? (
            <Button
              variant="contained"
              onClick={handleJoin}
              sx={{ bgcolor: '#34a853', '&:hover': { bgcolor: '#2d8f47' }, borderRadius: 9999, px: 3, py: 1, fontWeight: 600, textTransform: 'none', fontSize: 14 }}
            >
              面接を開始
            </Button>
          ) : (
            <Tooltip title="面接を終了">
              <Button
                variant="contained"
                startIcon={<CallEndIcon />}
                onClick={() => handleStop(false)}
                sx={{ bgcolor: '#ea4335', '&:hover': { bgcolor: '#c5221f' }, borderRadius: 9999, px: 3, py: 1, fontWeight: 600, textTransform: 'none', fontSize: 14 }}
              >
                終了
              </Button>
            </Tooltip>
          )}
        </Box>

        {/* Right: secondary actions */}
        <Box sx={{ display: { xs: 'none', md: 'flex' }, alignItems: 'center', gap: 1 }}>
          <Tooltip title="募集要項">
            <IconButton sx={{ color: '#9aa0a6', '&:hover': { bgcolor: 'rgba(255,255,255,0.08)' } }} onClick={() => {}}>
              <InfoOutlinedIcon />
            </IconButton>
          </Tooltip>
          <Typography variant="caption" sx={{ color: '#5f6368', ml: 1 }}>
            推定 ${estimatedCost.toFixed(2)}
          </Typography>
        </Box>
      </Box>

      <audio ref={audioRef} autoPlay />
    </Box>
  )
}

function getNextAvatarGender(): 'male' | 'female' {
  try {
    const key = 'interview_avatar_index'
    const current = Number(localStorage.getItem(key) || '0')
    const next = current + 1
    localStorage.setItem(key, String(next))
    return next % 2 === 0 ? 'female' : 'male'
  } catch {
    return 'male'
  }
}
