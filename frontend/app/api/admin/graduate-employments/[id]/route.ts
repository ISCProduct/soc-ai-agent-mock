import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(_req: NextRequest, { params }: { params: { id: string } }) {
  const response = await fetch(`${BACKEND_URL}/api/admin/graduate-employments/${params.id}`)
  const raw = await response.text()
  let data: any = {}
  if (raw) {
    try { data = JSON.parse(raw) } catch { data = { error: raw } }
  }
  return NextResponse.json(data, { status: response.status })
}

export async function PUT(request: NextRequest, { params }: { params: { id: string } }) {
  const body = await request.text()
  const response = await fetch(`${BACKEND_URL}/api/admin/graduate-employments/${params.id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'X-Admin-Email': request.headers.get('x-admin-email') || '',
    },
    body,
  })
  const raw = await response.text()
  let data: any = {}
  if (raw) {
    try { data = JSON.parse(raw) } catch { data = response.ok ? { message: raw } : { error: raw } }
  }
  return NextResponse.json(data, { status: response.status })
}
