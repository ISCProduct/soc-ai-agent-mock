'use client'

import { useEffect, useState } from 'react'
import { Box, Typography, Chip, CircularProgress, Paper, Alert, Button, Divider, Accordion, AccordionSummary, AccordionDetails } from '@mui/material'
import GitHubIcon from '@mui/icons-material/GitHub'
import RefreshIcon from '@mui/icons-material/Refresh'
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:80'

interface SkillScore {
  ID: number
  UserID: number
  Category: string
  Score: number
}

interface LanguageStat {
  Language: string
  Percentage: number
}

interface GitHubProfile {
  GitHubLogin: string
  TotalContributions: number
  PublicRepos: number
  Followers: number
}

interface RepoSummary {
  ID: number
  FullName: string
  SummaryText: string
  TechReason: string
  Challenge: string
  Achievement: string
}

// カテゴリ別の表示色
const CATEGORY_COLORS: Record<string, string> = {
  Frontend:       '#4FC3F7',
  Backend:        '#81C784',
  Infrastructure: '#FFB74D',
  Database:       '#F48FB1',
  Other:          '#CE93D8',
}

const CATEGORY_LABELS: Record<string, string> = {
  Frontend:       'フロントエンド',
  Backend:        'バックエンド',
  Infrastructure: 'インフラ',
  Database:       'DB',
  Other:          'その他',
}

const AXES_ORDER = ['Frontend', 'Backend', 'Infrastructure', 'Database', 'Other']

// --- SVG レーダーチャート ---

interface RadarChartProps {
  scores: SkillScore[]
  size?: number
}

function RadarChart({ scores, size = 240 }: RadarChartProps) {
  const center = size / 2
  const radius = size * 0.38
  const n = AXES_ORDER.length
  const scoreMap = Object.fromEntries(scores.map(s => [s.Category, s.Score]))

  const angleOf = (i: number) => (Math.PI * 2 * i) / n - Math.PI / 2

  const pointOnAxis = (i: number, ratio: number) => {
    const a = angleOf(i)
    return {
      x: center + radius * ratio * Math.cos(a),
      y: center + radius * ratio * Math.sin(a),
    }
  }

  // グリッド（25, 50, 75, 100）
  const gridLevels = [0.25, 0.5, 0.75, 1.0]
  const gridPolygons = gridLevels.map(level => {
    const pts = AXES_ORDER.map((_, i) => {
      const p = pointOnAxis(i, level)
      return `${p.x},${p.y}`
    }).join(' ')
    return pts
  })

  // データポリゴン
  const dataPoints = AXES_ORDER.map((cat, i) => {
    const score = scoreMap[cat] ?? 0
    const p = pointOnAxis(i, score / 100)
    return `${p.x},${p.y}`
  }).join(' ')

  // 軸ラベル
  const labelOffset = 22
  const labels = AXES_ORDER.map((cat, i) => {
    const a = angleOf(i)
    return {
      x: center + (radius + labelOffset) * Math.cos(a),
      y: center + (radius + labelOffset) * Math.sin(a),
      label: CATEGORY_LABELS[cat] ?? cat,
      color: CATEGORY_COLORS[cat] ?? '#888',
    }
  })

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      {/* グリッド */}
      {gridPolygons.map((pts, i) => (
        <polygon
          key={i}
          points={pts}
          fill="none"
          stroke="#334155"
          strokeWidth="1"
          strokeDasharray={i < gridPolygons.length - 1 ? '3,3' : undefined}
        />
      ))}
      {/* 軸線 */}
      {AXES_ORDER.map((_, i) => {
        const outer = pointOnAxis(i, 1)
        return (
          <line
            key={i}
            x1={center}
            y1={center}
            x2={outer.x}
            y2={outer.y}
            stroke="#334155"
            strokeWidth="1"
          />
        )
      })}
      {/* データ領域 */}
      <polygon
        points={dataPoints}
        fill="rgba(79,195,247,0.25)"
        stroke="#4FC3F7"
        strokeWidth="2"
      />
      {/* 頂点の丸 */}
      {AXES_ORDER.map((cat, i) => {
        const score = scoreMap[cat] ?? 0
        const p = pointOnAxis(i, score / 100)
        return (
          <circle
            key={i}
            cx={p.x}
            cy={p.y}
            r="4"
            fill={CATEGORY_COLORS[cat] ?? '#4FC3F7'}
          />
        )
      })}
      {/* ラベル */}
      {labels.map((l, i) => (
        <text
          key={i}
          x={l.x}
          y={l.y}
          textAnchor="middle"
          dominantBaseline="middle"
          fontSize="10"
          fill={l.color}
          fontWeight="600"
        >
          {l.label}
        </text>
      ))}
    </svg>
  )
}

