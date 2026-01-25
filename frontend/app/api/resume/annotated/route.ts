import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://app:8080'

export const dynamic = 'force-dynamic'

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const documentId = searchParams.get('document_id')
    if (!documentId) {
      return NextResponse.json({ error: 'document_id is required' }, { status: 400 })
    }

    const response = await fetch(`${BACKEND_URL}/api/resume/annotated?document_id=${documentId}`)
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      return NextResponse.json(data, { status: response.status })
    }

    const buffer = await response.arrayBuffer()
    const headers = new Headers()
    headers.set('Content-Type', response.headers.get('content-type') || 'application/pdf')
    const disposition = response.headers.get('content-disposition')
    if (disposition) {
      headers.set('Content-Disposition', disposition)
    }

    return new NextResponse(buffer, { status: 200, headers })
  } catch (error) {
    console.error('Resume annotated proxy error:', error)
    return NextResponse.json({ error: 'Failed to connect to backend' }, { status: 500 })
  }
}
