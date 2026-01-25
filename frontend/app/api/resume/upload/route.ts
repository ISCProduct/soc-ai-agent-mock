import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function POST(request: NextRequest) {
  try {
    const formData = await request.formData()
    const response = await fetch(`${BACKEND_URL}/api/resume/upload`, {
      method: 'POST',
      body: formData,
    })

    const raw = await response.text()
    let data: any = {}
    if (raw) {
      try {
        data = JSON.parse(raw)
      } catch {
        data = response.ok ? { message: raw } : { error: raw }
      }
    }
    return NextResponse.json(data, { status: response.status })
  } catch (error) {
    console.error('Resume upload proxy error:', error)
    return NextResponse.json({ error: 'Failed to connect to backend' }, { status: 500 })
  }
}
