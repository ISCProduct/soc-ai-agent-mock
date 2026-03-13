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
import * as THREE from 'three'
import { authService, User } from '@/lib/auth'
import { interviewApi, interviewLimits, InterviewReport, InterviewSession } from '@/lib/interview'
import ThreeAvatar from './components/ThreeAvatar'

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

export default function InterviewPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [status, setStatus] = useState<'idle' | 'connecting' | 'connected' | 'error' | 'finished'>('idle')
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
  const [jobInfoOpen, setJobInfoOpen] = useState(false)

  const pcRef = useRef<RTCPeerConnection | null>(null)
  const dcRef = useRef<RTCDataChannel | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const aiAudioStreamRef = useRef<MediaStream | null>(null)
  const aiAudioCtxRef = useRef<AudioContext | null>(null)
  const aiAnalyserRef = useRef<AnalyserNode | null>(null)
  const aiAnimationRef = useRef<number | null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const sessionStartRef = useRef<number | null>(null)

  useEffect(() => {
    const storedUser = authService.getStoredUser()
    if (!storedUser) {
      router.replace('/login')
      return
    }
    if (storedUser.target_level !== '新卒' && storedUser.target_level !== '中途') {
      router.replace('/onboarding')
      return
    }
    setUser(storedUser)
    setLoading(false)
  }, [router])

  useEffect(() => {
    let cancelled = false
    const fromQuery = (() => {
      try {
        const params = new URLSearchParams(window.location.search)
        return Number(params.get('company_id') || '')
      } catch {
        return NaN
      }
    })()
    const fromStorage = (() => {
      try {
        return Number(localStorage.getItem('interview_company_id') || '')
      } catch {
        return NaN
      }
    })()
    const companyId = Number.isFinite(fromQuery) && fromQuery > 0
      ? fromQuery
      : Number.isFinite(fromStorage) && fromStorage > 0
      ? fromStorage
      : NaN

    const loadCompany = async () => {
      try {
        if (Number.isFinite(companyId)) {
          const detailRes = await fetch(`/api/companies/${companyId}`, { cache: 'no-store' })
          if (detailRes.ok) {
            const detail = await detailRes.json()
            if (!cancelled) setInterviewCompany(detail)
            return
          }
        }
        const fallbackRes = await fetch('/api/companies?limit=1&offset=0', { cache: 'no-store' })
        if (!fallbackRes.ok) return
        const fallback = await fallbackRes.json()
        const first = Array.isArray(fallback?.companies) ? fallback.companies[0] : null
        if (!cancelled && first) setInterviewCompany(first)
      } catch {
        // ignore
      }
    }

    loadCompany()
    return () => { cancelled = true }
  }, [])

  useEffect(() => {
    return () => { cleanupConnection() }
  }, [])

  const formatSeconds = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}:${s.toString().padStart(2, '0')}`
  }

  const parseJsonSafe = (value?: string) => {
    if (!value) return null
    try { return JSON.parse(value) } catch { return null }
  }

  const cleanupConnection = () => {
    if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null }
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null }
    if (aiAnimationRef.current) { cancelAnimationFrame(aiAnimationRef.current); aiAnimationRef.current = null }
    if (aiAudioCtxRef.current) {
      aiAudioCtxRef.current.close().catch(() => undefined)
      aiAudioCtxRef.current = null
      aiAnalyserRef.current = null
    }
    if (dcRef.current) { dcRef.current.close(); dcRef.current = null }
    if (pcRef.current) { pcRef.current.close(); pcRef.current = null }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop())
      streamRef.current = null
    }
  }

  const startTimer = () => {
    sessionStartRef.current = Date.now()
    timerRef.current = setInterval(() => {
      if (!sessionStartRef.current) return
      const elapsed = Math.floor((Date.now() - sessionStartRef.current) / 1000)
      const remaining = Math.max(0, interviewLimits.maxMinutes * 60 - elapsed)
      setRemainingSeconds(remaining)
      const cost = (elapsed / 60) * interviewLimits.costPerMinuteUSD
      setEstimatedCost(cost)
      if (remaining <= 0 || cost >= interviewLimits.maxCostUSD) {
        handleStop(true)
      }
    }, 1000)
  }

  const handleStart = async () => {
    if (!user) return
    setErrorMessage(null)
    setUtterances([])
    setPartialUser('')
    setPartialAi('')
    setReport(null)
    setReportStatus('idle')
    setRemainingSeconds(interviewLimits.maxMinutes * 60)
    setEstimatedCost(0)
    setMicEnabled(true)
    setCameraEnabled(true)

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

  const parseStartError = (error: any): string => {
    const msg: string = error?.message || ''
    if (msg.includes('NotAllowedError') || msg.toLowerCase().includes('permission denied') || msg.toLowerCase().includes('denied')) {
      return 'マイクとカメラへのアクセスが拒否されました。ブラウザのアドレスバー横のカメラ/マイクアイコンから権限を許可してください。'
    }
    if (msg.includes('NotFoundError') || msg.includes('DevicesNotFoundError') || msg.toLowerCase().includes('not found')) {
      return 'マイクまたはカメラが見つかりません。デバイスが正しく接続されているか確認してください。'
    }
    if (msg.toLowerCase().includes('api key') || msg.toLowerCase().includes('unauthorized') || msg.includes('401')) {
      return 'AIサービスへの接続に失敗しました。サービス設定を確認してください。（OpenAI APIキー未設定の可能性があります）'
    }
    if (msg.toLowerCase().includes('forbidden') || msg.includes('403')) {
      return 'この操作を行う権限がありません。'
    }
    return msg || '接続に失敗しました。ネットワーク状況を確認してから再試行してください。'
  }

  const handleStop = async (forced = false) => {
    if (!user || !session) {
      cleanupConnection()
      setStatus('finished')
      return
    }
    cleanupConnection()
    try {
      await interviewApi.finishSession(session.id, user.user_id)
    } catch (error: any) {
      setErrorMessage(error?.message || '終了処理に失敗しました')
    }
    setStatus('finished')
    setReportStatus('pending')
    if (forced) {
      setErrorMessage('時間またはコスト上限に達したため面接を終了しました。')
    }
    startReportPolling(session.id, user.user_id)
  }

  const startReportPolling = (sessionId: number, userId: number) => {
    if (pollRef.current) clearInterval(pollRef.current)
    pollRef.current = setInterval(async () => {
      try {
        const detail = await interviewApi.getDetail(sessionId, userId)
        if (detail.report) {
          setReport(detail.report)
          setReportStatus('ready')
          clearInterval(pollRef.current!)
          pollRef.current = null
        }
      } catch {
        setReportStatus('error')
      }
    }, 3000)
  }

  const startConnection = async (token: string, sessionId: number) => {
    const pc = new RTCPeerConnection()
    pcRef.current = pc

    const dc = pc.createDataChannel('oai-events')
    dcRef.current = dc
    dc.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data)
        handleRealtimeEvent(event, sessionId)
      } catch {
        // ignore
      }
    }

    pc.ontrack = (event) => {
      if (audioRef.current) {
        audioRef.current.srcObject = event.streams[0]
        audioRef.current.play().catch(() => undefined)
        setupAiAudioAnalyser(event.streams[0])
      }
    }

    let stream: MediaStream
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
    } catch (err: any) {
      if (err.name === 'NotAllowedError' || err.name === 'PermissionDeniedError') {
        throw new Error('NotAllowedError: マイクとカメラへのアクセスが拒否されました。')
      }
      if (err.name === 'NotFoundError' || err.name === 'DevicesNotFoundError') {
        throw new Error('NotFoundError: マイクまたはカメラが見つかりません。')
      }
      throw err
    }

    streamRef.current = stream
    if (videoRef.current) {
      videoRef.current.srcObject = stream
      videoRef.current.play().catch(() => undefined)
    }
    stream.getTracks().forEach(track => pc.addTrack(track, stream))

    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)

    const response = await fetch('https://api.openai.com/v1/realtime/calls', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/sdp',
      },
      body: offer.sdp,
    })
    if (!response.ok) {
      throw new Error(await response.text())
    }
    const answer = await response.text()
    await pc.setRemoteDescription({ type: 'answer', sdp: answer })

    dc.onopen = () => {
      const event = {
        type: 'conversation.item.create',
        item: {
          type: 'message',
          role: 'user',
          content: [{ type: 'input_text', text: '面接を開始してください。最初の質問をお願いします。' }],
        },
      }
      dc.send(JSON.stringify(event))
      dc.send(JSON.stringify({ type: 'response.create', response: { modalities: ['audio'] } }))
    }
  }

  const setupAiAudioAnalyser = (stream: MediaStream) => {
    if (aiAudioCtxRef.current) return
    aiAudioStreamRef.current = stream
    const audioCtx = new AudioContext()
    aiAudioCtxRef.current = audioCtx
    const source = audioCtx.createMediaStreamSource(stream)
    const analyser = audioCtx.createAnalyser()
    analyser.fftSize = 512
    aiAnalyserRef.current = analyser
    source.connect(analyser)
    const data = new Uint8Array(analyser.frequencyBinCount)
    const tick = () => {
      analyser.getByteTimeDomainData(data)
      let sum = 0
      for (let i = 0; i < data.length; i += 1) {
        const v = (data[i] - 128) / 128
        sum += v * v
      }
      const rms = Math.sqrt(sum / data.length)
      const level = Math.min(1, rms * 2.5)
      setAiLevel(level)
      setAiSpeaking(level > 0.08)
      aiAnimationRef.current = requestAnimationFrame(tick)
    }
    tick()
  }

  const handleRealtimeEvent = async (event: any, sessionId: number) => {
    if (!user) return
    switch (event.type) {
      case 'conversation.item.input_audio_transcription.delta': {
        if (event.delta) setPartialUser(prev => prev + event.delta)
        break
      }
      case 'conversation.item.input_audio_transcription.completed': {
        const text = (event.transcript || event.text || partialUser).trim()
        if (text) {
          setUtterances(prev => [...prev, { role: 'user', text }])
          setPartialUser('')
          try { await interviewApi.saveUtterance(sessionId, user.user_id, 'user', text) } catch { /* ignore */ }
        }
        break
      }
      case 'response.audio_transcript.delta': {
        if (event.delta) setPartialAi(prev => prev + event.delta)
        break
      }
      case 'response.audio_transcript.done': {
        const text = (event.transcript || partialAi).trim()
        if (text) {
          setUtterances(prev => [...prev, { role: 'ai', text }])
          setPartialAi('')
          try { await interviewApi.saveUtterance(sessionId, user.user_id, 'ai', text) } catch { /* ignore */ }
        }
        break
      }
      default:
        break
    }
  }

  const toggleMic = () => {
    if (!streamRef.current) return
    const newEnabled = !micEnabled
    streamRef.current.getAudioTracks().forEach(t => { t.enabled = newEnabled })
    setMicEnabled(newEnabled)
  }

  const toggleCamera = () => {
    if (!streamRef.current) return
    const newEnabled = !cameraEnabled
    streamRef.current.getVideoTracks().forEach(t => { t.enabled = newEnabled })
    setCameraEnabled(newEnabled)
  }

  if (loading || !user) return null

  const isActive = status === 'connecting' || status === 'connected'
  const isConnected = status === 'connected'
  const progress = Math.min(100, Math.round(((interviewLimits.maxMinutes * 60 - remainingSeconds) / (interviewLimits.maxMinutes * 60)) * 100))
  const scores = report ? parseJsonSafe(report.scores_json) : null
  const evidence = report ? parseJsonSafe(report.evidence_json) : null
  const isFemaleAvatar = avatarGender === 'female'
  const companyName = interviewCompany?.name || '企業情報を読み込み中'
  const employeeText = interviewCompany?.employee_count ? `${interviewCompany.employee_count}名` : '非公開'
  const recruitingText = [
    '【募集背景】',
    interviewCompany?.description || '企業情報の取得後に表示されます。',
    '',
    '【仕事内容】',
    interviewCompany?.main_business || interviewCompany?.industry || '詳細は面接内でご案内します。',
    '',
    '【職場環境】',
    interviewCompany?.work_style || '勤務形態は選考でご説明します。',
    '',
    '【企業文化・福利厚生】',
    `${interviewCompany?.culture || 'チームで成果を重視する文化'} / ${interviewCompany?.welfare_details || '福利厚生情報は準備中です。'}`,
    '',
    `【勤務地・人数】 ${interviewCompany?.location || '勤務地未設定'} / ${employeeText}`,
  ].join('\n')

  const latestSubtitle = partialAi || (utterances.length > 0 ? utterances[utterances.length - 1].text : '')

  return (
    <Box
      sx={{
        width: '100vw',
        height: '100vh',
        bgcolor: '#202124',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      {/* ─── ヘッダー ─── */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          px: 2,
          py: 1.2,
          bgcolor: 'rgba(32,33,36,0.95)',
          borderBottom: '1px solid rgba(255,255,255,0.08)',
          flexShrink: 0,
          zIndex: 10,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <IconButton size="small" sx={{ color: '#bdc1c6' }} onClick={() => router.push('/')}>
            <ArrowBackIcon fontSize="small" />
          </IconButton>
          <Tooltip title="募集要項を見る">
            <IconButton size="small" sx={{ color: '#bdc1c6' }} onClick={() => setJobInfoOpen(true)}>
              <InfoOutlinedIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Typography variant="body2" sx={{ color: '#e8eaed', fontWeight: 600 }}>
            {companyName}
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          {isActive && (
            <Chip
              size="small"
              label={formatSeconds(remainingSeconds)}
              sx={{ bgcolor: '#3c4043', color: '#e8eaed', fontWeight: 600 }}
            />
          )}
          {isActive && (
            <Typography variant="caption" sx={{ color: '#9aa0a6' }}>
              推定 ${estimatedCost.toFixed(2)}
            </Typography>
          )}
          {status === 'connected' && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: '#34a853', animation: 'pulse 2s infinite', '@keyframes pulse': { '0%,100%': { opacity: 1 }, '50%': { opacity: 0.4 } } }} />
              <Typography variant="caption" sx={{ color: '#34a853', fontWeight: 600 }}>接続中</Typography>
            </Box>
          )}
        </Box>
      </Box>

      {/* ─── メインエリア ─── */}
      <Box sx={{ flex: 1, position: 'relative', overflow: 'hidden' }}>

        {/* AIアバター（中央・大） */}
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Box
            sx={{
              width: { xs: 260, md: 420, lg: 520 },
              height: { xs: 260, md: 420, lg: 520 },
              borderRadius: '50%',
              background: 'radial-gradient(circle at 48% 36%, rgba(255,255,255,0.12) 0%, rgba(60,64,67,0.6) 60%, transparent 100%)',
              boxShadow: aiSpeaking
                ? '0 0 0 24px rgba(66,133,244,0.15), 0 0 60px rgba(66,133,244,0.1)'
                : 'none',
              transition: 'box-shadow 0.3s ease',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <ThreeAvatar
              gender={avatarGender}
              audioStream={aiAudioStreamRef.current}
              level={aiLevel}
              speaking={aiSpeaking}
            />
          </Box>
        </Box>

        {/* 面接官ラベル */}
        <Box sx={{ position: 'absolute', top: 16, left: 20, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{
            width: 8, height: 8, borderRadius: '50%',
            bgcolor: aiSpeaking ? '#ea4335' : '#34a853',
            transition: 'background-color 0.2s',
          }} />
          <Typography variant="body2" sx={{ color: '#e8eaed', fontWeight: 600 }}>
            面接官AI（{isFemaleAvatar ? '女性' : '男性'}）
          </Typography>
          {aiSpeaking && (
            <Typography variant="caption" sx={{ color: '#9aa0a6' }}>話しています...</Typography>
          )}
        </Box>

        {/* 字幕オーバーレイ */}
        {latestSubtitle && (
          <Box
            sx={{
              position: 'absolute',
              bottom: { xs: 90, md: 100 },
              left: '50%',
              transform: 'translateX(-50%)',
              maxWidth: '70%',
              bgcolor: 'rgba(0,0,0,0.72)',
              borderRadius: 2,
              px: 2.5,
              py: 1,
              textAlign: 'center',
            }}
          >
            <Typography sx={{ color: '#ffffff', fontSize: { xs: 14, md: 16 }, lineHeight: 1.5 }}>
              {latestSubtitle}
            </Typography>
          </Box>
        )}

        {/* 自分の映像（PiP） */}
        <Box
          sx={{
            position: 'absolute',
            bottom: { xs: 90, md: 96 },
            right: 16,
            width: { xs: 120, md: 180 },
            height: { xs: 90, md: 135 },
            borderRadius: 2,
            overflow: 'hidden',
            bgcolor: '#3c4043',
            border: '2px solid rgba(255,255,255,0.12)',
            boxShadow: '0 4px 16px rgba(0,0,0,0.4)',
          }}
        >
          <video
            ref={videoRef}
            muted
            playsInline
            style={{
              width: '100%',
              height: '100%',
              objectFit: 'cover',
              transform: 'scaleX(-1)',
              display: cameraEnabled ? 'block' : 'none',
            }}
          />
          {!cameraEnabled && (
            <Box sx={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <VideocamOffIcon sx={{ color: '#9aa0a6' }} />
            </Box>
          )}
          <Box sx={{ position: 'absolute', bottom: 4, left: 8 }}>
            <Typography variant="caption" sx={{ color: '#e8eaed', fontSize: 11 }}>あなた</Typography>
          </Box>
        </Box>

        {/* エラー表示 */}
        {status === 'error' && errorMessage && (
          <Box
            sx={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              bgcolor: 'rgba(32,33,36,0.95)',
              border: '1px solid rgba(234,67,53,0.5)',
              borderRadius: 3,
              p: 3,
              maxWidth: 480,
              width: '90%',
              textAlign: 'center',
            }}
          >
            <Typography variant="body1" sx={{ color: '#f28b82', mb: 2, lineHeight: 1.6 }}>
              {errorMessage}
            </Typography>
            <Button
              variant="contained"
              startIcon={<RefreshIcon />}
              onClick={handleStart}
              sx={{ bgcolor: '#4285f4', '&:hover': { bgcolor: '#3367d6' } }}
            >
              再接続する
            </Button>
          </Box>
        )}

        {/* 接続待機画面 */}
        {status === 'idle' && (
          <Box
            sx={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              textAlign: 'center',
              pointerEvents: 'none',
            }}
          >
            <Typography variant="h6" sx={{ color: '#9aa0a6', mb: 1 }}>
              面接を開始するには下のボタンを押してください
            </Typography>
            <Typography variant="body2" sx={{ color: '#5f6368' }}>
              マイクとカメラの使用許可が必要です
            </Typography>
          </Box>
        )}

        {/* 接続中インジケーター */}
        {status === 'connecting' && (
          <Box
            sx={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              textAlign: 'center',
            }}
          >
            <Typography variant="body1" sx={{ color: '#e8eaed', mb: 1 }}>接続中...</Typography>
            <LinearProgress sx={{ width: 200, mx: 'auto', bgcolor: '#3c4043', '& .MuiLinearProgress-bar': { bgcolor: '#4285f4' } }} />
          </Box>
        )}
      </Box>

      {/* ─── ボトムバー ─── */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: { xs: 1.5, md: 2.5 },
          py: 1.5,
          px: 2,
          bgcolor: 'rgba(32,33,36,0.97)',
          borderTop: '1px solid rgba(255,255,255,0.06)',
          flexShrink: 0,
        }}
      >
        {/* マイクトグル */}
        <Tooltip title={micEnabled ? 'マイクをオフ' : 'マイクをオン'}>
          <span>
            <IconButton
              onClick={toggleMic}
              disabled={!isConnected}
              sx={{
                bgcolor: micEnabled ? '#3c4043' : '#ea4335',
                '&:hover': { bgcolor: micEnabled ? '#5f6368' : '#c5221f' },
                '&:disabled': { bgcolor: '#2a2b2e' },
                width: 52, height: 52,
              }}
            >
              {micEnabled
                ? <MicIcon sx={{ color: '#e8eaed' }} />
                : <MicOffIcon sx={{ color: '#fff' }} />
              }
            </IconButton>
          </span>
        </Tooltip>

        {/* カメラトグル */}
        <Tooltip title={cameraEnabled ? 'カメラをオフ' : 'カメラをオン'}>
          <span>
            <IconButton
              onClick={toggleCamera}
              disabled={!isConnected}
              sx={{
                bgcolor: cameraEnabled ? '#3c4043' : '#ea4335',
                '&:hover': { bgcolor: cameraEnabled ? '#5f6368' : '#c5221f' },
                '&:disabled': { bgcolor: '#2a2b2e' },
                width: 52, height: 52,
              }}
            >
              {cameraEnabled
                ? <VideocamIcon sx={{ color: '#e8eaed' }} />
                : <VideocamOffIcon sx={{ color: '#fff' }} />
              }
            </IconButton>
          </span>
        </Tooltip>

        {/* 開始/終了ボタン */}
        {!isActive && status !== 'finished' ? (
          <Button
            variant="contained"
            onClick={handleStart}
            sx={{
              bgcolor: '#34a853',
              '&:hover': { bgcolor: '#2d8f47' },
              borderRadius: 8,
              px: 4,
              py: 1.2,
              fontWeight: 700,
              fontSize: 15,
            }}
          >
            面接を開始
          </Button>
        ) : isActive ? (
          <Tooltip title="面接を終了">
            <IconButton
              onClick={() => handleStop(false)}
              sx={{
                bgcolor: '#ea4335',
                '&:hover': { bgcolor: '#c5221f' },
                width: 56, height: 56,
              }}
            >
              <CallEndIcon sx={{ color: '#fff', fontSize: 28 }} />
            </IconButton>
          </Tooltip>
        ) : null}

        {/* タイムプログレス（接続中のみ） */}
        {isActive && (
          <Box sx={{ width: { xs: 80, md: 120 } }}>
            <LinearProgress
              variant="determinate"
              value={progress}
              sx={{
                height: 6,
                borderRadius: 3,
                bgcolor: '#3c4043',
                '& .MuiLinearProgress-bar': { bgcolor: '#fbbc04' },
              }}
            />
            <Typography variant="caption" sx={{ color: '#9aa0a6', display: 'block', textAlign: 'center', mt: 0.3, fontSize: 11 }}>
              {formatSeconds(remainingSeconds)}
            </Typography>
          </Box>
        )}
      </Box>

      {/* ─── 面接レポート（終了後） ─── */}
      {status === 'finished' && (
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            bgcolor: '#202124',
            overflowY: 'auto',
            zIndex: 20,
            p: { xs: 2, md: 4 },
          }}
        >
          <Box sx={{ maxWidth: 720, mx: 'auto' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
              <IconButton sx={{ color: '#bdc1c6' }} onClick={() => router.push('/')}>
                <ArrowBackIcon />
              </IconButton>
              <Typography variant="h5" sx={{ color: '#e8eaed', fontWeight: 700 }}>
                面接レポート
              </Typography>
            </Box>

            {errorMessage && (
              <Box sx={{ bgcolor: 'rgba(234,67,53,0.15)', border: '1px solid rgba(234,67,53,0.4)', borderRadius: 2, p: 2, mb: 2 }}>
                <Typography variant="body2" sx={{ color: '#f28b82' }}>{errorMessage}</Typography>
              </Box>
            )}

            {reportStatus === 'pending' && (
              <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                <Typography variant="body1" sx={{ color: '#e8eaed', mb: 1.5 }}>レポートを生成中です。しばらくお待ちください...</Typography>
                <LinearProgress sx={{ bgcolor: '#3c4043', '& .MuiLinearProgress-bar': { bgcolor: '#4285f4' } }} />
              </Paper>
            )}

            {reportStatus === 'ready' && report && (
              <Stack spacing={2}>
                <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                  <Typography variant="subtitle1" sx={{ color: '#8ab4f8', fontWeight: 700, mb: 1 }}>要約</Typography>
                  <Typography variant="body2" sx={{ color: '#bdc1c6', whiteSpace: 'pre-line', lineHeight: 1.7 }}>
                    {report.summary_text || '要約がありません'}
                  </Typography>
                </Paper>

                <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                  <Typography variant="subtitle1" sx={{ color: '#8ab4f8', fontWeight: 700, mb: 1.5 }}>評価スコア</Typography>
                  {scores ? (
                    <Stack spacing={1}>
                      {Object.entries(scores).map(([key, value]) => (
                        <Box key={key} sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Typography variant="body2" sx={{ color: '#bdc1c6' }}>{key}</Typography>
                          <Chip
                            label={String(value)}
                            size="small"
                            sx={{ bgcolor: '#3c4043', color: '#e8eaed', fontWeight: 700 }}
                          />
                        </Box>
                      ))}
                    </Stack>
                  ) : (
                    <Typography variant="body2" sx={{ color: '#bdc1c6', whiteSpace: 'pre-line' }}>{report.scores_json}</Typography>
                  )}
                </Paper>

                {evidence && (
                  <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                    <Typography variant="subtitle1" sx={{ color: '#8ab4f8', fontWeight: 700, mb: 1.5 }}>根拠</Typography>
                    <Stack spacing={1}>
                      {Object.entries(evidence).map(([key, value]) => (
                        <Box key={key}>
                          <Typography variant="body2" sx={{ color: '#9aa0a6', fontWeight: 600 }}>{key}</Typography>
                          <Typography variant="body2" sx={{ color: '#bdc1c6', lineHeight: 1.6 }}>{String(value)}</Typography>
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

            {reportStatus === 'idle' && (
              <Paper sx={{ bgcolor: '#2d2e31', border: '1px solid rgba(255,255,255,0.08)', p: 3, borderRadius: 2 }}>
                <Typography variant="body2" sx={{ color: '#9aa0a6' }}>面接終了後にレポートが表示されます。</Typography>
              </Paper>
            )}
          </Box>
        </Box>
      )}

      {/* ─── 求人情報ドロワー ─── */}
      <Drawer
        anchor="left"
        open={jobInfoOpen}
        onClose={() => setJobInfoOpen(false)}
        PaperProps={{
          sx: {
            width: { xs: '85vw', md: 360 },
            bgcolor: '#2d2e31',
            color: '#e8eaed',
            p: 3,
          },
        }}
      >
        <Box sx={{ mb: 2 }}>
          <Typography variant="h6" sx={{ fontWeight: 700, color: '#e8eaed' }}>
            {companyName}
          </Typography>
          <Typography variant="caption" sx={{ color: '#9aa0a6' }}>募集要項</Typography>
        </Box>
        <Divider sx={{ borderColor: 'rgba(255,255,255,0.1)', mb: 2 }} />
        <Typography
          sx={{
            fontSize: 14,
            color: '#bdc1c6',
            whiteSpace: 'pre-line',
            lineHeight: 1.9,
          }}
        >
          {recruitingText}
        </Typography>
      </Drawer>

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

function InterviewerFallbackAvatar({
  gender,
  level,
  speaking,
}: {
  gender: 'male' | 'female'
  level: number
  speaking: boolean
}) {
  const mouthOpen = Math.max(4, Math.min(18, Math.round(4 + level * 20)))
  const hairColor = gender === 'female' ? '#4f3326' : '#2b2b34'
  const suitColor = gender === 'female' ? '#48607f' : '#2f4a66'
  const accentColor = gender === 'female' ? '#e6d8c4' : '#c8d6e5'

  return (
    <Box
      sx={{
        width: { xs: 190, md: 280 },
        height: { xs: 190, md: 280 },
        borderRadius: '50%',
        overflow: 'hidden',
        position: 'relative',
        display: 'grid',
        placeItems: 'center',
        background: 'radial-gradient(circle at 48% 30%, #fefcf7 0%, #f2eadf 38%, #d7c7b0 100%)',
        boxShadow: speaking
          ? '0 0 0 10px rgba(30, 64, 175, 0.15), inset 0 -8px 20px rgba(0,0,0,0.1)'
          : 'inset 0 -8px 20px rgba(0,0,0,0.08)',
        transform: speaking ? 'scale(1.01)' : 'scale(1)',
        transition: 'all 0.16s ease',
        '@keyframes floatAvatar': {
          '0%': { transform: 'translateY(0px)' },
          '50%': { transform: 'translateY(-4px)' },
          '100%': { transform: 'translateY(0px)' },
        },
      }}
    >
      <Box
        sx={{
          width: '100%',
          height: '100%',
          animation: 'floatAvatar 2.8s ease-in-out infinite',
          transformOrigin: '50% 55%',
        }}
      >
        <svg viewBox="0 0 320 320" width="100%" height="100%" role="img" aria-label="interviewer avatar">
          <ellipse cx="160" cy="308" rx="105" ry="22" fill="rgba(0,0,0,0.1)" />
          <path d="M85 290 L235 290 L260 200 L60 200 Z" fill={suitColor} />
          <path d="M133 200 L187 200 L176 292 L144 292 Z" fill={accentColor} />
          <path d="M146 216 L174 216 L165 260 L155 260 Z" fill={gender === 'female' ? '#8aa0bf' : '#93a9bf'} />
          <circle cx="160" cy="145" r="72" fill="#f5c9a8" />
          <ellipse cx="130" cy="168" rx="12" ry="8" fill="#e6a99b" opacity="0.7" />
          <ellipse cx="190" cy="168" rx="12" ry="8" fill="#e6a99b" opacity="0.7" />
          {gender === 'female' ? (
            <>
              <path d="M88 120 C88 64 126 44 160 44 C208 44 238 84 236 128 L232 170 C228 150 220 136 206 126 C184 110 162 112 142 116 C124 120 104 132 92 152 Z" fill={hairColor} />
              <path d="M88 164 C78 202 94 224 120 228 L114 202 C108 184 106 170 110 156 Z" fill={hairColor} />
              <path d="M232 164 C242 202 226 224 200 228 L206 202 C212 184 214 170 210 156 Z" fill={hairColor} />
            </>
          ) : (
            <>
              <path d="M96 126 C94 78 128 48 168 48 C204 48 230 76 224 120 C206 102 186 96 164 96 C140 96 116 104 96 126 Z" fill={hairColor} />
              <path d="M100 120 C112 98 136 86 164 86 C190 86 210 95 222 114 C215 130 204 142 192 148 C182 132 170 124 160 124 C148 124 136 130 126 144 C114 138 106 130 100 120 Z" fill={hairColor} />
            </>
          )}
          <ellipse cx="138" cy="144" rx="11" ry="12" fill="#ffffff" />
          <ellipse cx="182" cy="144" rx="11" ry="12" fill="#ffffff" />
          <circle cx="138" cy="146" r="6.5" fill="#2a1b14" />
          <circle cx="182" cy="146" r="6.5" fill="#2a1b14" />
          <circle cx="140" cy="143.5" r="1.8" fill="#ffffff" />
          <circle cx="184" cy="143.5" r="1.8" fill="#ffffff" />
          <path d="M125 130 Q138 122 151 130" stroke="#3c251a" strokeWidth="3" fill="none" strokeLinecap="round" />
          <path d="M169 130 Q182 122 195 130" stroke="#3c251a" strokeWidth="3" fill="none" strokeLinecap="round" />
          <path d="M160 152 C156 160 156 165 160 169" stroke="#cf9d83" strokeWidth="3" fill="none" strokeLinecap="round" />
          <ellipse cx="160" cy="184" rx="20" ry={mouthOpen} fill="#b94848" />
          <ellipse cx="160" cy={185 + Math.floor(mouthOpen / 5)} rx="12" ry={Math.max(2, mouthOpen - 5)} fill="#f17f7f" />
        </svg>
      </Box>
    </Box>
  )
}
