import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  const url = `${BACKEND_URL}/api/admin/interviews/${params.id}/videos`
  const response = await fetch(url, {
    headers: { 'X-Admin-Email': request.headers.get('x-admin-email') || '' },
  })
  const data = await response.json().catch(() => ({}))
  return NextResponse.json(data, { status: response.status })
}
