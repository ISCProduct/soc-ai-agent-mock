const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:80'

function fixMojibake(s: string): string {
  // Detect common mojibake patterns (Ã, å followed by other Latin-1 chars)
  return /[Ãå][^\s]/.test(s) ? (() => {
    try {
      // Interpret the current UTF-16 code units as Latin-1 bytes and decode as UTF-8
      const bytes = new Uint8Array([...s].map(c => c.charCodeAt(0)))
      return new TextDecoder('utf-8').decode(bytes)
    } catch {
      try {
        // Fallback legacy method
        // @ts-ignore
        return decodeURIComponent(escape(s))
      } catch {
        return s
      }
    }
  })() : s
}

export interface User {
  user_id: number
  email: string
  name: string
  is_guest: boolean
  target_level?: string
  school_name?: string
  is_admin?: boolean
  certifications_acquired?: string
  certifications_in_progress?: string
  oauth_provider?: string
  avatar_url?: string
}

export interface AuthResponse {
  user_id: number | string
  email: string
  name: string
  is_guest: boolean
  target_level?: string
  school_name?: string
  is_admin?: boolean
  certifications_acquired?: string
  certifications_in_progress?: string
  oauth_provider?: string
  avatar_url?: string
  token?: string
}

export const authService = {
  async login(email: string, password: string): Promise<AuthResponse> {
    const res = await fetch(`${BACKEND_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const error = await res.text()
      throw new Error(error || 'Login failed')
    }
    return res.json()
  },

  async register(
    email: string,
    password: string,
    name: string,
    targetLevel: string,
    certificationsAcquired: string,
    certificationsInProgress: string,
  ): Promise<AuthResponse> {
    const res = await fetch(`${BACKEND_URL}/api/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email,
        password,
        name,
        target_level: targetLevel,
        certifications_acquired: certificationsAcquired,
        certifications_in_progress: certificationsInProgress,
      }),
    })
    if (!res.ok) {
      const error = await res.text()
      throw new Error(error || 'Registration failed')
    }
    return res.json()
  },

  async createGuest(): Promise<AuthResponse> {
    const res = await fetch(`${BACKEND_URL}/api/auth/guest`, {
      method: 'POST',
    })
    if (!res.ok) throw new Error('Failed to create guest user')
    return res.json()
  },

  async getUser(userId: number): Promise<User> {
    const res = await fetch(`${BACKEND_URL}/api/auth/user?user_id=${userId}`)
    if (!res.ok) throw new Error('Failed to get user')
    return res.json()
  },

  async updateProfile(
    userId: number,
    name: string,
    targetLevel: string,
    schoolName: string,
    certificationsAcquired: string,
    certificationsInProgress: string,
  ): Promise<AuthResponse> {
    const res = await fetch(`${BACKEND_URL}/api/auth/profile`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: userId,
        name,
        target_level: targetLevel,
        school_name: schoolName,
        certifications_acquired: certificationsAcquired,
        certifications_in_progress: certificationsInProgress,
      }),
    })
    if (!res.ok) {
      const error = await res.text()
      throw new Error(error || 'Failed to update profile')
    }
    return res.json()
  },

  async getGoogleAuthUrl(): Promise<{ auth_url: string; state: string }> {
    const res = await fetch(`${BACKEND_URL}/api/auth/google`)
    if (!res.ok) throw new Error('Failed to get Google auth URL')
    return res.json()
  },

  async getGithubAuthUrl(): Promise<{ auth_url: string; state: string }> {
    const res = await fetch(`${BACKEND_URL}/api/auth/github`)
    if (!res.ok) throw new Error('Failed to get GitHub auth URL')
    return res.json()
  },

  saveAuth(authResponse: AuthResponse) {
    const user: User = {
      user_id: typeof authResponse.user_id === 'string' ? Number(authResponse.user_id) : authResponse.user_id,
      email: authResponse.email,
      name: fixMojibake(authResponse.name),
      is_guest: authResponse.is_guest,
      target_level: authResponse.target_level,
      school_name: authResponse.school_name,
      is_admin: authResponse.is_admin,
      certifications_acquired: authResponse.certifications_acquired,
      certifications_in_progress: authResponse.certifications_in_progress,
      oauth_provider: authResponse.oauth_provider,
      avatar_url: authResponse.avatar_url,
    }
    localStorage.setItem('user', JSON.stringify(user))
    if (authResponse.token) {
      localStorage.setItem('token', authResponse.token)
    }
  },

  getStoredUser(): User | null {
    const user = localStorage.getItem('user')
    if (!user) return null
    try {
      const parsed: User = JSON.parse(user)
      // Repair potential mojibake stored earlier
      return { ...parsed, name: fixMojibake(parsed.name) }
    } catch {
      return null
    }
  },

  getStoredToken(): string | null {
    return localStorage.getItem('token')
  },

  logout() {
    // ユーザー情報とトークンを削除
    localStorage.removeItem('user')
    localStorage.removeItem('token')
    
    // チャットキャッシュを削除
    const sessionId = localStorage.getItem('chat_session_id')
    if (sessionId) {
      localStorage.removeItem(`chat_cache_${sessionId}`)
    }
    localStorage.removeItem('chat_session_id')
  },
}
