const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:80'

export interface User {
  user_id: number
  email: string
  name: string
  is_guest: boolean
  oauth_provider?: string
  avatar_url?: string
}

export interface AuthResponse extends User {
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

  async register(email: string, password: string, name: string): Promise<AuthResponse> {
    const res = await fetch(`${BACKEND_URL}/api/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, name }),
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
      user_id: authResponse.user_id,
      email: authResponse.email,
      name: authResponse.name,
      is_guest: authResponse.is_guest,
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
    return user ? JSON.parse(user) : null
  },

  getStoredToken(): string | null {
    return localStorage.getItem('token')
  },

  logout() {
    localStorage.removeItem('user')
    localStorage.removeItem('token')
  },
}
