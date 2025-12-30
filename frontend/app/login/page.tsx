'use client'

import { LoginPage } from '@/components/login-page'
import { authService, AuthResponse } from '@/lib/auth'
import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

export default function Login() {
  const router = useRouter()

  useEffect(() => {
    const storedUser = authService.getStoredUser()
    if (storedUser) {
      if (storedUser.target_level !== '新卒' && storedUser.target_level !== '中途') {
        router.replace('/onboarding')
        return
      }
      router.replace('/')
    }
  }, [router])

  const handleAuthSuccess = (authResponse: AuthResponse) => {
    router.push('/')
  }

  return <LoginPage onAuthSuccess={handleAuthSuccess} />
}
