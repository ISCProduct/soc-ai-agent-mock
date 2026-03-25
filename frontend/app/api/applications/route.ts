import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    const response = await fetch(`${BACKEND_URL}/api/applications`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    const text = await response.text()
    if (!response.ok) {
      return NextResponse.json({ error: text || 'Failed to apply' }, { status: response.status })
    }
    return NextResponse.json(JSON.parse(text), { status: 201 })
  } catch (error) {
    console.error('[applications] POST error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const userId = searchParams.get('user_id')
    if (!userId) {
      return NextResponse.json({ error: 'user_id is required' }, { status: 400 })
    }
    const response = await fetch(`${BACKEND_URL}/api/applications?user_id=${userId}`)
    const text = await response.text()
    if (!response.ok) {
      return NextResponse.json({ error: text }, { status: response.status })
    }
    return NextResponse.json(JSON.parse(text))
  } catch (error) {
    console.error('[applications] GET error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}
