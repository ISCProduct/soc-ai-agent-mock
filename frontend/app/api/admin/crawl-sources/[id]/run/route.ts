import { NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function POST(request: Request, { params }: { params: { id: string } }) {
  const response = await fetch(`${BACKEND_URL}/api/admin/crawl-sources/${params.id}/run`, {
    method: 'POST',
    headers: {
      'X-Admin-Email': request.headers.get('x-admin-email') || '',
    },
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
