import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET() {
  const response = await fetch(`${BACKEND_URL}/api/admin/crawl-sources`)
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
}

export async function POST(request: NextRequest) {
  const body = await request.text()
  const response = await fetch(`${BACKEND_URL}/api/admin/crawl-sources`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Admin-Email': request.headers.get('x-admin-email') || '',
    },
    body,
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
}
