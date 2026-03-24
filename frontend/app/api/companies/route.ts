import { NextRequest, NextResponse } from 'next/server'

const API_BASE_URL = process.env.BACKEND_URL || 'http://app:8080'

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const limit = searchParams.get('limit') || '10'
    const offset = searchParams.get('offset') || '0'
    const industry = searchParams.get('industry') || ''
    const name = searchParams.get('name') || ''
    const tech = searchParams.get('tech') || ''

    let url = `${API_BASE_URL}/api/companies?limit=${limit}&offset=${offset}`
    if (industry) {
      url += `&industry=${encodeURIComponent(industry)}`
    }
    if (name) {
      url += `&name=${encodeURIComponent(name)}`
    }
    if (tech) {
      url += `&tech=${encodeURIComponent(tech)}`
    }
    
    console.log('[API] Fetching companies from:', url)
    
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    })

    if (!response.ok) {
      console.error('[API] Backend error:', response.status, response.statusText)
      return NextResponse.json(
        { error: 'Failed to fetch companies from backend' },
        { status: response.status }
      )
    }

    const data = await response.json()
    console.log('[API] Companies fetched:', data.companies?.length || 0)
    
    return NextResponse.json(data)
  } catch (error) {
    console.error('[API] Error fetching companies:', error)
    return NextResponse.json(
      { error: 'Internal server error', details: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    )
  }
}
