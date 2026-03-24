import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(request: NextRequest) {
  const admin = request.headers.get('x-admin-email') || ''
  const { searchParams } = new URL(request.url)
  const days = searchParams.get('days') || '30'
  const res = await fetch(`${BACKEND_URL}/api/admin/costs/daily?days=${days}`, {
    headers: { 'X-Admin-Email': admin },
  })
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
