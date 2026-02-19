'use client'

import { useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import {
  Box,
  Button,
  Chip,
  Divider,
  LinearProgress,
  Paper,
  Stack,
  Typography,
} from '@mui/material'
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
            if (!cancelled) {
              setInterviewCompany(detail)
            }
            return
          }
        }

        const fallbackRes = await fetch('/api/companies?limit=1&offset=0', { cache: 'no-store' })
        if (!fallbackRes.ok) return
        const fallback = await fallbackRes.json()
        const first = Array.isArray(fallback?.companies) ? fallback.companies[0] : null
        if (!cancelled && first) {
          setInterviewCompany(first)
        }
      } catch {
        // ignore
      }
    }

    loadCompany()
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    return () => {
      cleanupConnection()
    }
  }, [])

  const formatSeconds = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}:${s.toString().padStart(2, '0')}`
  }

  const parseJsonSafe = (value?: string) => {
    if (!value) return null
    try {
      return JSON.parse(value)
    } catch {
      return null
    }
  }

  const cleanupConnection = () => {
    if (timerRef.current) {
      clearInterval(timerRef.current)
      timerRef.current = null
    }
    if (pollRef.current) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
    if (aiAnimationRef.current) {
      cancelAnimationFrame(aiAnimationRef.current)
      aiAnimationRef.current = null
    }
    if (aiAudioCtxRef.current) {
      aiAudioCtxRef.current.close().catch(() => undefined)
      aiAudioCtxRef.current = null
      aiAnalyserRef.current = null
    }
    if (dcRef.current) {
      dcRef.current.close()
      dcRef.current = null
    }
    if (pcRef.current) {
      pcRef.current.close()
      pcRef.current = null
    }
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
      setErrorMessage(error?.message || '接続に失敗しました')
      cleanupConnection()
    }
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
    if (pollRef.current) {
      clearInterval(pollRef.current)
    }
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

    const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
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
        if (event.delta) {
          setPartialUser(prev => prev + event.delta)
        }
        break
      }
      case 'conversation.item.input_audio_transcription.completed': {
        const text = (event.transcript || event.text || partialUser).trim()
        if (text) {
          setUtterances(prev => [...prev, { role: 'user', text }])
          setPartialUser('')
          try {
            await interviewApi.saveUtterance(sessionId, user.user_id, 'user', text)
          } catch {
            // ignore
          }
        }
        break
      }
      case 'response.audio_transcript.delta': {
        if (event.delta) {
          setPartialAi(prev => prev + event.delta)
        }
        break
      }
      case 'response.audio_transcript.done': {
        const text = (event.transcript || partialAi).trim()
        if (text) {
          setUtterances(prev => [...prev, { role: 'ai', text }])
          setPartialAi('')
          try {
            await interviewApi.saveUtterance(sessionId, user.user_id, 'ai', text)
          } catch {
            // ignore
          }
        }
        break
      }
      default:
        break
    }
  }

  if (loading || !user) {
    return null
  }

  const isActive = status === 'connecting' || status === 'connected'
  const progress = Math.min(100, Math.round(((interviewLimits.maxMinutes * 60 - remainingSeconds) / (interviewLimits.maxMinutes * 60)) * 100))
  const scores = report ? parseJsonSafe(report.scores_json) : null
  const evidence = report ? parseJsonSafe(report.evidence_json) : null
  const isFemaleAvatar = avatarGender === 'female'
  const statusLabel =
    status === 'connected'
      ? '接続中'
      : status === 'connecting'
      ? '接続中...'
      : status === 'error'
      ? 'エラー'
      : status === 'finished'
      ? '終了'
      : '待機中'
  const aiMessages = utterances.filter((u) => u.role === 'ai').slice(-2)
  const fallbackAiMessages = [
    'はじめまして。弊社にご関心いただきありがとうございます。',
    'まずは相互理解のため、簡単に自己紹介をお願いします。',
  ]
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

  return (
    <Box
      component="main"
      sx={{
        minHeight: '100vh',
        p: { xs: 1.5, md: 3 },
        background: 'linear-gradient(165deg, #ddded2 0%, #d2d1c2 45%, #c9c7b2 100%)',
      }}
    >
      <Box
        sx={{
          position: 'relative',
          maxWidth: 1400,
          mx: 'auto',
          borderRadius: { xs: 4, md: 6 },
          overflow: 'hidden',
          minHeight: { xs: 'calc(100vh - 24px)', md: 'calc(100vh - 48px)' },
          background: 'radial-gradient(circle at 72% 28%, #ececdd 0%, #dddcca 42%, #c8c5ad 100%)',
          boxShadow: '0 26px 60px rgba(71, 82, 56, 0.18)',
          p: { xs: 1.5, md: 3 },
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', lg: '310px 1fr 360px' },
          gap: { xs: 1.5, md: 2.5 },
        }}
      >
        <Paper
          sx={{
            p: 2.2,
            borderRadius: 3.5,
            display: 'flex',
            flexDirection: 'column',
            gap: 1.8,
            border: '1px solid rgba(48, 73, 52, 0.12)',
            background: 'rgba(248, 248, 242, 0.86)',
            backdropFilter: 'blur(6px)',
            minHeight: { xs: 'auto', lg: '100%' },
          }}
        >
          <Box sx={{ p: 1.5, borderRadius: 2.5, bgcolor: '#f5f6ed', border: '1px solid #e2e7d9' }}>
            <Typography sx={{ fontWeight: 700, fontSize: 30, lineHeight: 1 }}>
              {companyName}
            </Typography>
          </Box>

          <Box
            sx={{
              p: 1.6,
              borderRadius: 2.5,
              bgcolor: '#f1f2e8',
              border: '1px solid #e2e7d9',
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
              minHeight: { xs: 260, lg: 470 },
            }}
          >
            <Typography sx={{ fontWeight: 800, fontSize: 28, mb: 1.2 }}>
              セールス募集要項
            </Typography>
            <Box sx={{ pr: 0.5, overflow: 'auto' }}>
              <Typography sx={{ fontSize: 16, color: '#38423a', whiteSpace: 'pre-line', lineHeight: 1.85 }}>
                {recruitingText}
              </Typography>
            </Box>
          </Box>

          <Box sx={{ mt: 'auto', display: 'flex', alignItems: 'center', gap: 1 }}>
            <Chip
              label={statusLabel}
              color={status === 'error' ? 'error' : status === 'connected' ? 'success' : 'default'}
              sx={{ fontWeight: 600 }}
            />
            <Typography variant="caption" color="text.secondary">
              推定コスト ${estimatedCost.toFixed(2)}
            </Typography>
          </Box>
        </Paper>

        <Paper
          sx={{
            position: 'relative',
            borderRadius: 3.5,
            minHeight: { xs: 420, lg: '100%' },
            border: '1px solid rgba(48, 73, 52, 0.12)',
            background: 'transparent',
            boxShadow: 'none',
          }}
        >
          <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Box
              sx={{
                width: { xs: 250, md: 460, lg: 560 },
                height: { xs: 250, md: 460, lg: 560 },
                borderRadius: '50%',
                background: 'radial-gradient(circle at 48% 36%, rgba(255,255,255,0.94) 0%, rgba(239,236,221,0.85) 45%, rgba(206,196,165,0.4) 100%)',
                boxShadow: aiSpeaking
                  ? '0 0 0 20px rgba(255,255,255,0.22), 0 20px 40px rgba(93, 94, 58, 0.22)'
                  : '0 0 0 10px rgba(255,255,255,0.12), 0 20px 40px rgba(93, 94, 58, 0.18)',
                transition: 'box-shadow 0.2s ease, transform 0.2s ease',
                transform: aiSpeaking ? 'scale(1.015)' : 'scale(1)',
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

          <Box sx={{ position: 'absolute', left: 18, top: 16, display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 10, height: 10, borderRadius: '50%', background: aiSpeaking ? '#ef4444' : '#22c55e' }} />
            <Typography variant="caption" sx={{ color: '#3d4f42', fontWeight: 600 }}>
              AI面接中
            </Typography>
          </Box>

          <Box sx={{ position: 'absolute', left: 18, bottom: 18 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#2f4234' }}>
              面接官AI（{isFemaleAvatar ? '女性' : '男性'}）
            </Typography>
            <Typography variant="caption" sx={{ color: '#58695d' }}>
              {aiSpeaking ? '話しています' : '待機中'}
            </Typography>
          </Box>

          <Box
            sx={{
              position: 'absolute',
              left: 16,
              bottom: { xs: 74, lg: 16 },
              width: { xs: 140, md: 190 },
              height: { xs: 104, md: 136 },
              borderRadius: 3,
              background: '#efeee3',
              border: '1px solid rgba(68, 86, 68, 0.22)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#405145',
              overflow: 'hidden',
              boxShadow: '0 10px 18px rgba(55, 62, 47, 0.2)',
            }}
          >
            <Box sx={{ position: 'absolute', inset: 0 }}>
              <video
                ref={videoRef}
                muted
                playsInline
                style={{ width: '100%', height: '100%', objectFit: 'cover' }}
              />
            </Box>
            <Box sx={{ position: 'absolute', bottom: 8, left: 10, bgcolor: 'rgba(33,43,35,0.65)', px: 1, borderRadius: 5 }}>
              <Typography variant="caption" sx={{ color: '#fff' }}>あなた</Typography>
            </Box>
          </Box>
        </Paper>

        <Paper
          sx={{
            p: 2,
            borderRadius: 3.5,
            display: 'flex',
            flexDirection: 'column',
            gap: 1.6,
            background: 'rgba(247, 248, 240, 0.92)',
            border: '1px solid rgba(48, 73, 52, 0.12)',
            minHeight: { xs: 'auto', lg: '100%' },
          }}
        >
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#33443a' }}>
              面接チャット
            </Typography>
            <Chip size="small" label={formatSeconds(remainingSeconds)} sx={{ bgcolor: '#ecefdf' }} />
          </Box>

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.2 }}>
            {(aiMessages.length ? aiMessages : fallbackAiMessages.map((text) => ({ role: 'ai' as const, text }))).map((msg, idx) => (
              <Box
                key={`bubble-${idx}-${msg.text.slice(0, 8)}`}
                sx={{
                  borderRadius: 3,
                  bgcolor: '#ffffff',
                  p: 1.4,
                  border: '1px solid #e0e6d8',
                  color: '#2f3f35',
                }}
              >
                <Typography sx={{ fontSize: 20, lineHeight: 1.45 }}>{msg.text}</Typography>
              </Box>
            ))}
            {partialAi && (
              <Box sx={{ borderRadius: 3, bgcolor: '#ffffff', p: 1.4, border: '1px solid #e0e6d8' }}>
                <Typography sx={{ fontSize: 18, color: '#4d6153' }}>{partialAi}</Typography>
              </Box>
            )}
          </Box>

          <Box sx={{ mt: 'auto', pt: 1 }}>
            <Box
              sx={{
                borderRadius: 3,
                bgcolor: aiSpeaking ? '#5f7260' : '#728272',
                color: '#fff',
                px: 2,
                py: 1.3,
                display: 'flex',
                alignItems: 'center',
                gap: 1,
              }}
            >
              <Typography sx={{ fontWeight: 700, fontSize: 18 }}>●</Typography>
              <Typography sx={{ fontSize: 28, fontWeight: 700 }}>
                {isActive ? 'レコーディング中...' : '待機中'}
              </Typography>
            </Box>
            <LinearProgress
              variant="determinate"
              value={progress}
              sx={{ mt: 1.2, height: 7, borderRadius: 4, bgcolor: '#d8decb', '& .MuiLinearProgress-bar': { bgcolor: '#355a46' } }}
            />
            <Stack direction="row" spacing={1} sx={{ mt: 1.2 }}>
              <Button
                variant="contained"
                onClick={handleStart}
                disabled={isActive}
                sx={{ flex: 1, borderRadius: 8, bgcolor: '#365f4c', py: 1.1, fontWeight: 700 }}
              >
                レコーディング開始
              </Button>
              <Button
                variant="contained"
                onClick={() => handleStop(false)}
                disabled={!isActive}
                sx={{ flex: 1, borderRadius: 8, bgcolor: '#244f40', py: 1.1, fontWeight: 700 }}
              >
                完了する
              </Button>
            </Stack>
            {errorMessage && (
              <Typography variant="body2" color="error" sx={{ mt: 1 }}>
                {errorMessage}
              </Typography>
            )}
            {partialUser && (
              <Typography variant="body2" sx={{ mt: 1, color: '#5a6559' }}>
                あなた（入力中）: {partialUser}
              </Typography>
            )}
          </Box>
        </Paper>
      </Box>

      <Paper
        sx={{
          mt: 2,
          p: 2,
          borderRadius: 3.5,
          background: 'rgba(247, 248, 240, 0.88)',
          border: '1px solid rgba(48, 73, 52, 0.12)',
        }}
      >
        <Typography variant="subtitle1" sx={{ fontWeight: 700, mb: 1 }}>
          面接レポート
        </Typography>
        <Box>
            {reportStatus === 'pending' && (
              <Stack spacing={1}>
                <Typography variant="body2">生成中です。しばらくお待ちください。</Typography>
                <LinearProgress />
              </Stack>
            )}
            {reportStatus === 'ready' && report && (
              <Stack spacing={2}>
                <Box>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    要約
                  </Typography>
                  <Typography variant="body2" sx={{ whiteSpace: 'pre-line' }}>
                    {report.summary_text || '要約がありません'}
                  </Typography>
                </Box>
                <Divider />
                <Box>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    評価
                  </Typography>
                  {scores ? (
                    <Stack spacing={0.5}>
                      {Object.entries(scores).map(([key, value]) => (
                        <Typography key={key} variant="body2">
                          {key}: {String(value)}
                        </Typography>
                      ))}
                    </Stack>
                  ) : (
                    <Typography variant="body2" sx={{ whiteSpace: 'pre-line' }}>
                      {report.scores_json}
                    </Typography>
                  )}
                </Box>
                <Box>
                  <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    根拠
                  </Typography>
                  {evidence ? (
                    <Stack spacing={0.5}>
                      {Object.entries(evidence).map(([key, value]) => (
                        <Typography key={key} variant="body2">
                          {key}: {String(value)}
                        </Typography>
                      ))}
                    </Stack>
                  ) : (
                    <Typography variant="body2" sx={{ whiteSpace: 'pre-line' }}>
                      {report.evidence_json}
                    </Typography>
                  )}
                </Box>
              </Stack>
            )}
            {reportStatus === 'error' && (
              <Typography variant="body2" color="error">
                レポート生成に失敗しました。
              </Typography>
            )}
            {reportStatus === 'idle' && (
              <Typography variant="body2" color="text.secondary">
                面接終了後に表示されます。
              </Typography>
            )}
        </Box>
      </Paper>

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

function InterviewerAvatar({
  imageUrl,
  fallbackGender,
  level,
  speaking,
}: {
  imageUrl: string
  fallbackGender: 'male' | 'female'
  level: number
  speaking: boolean
}) {
  const [useFallback, setUseFallback] = useState(false)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const levelRef = useRef(level)
  const speakingRef = useRef(speaking)

  useEffect(() => {
    levelRef.current = level
    speakingRef.current = speaking
  }, [level, speaking])

  useEffect(() => {
    if (!containerRef.current || useFallback) return

    const container = containerRef.current
    const width = container.clientWidth
    const height = container.clientHeight

    const scene = new THREE.Scene()
    const camera = new THREE.PerspectiveCamera(34, width / height, 0.1, 20)
    camera.position.set(0, 0.2, 4.6)

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
    renderer.setPixelRatio(window.devicePixelRatio)
    renderer.setSize(width, height)
    renderer.outputColorSpace = THREE.SRGBColorSpace
    container.appendChild(renderer.domElement)

    scene.add(new THREE.AmbientLight(0xffffff, 1.05))
    const key = new THREE.DirectionalLight(0xffffff, 0.85)
    key.position.set(2.2, 2.4, 3.2)
    scene.add(key)
    const rim = new THREE.DirectionalLight(0xffffff, 0.45)
    rim.position.set(-2.0, 1.4, -2.0)
    scene.add(rim)

    const group = new THREE.Group()
    scene.add(group)

    const halo = new THREE.Mesh(
      new THREE.CircleGeometry(2.0, 40),
      new THREE.MeshBasicMaterial({ color: 0xf7f4ec, transparent: true, opacity: 0.45 })
    )
    halo.position.set(0, 0.15, -0.35)
    group.add(halo)

    const loader = new THREE.TextureLoader()
    let disposed = false
    let frameId = 0

    const animate = (avatarMesh?: THREE.Mesh) => {
      const start = performance.now()
      const loop = () => {
        const t = (performance.now() - start) / 1000
        const talk = Math.min(1, levelRef.current * 1.2)
        group.rotation.y = Math.sin(t * 0.8) * 0.06 + (talk - 0.2) * 0.03
        group.position.y = Math.sin(t * 1.25) * 0.04
        if (avatarMesh) {
          const pulse = speakingRef.current ? 1 + talk * 0.025 : 1
          avatarMesh.scale.set(1.92 * pulse, 1.92 * pulse, 1)
        }
        renderer.render(scene, camera)
        frameId = requestAnimationFrame(loop)
      }
      loop()
    }

    loader.load(
      imageUrl,
      (texture) => {
        if (disposed) return
        texture.colorSpace = THREE.SRGBColorSpace
        texture.minFilter = THREE.LinearFilter
        texture.magFilter = THREE.LinearFilter

        const avatarMat = new THREE.MeshStandardMaterial({
          map: texture,
          roughness: 0.68,
          metalness: 0.02,
        })
        const avatar = new THREE.Mesh(new THREE.PlaneGeometry(1.92, 1.08), avatarMat)
        avatar.position.set(0, 0.05, 0)
        group.add(avatar)

        const depth1 = new THREE.Mesh(
          new THREE.PlaneGeometry(1.94, 1.1),
          new THREE.MeshBasicMaterial({ color: 0x2f2d2a, transparent: true, opacity: 0.15 })
        )
        depth1.position.set(0.03, -0.01, -0.06)
        group.add(depth1)
        const depth2 = depth1.clone()
        depth2.position.set(0.05, -0.02, -0.12)
        group.add(depth2)

        animate(avatar)
      },
      undefined,
      () => {
        if (!disposed) {
          setUseFallback(true)
        }
      }
    )

    const onResize = () => {
      const w = container.clientWidth
      const h = container.clientHeight
      camera.aspect = w / h
      camera.updateProjectionMatrix()
      renderer.setSize(w, h)
    }
    window.addEventListener('resize', onResize)

    return () => {
      disposed = true
      cancelAnimationFrame(frameId)
      window.removeEventListener('resize', onResize)
      renderer.dispose()
      if (renderer.domElement.parentElement === container) {
        container.removeChild(renderer.domElement)
      }
      scene.traverse((obj) => {
        if (obj instanceof THREE.Mesh) {
          obj.geometry.dispose()
          const m = obj.material
          if (Array.isArray(m)) {
            m.forEach((mm) => mm.dispose())
          } else {
            m.dispose()
          }
        }
      })
    }
  }, [imageUrl, useFallback])

  if (!useFallback) {
    return (
      <Box
        ref={containerRef}
        sx={{
          width: { xs: 214, md: 330, lg: 380 },
          height: { xs: 214, md: 330, lg: 380 },
        }}
      />
    )
  }

  return (
    <InterviewerFallbackAvatar
      gender={fallbackGender}
      level={level}
      speaking={speaking}
    />
  )
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
