import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params
  const admin = request.headers.get('x-admin-email') || ''

  const res = await fetch(`${BACKEND_URL}/api/admin/dashboard/users/${id}/sessions`, {
    headers: { 'X-Admin-Email': admin },
  })
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
