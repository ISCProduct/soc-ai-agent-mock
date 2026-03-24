import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const userId = searchParams.get('user_id')
  if (!userId) return NextResponse.json({ error: 'user_id is required' }, { status: 400 })

  const res = await fetch(`${BACKEND_URL}/api/schedule/export/ics?user_id=${userId}`)
  if (!res.ok) {
    return NextResponse.json({ error: 'Export failed' }, { status: res.status })
  }
  const ics = await res.text()
  return new NextResponse(ics, {
    status: 200,
    headers: {
      'Content-Type': 'text/calendar; charset=utf-8',
      'Content-Disposition': 'attachment; filename="schedule.ics"',
    },
  })
}
