const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:80'

export type InterviewSession = {
  id: number
  user_id: number
  status: string
  started_at?: string
  ended_at?: string
  estimated_cost_usd: number
  template_version: string
  created_at: string
  updated_at: string
}

export type InterviewUtterance = {
  id: number
  session_id: number
  role: 'user' | 'ai'
  text: string
  created_at: string
}

export type InterviewReport = {
  session_id: number
  summary_text: string
  scores_json: string
  evidence_json: string
  created_at: string
  updated_at: string
}

export type InterviewDetail = {
  session: InterviewSession
  utterances: InterviewUtterance[]
  report?: InterviewReport
}

export const interviewApi = {
  async createSession(userId: number): Promise<InterviewSession> {
    const res = await fetch(`${BACKEND_URL}/api/interviews`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },

  async startSession(sessionId: number, userId: number): Promise<InterviewSession> {
    const res = await fetch(`${BACKEND_URL}/api/interviews/${sessionId}/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },

  async finishSession(sessionId: number, userId: number): Promise<InterviewSession> {
    const res = await fetch(`${BACKEND_URL}/api/interviews/${sessionId}/finish`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },

  async saveUtterance(sessionId: number, userId: number, role: 'user' | 'ai', text: string): Promise<void> {
    const res = await fetch(`${BACKEND_URL}/api/interviews/${sessionId}/utterances`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, role, text }),
    })
    if (!res.ok) throw new Error(await res.text())
  },

  async getDetail(sessionId: number, userId: number): Promise<InterviewDetail> {
    const res = await fetch(`${BACKEND_URL}/api/interviews/${sessionId}?user_id=${userId}`)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },

  async listSessions(userId: number, page = 1, limit = 20): Promise<{ sessions: InterviewSession[]; total: number }> {
    const res = await fetch(`${BACKEND_URL}/api/interviews?user_id=${userId}&page=${page}&limit=${limit}`)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },

  async createRealtimeToken(userId: number, interviewId: number): Promise<string> {
    const res = await fetch(`${BACKEND_URL}/api/realtime/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, interview_id: interviewId }),
    })
    if (!res.ok) throw new Error(await res.text())
    const data = await res.json()
    return data.client_secret
  },

  async sendReportEmail(sessionId: number, userId: number): Promise<{ message: string }> {
    const res = await fetch(`${BACKEND_URL}/api/interviews/${sessionId}/send-report`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId }),
    })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
}

export const interviewLimits = {
  maxMinutes: Number(process.env.NEXT_PUBLIC_INTERVIEW_MAX_MINUTES || 10),
  maxCostUSD: Number(process.env.NEXT_PUBLIC_INTERVIEW_MAX_COST_USD || 1.8),
  costPerMinuteUSD: Number(process.env.NEXT_PUBLIC_INTERVIEW_COST_PER_MIN_USD || 0.18),
}