// --- メインコンポーネント ---

export default function GitHubSkills({ userId }: { userId: number }) {
  const [scores, setScores] = useState<SkillScore[]>([])
  const [profile, setProfile] = useState<GitHubProfile | null>(null)
  const [langStats, setLangStats] = useState<LanguageStat[]>([])
  const [summaries, setSummaries] = useState<RepoSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [notLinked, setNotLinked] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [connecting, setConnecting] = useState(false)
  const [summarizingRepo, setSummarizingRepo] = useState<string | null>(null)

  useEffect(() => {
    fetchAll()
  }, [userId])

  const fetchAll = async () => {
    setLoading(true)
    setError(null)
    setNotLinked(false)
    try {
      const [skillsRes, profileRes, summariesRes] = await Promise.all([
        fetch(`${BACKEND_URL}/api/github/skills?user_id=${userId}`),
        fetch(`${BACKEND_URL}/api/github/profile?user_id=${userId}`),
        fetch(`${BACKEND_URL}/api/github/repo/summaries?user_id=${userId}`),
      ])

      if (skillsRes.ok) {
        const data = await skillsRes.json()
        setScores(Array.isArray(data) ? data : [])
      }

      if (profileRes.ok) {
        const data = await profileRes.json()
        setProfile(data.profile ?? null)
        setLangStats(
          Array.isArray(data.language_stats)
            ? data.language_stats.sort((a: LanguageStat, b: LanguageStat) => b.Percentage - a.Percentage).slice(0, 8)
            : []
        )
      } else if (profileRes.status === 404) {
        setNotLinked(true)
      }

      if (summariesRes.ok) {
        const data = await summariesRes.json()
        setSummaries(Array.isArray(data) ? data : [])
      }
    } catch {
      setError('データの取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  const handleSummarizeRepo = async (fullName: string) => {
    setSummarizingRepo(fullName)
    try {
      const res = await fetch(`${BACKEND_URL}/api/github/repo/summarize?user_id=${userId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ full_name: fullName }),
      })
      if (!res.ok) throw new Error()
      const newSummary: RepoSummary = await res.json()
      setSummaries(prev => {
        const filtered = prev.filter(s => s.FullName !== fullName)
        return [newSummary, ...filtered]
      })
    } catch {
      setError(`${fullName} の要約生成に失敗しました`)
    } finally {
      setSummarizingRepo(null)
    }
  }

  const handleConnect = async () => {
    setConnecting(true)
    try {
      const res = await fetch(`${BACKEND_URL}/api/auth/github`)
      if (!res.ok) throw new Error()
      const { auth_url } = await res.json()
      window.location.href = auth_url
    } catch {
      setError('GitHub連携URLの取得に失敗しました')
      setConnecting(false)
    }
  }

  const handleSync = async () => {
    setSyncing(true)
    try {
      await fetch(`${BACKEND_URL}/api/github/sync?user_id=${userId}`, { method: 'POST' })
      // 非同期syncなので少し待ってから再取得
      setTimeout(() => {
        fetchAll().finally(() => setSyncing(false))
      }, 3000)
    } catch {
      setSyncing(false)
    }
  }

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress size={32} />
      </Box>
    )
  }

  if (notLinked) {
    return (
      <Paper sx={{ p: 3, mt: 3, bgcolor: '#0f172a', borderRadius: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
          <GitHubIcon sx={{ color: '#94a3b8' }} />
          <Typography variant="h6" sx={{ fontWeight: 700, color: '#f1f5f9' }}>
            GitHub スキル分析
          </Typography>
        </Box>
        <Alert
          severity="info"
          sx={{ mb: 2, bgcolor: '#1e293b', color: '#94a3b8', '& .MuiAlert-icon': { color: '#4FC3F7' } }}
        >
          GitHubアカウントが連携されていません。GitHubでログインするとスキル分析が自動的に生成されます。
        </Alert>
        <Button
          variant="contained"
          startIcon={connecting ? <CircularProgress size={16} /> : <GitHubIcon />}
          onClick={handleConnect}
          disabled={connecting}
          sx={{ bgcolor: '#24292e', '&:hover': { bgcolor: '#444d56' } }}
        >
          {connecting ? '連携中...' : 'GitHubと連携する'}
        </Button>
      </Paper>
    )
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 2 }}>
        {error}
      </Alert>
    )
  }

  const hasScores = scores.some(s => s.Score > 0)

  return (
    <Paper sx={{ p: 3, mt: 3, bgcolor: '#0f172a', color: '#e2e8f0', borderRadius: 2 }}>
      {/* ヘッダー */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <GitHubIcon sx={{ color: '#94a3b8' }} />
          <Typography variant="h6" sx={{ fontWeight: 700, color: '#f1f5f9' }}>
            GitHub スキル分析
          </Typography>
          {profile && (
            <Typography variant="body2" sx={{ color: '#64748b', ml: 1 }}>
              @{profile.GitHubLogin}
            </Typography>
          )}
        </Box>
        <Button
          size="small"
          startIcon={syncing ? <CircularProgress size={14} /> : <RefreshIcon />}
          onClick={handleSync}
          disabled={syncing}
          sx={{ color: '#64748b', fontSize: '0.75rem' }}
        >
          {syncing ? '同期中...' : '同期'}
        </Button>
      </Box>

      {/* プロフィール統計 */}
      {profile && (
        <Box sx={{ display: 'flex', gap: 3, mb: 3 }}>
          {[
            { label: 'コントリビューション', value: profile.TotalContributions },
            { label: 'リポジトリ', value: profile.PublicRepos },
            { label: 'フォロワー', value: profile.Followers },
          ].map(item => (
            <Box key={item.label} sx={{ textAlign: 'center' }}>
              <Typography variant="h6" sx={{ color: '#4FC3F7', fontWeight: 700 }}>
                {item.value}
              </Typography>
              <Typography variant="caption" sx={{ color: '#64748b' }}>
                {item.label}
              </Typography>
            </Box>
          ))}
        </Box>
      )}

      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 3, alignItems: 'flex-start' }}>
        {/* レーダーチャート */}
        {hasScores && (
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
            <RadarChart scores={scores} size={240} />
            {/* スコア凡例 */}
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1, justifyContent: 'center', maxWidth: 240 }}>
              {AXES_ORDER.map(cat => {
                const s = scores.find(x => x.Category === cat)
                if (!s) return null
                return (
                  <Chip
                    key={cat}
                    label={`${CATEGORY_LABELS[cat]} ${s.Score.toFixed(1)}`}
                    size="small"
                    sx={{
                      bgcolor: CATEGORY_COLORS[cat] + '33',
                      color: CATEGORY_COLORS[cat],
                      border: `1px solid ${CATEGORY_COLORS[cat]}55`,
                      fontSize: '0.7rem',
                    }}
                  />
                )
              })}
            </Box>
          </Box>
        )}

        {/* 言語スキルタグ */}
        {langStats.length > 0 && (
          <Box sx={{ flex: 1, minWidth: 180 }}>
            <Typography variant="subtitle2" sx={{ color: '#94a3b8', mb: 1.5 }}>
              使用言語
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              {langStats.map(stat => (
                <Chip
                  key={stat.Language}
                  label={`${stat.Language} ${stat.Percentage.toFixed(1)}%`}
                  size="small"
                  sx={{
                    bgcolor: '#1e293b',
                    color: '#94a3b8',
                    border: '1px solid #334155',
                    fontSize: '0.75rem',
                  }}
                />
              ))}
            </Box>
          </Box>
        )}
      </Box>

      {!hasScores && !loading && (
        <Typography variant="body2" sx={{ color: '#64748b', textAlign: 'center', py: 2 }}>
          スキルデータがありません。「同期」ボタンでGitHubデータを取得してください。
        </Typography>
      )}

      {/* リポジトリAI要約セクション */}
      {summaries.length > 0 && (
        <>
          <Divider sx={{ borderColor: '#1e293b', my: 3 }} />
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
            <AutoAwesomeIcon sx={{ color: '#818cf8', fontSize: 18 }} />
            <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#f1f5f9' }}>
              リポジトリAI要約
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {summaries.map(s => (
              <Accordion
                key={s.ID}
                sx={{ bgcolor: '#1e293b', color: '#e2e8f0', borderRadius: 1, '&:before': { display: 'none' } }}
                disableGutters
              >
                <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#64748b' }} />}>
                  <Typography variant="body2" sx={{ fontWeight: 600, color: '#94a3b8' }}>
                    {s.FullName}
                  </Typography>
                </AccordionSummary>
                <AccordionDetails sx={{ pt: 0 }}>
                  <Typography variant="body2" sx={{ color: '#cbd5e1', mb: 1.5 }}>{s.SummaryText}</Typography>
                  {[
                    { label: '技術選定の理由', value: s.TechReason, color: '#4FC3F7' },
                    { label: '解決した課題', value: s.Challenge, color: '#81C784' },
                    { label: '成果', value: s.Achievement, color: '#FFB74D' },
                  ].map(item => (
                    <Box key={item.label} sx={{ mb: 1 }}>
                      <Typography variant="caption" sx={{ color: item.color, fontWeight: 700 }}>
                        {item.label}
                      </Typography>
                      <Typography variant="body2" sx={{ color: '#94a3b8' }}>{item.value}</Typography>
                    </Box>
                  ))}
                  <Button
                    size="small"
                    startIcon={summarizingRepo === s.FullName ? <CircularProgress size={12} /> : <AutoAwesomeIcon />}
                    onClick={() => handleSummarizeRepo(s.FullName)}
                    disabled={summarizingRepo !== null}
                    sx={{ mt: 1, color: '#818cf8', fontSize: '0.7rem' }}
                  >
                    再生成
                  </Button>
                </AccordionDetails>
              </Accordion>
            ))}
          </Box>
        </>
      )}

      {/* リポジトリ要約生成ボタン（未生成時） */}
      {profile && summaries.length === 0 && !loading && (
        <>
          <Divider sx={{ borderColor: '#1e293b', my: 3 }} />
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            <AutoAwesomeIcon sx={{ color: '#818cf8', fontSize: 18 }} />
            <Typography variant="subtitle2" sx={{ color: '#94a3b8' }}>
              リポジトリAI要約
            </Typography>
          </Box>
          <Typography variant="body2" sx={{ color: '#64748b', mb: 2 }}>
            リポジトリのREADMEをAIが解析し、技術的強みを要約します。
          </Typography>
          <Typography variant="caption" sx={{ color: '#475569' }}>
            個別リポジトリから要約を生成するには、プロフィールページのリポジトリ一覧から選択してください。
          </Typography>
        </>
      )}
    </Paper>
  )
}
