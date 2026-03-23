import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const userId = searchParams.get('user_id')
  if (!userId) return NextResponse.json({ error: 'user_id is required' }, { status: 400 })

  const res = await fetch(`${BACKEND_URL}/api/schedule?user_id=${userId}`)
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}

export async function POST(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const userId = searchParams.get('user_id')
  if (!userId) return NextResponse.json({ error: 'user_id is required' }, { status: 400 })

  const body = await request.text()
  const res = await fetch(`${BACKEND_URL}/api/schedule?user_id=${userId}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  })
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
