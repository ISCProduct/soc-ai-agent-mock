import { NextRequest, NextResponse } from 'next/server'
import { spawn } from 'child_process'
import path from 'path'

export const dynamic = 'force-dynamic'

// Python パイプラインのルートディレクトリ
const PIPELINE_DIR = path.resolve(process.cwd(), '../tools/company-graph')
const PIPELINE_SCRIPT = path.join(PIPELINE_DIR, 'pipeline.py')

export async function POST(request: NextRequest) {
  const body = await request.json().catch(() => ({}))
  const {
    sites = ['mynavi', 'rikunabi', 'career_tasu'],
    query = 'IT',
    pages = 2,
    year,
    threshold = 0.75,
  } = body

  const outputDir = path.join(PIPELINE_DIR, 'output')

  const args = [
    PIPELINE_SCRIPT,
    '--sites', ...sites,
    '--query', String(query),
    '--pages', String(pages),
    '--out', outputDir,
    '--threshold', String(threshold),
  ]
  if (year) {
    args.push('--year', String(year))
  }

  return new Promise<NextResponse>((resolve) => {
    const proc = spawn('python3', args, {
      cwd: PIPELINE_DIR,
      env: {
        ...process.env,
        PYTHONPATH: PIPELINE_DIR,
      },
      timeout: 300_000, // 5 min
    })

    const stdout: string[] = []
    const stderr: string[] = []

    proc.stdout.on('data', (d: Buffer) => stdout.push(d.toString()))
    proc.stderr.on('data', (d: Buffer) => stderr.push(d.toString()))

    proc.on('close', (code) => {
      const logs = [...stdout, ...stderr].join('')
      if (code === 0) {
        resolve(
          NextResponse.json({ ok: true, logs, output_dir: outputDir }, { status: 200 }),
        )
      } else {
        resolve(
          NextResponse.json({ ok: false, logs, error: `Process exited with code ${code}` }, { status: 500 }),
        )
      }
    })

    proc.on('error', (err) => {
      resolve(
        NextResponse.json({ ok: false, error: err.message, logs: stderr.join('') }, { status: 500 }),
      )
    })
  })
}

/** 年度の自動計算（フロントエンド表示用）*/
export async function GET() {
  const today = new Date()
  const month = today.getMonth() + 1 // 1-based
  const year = today.getFullYear()
  const targetYear = month >= 4 ? year + 2 : year + 1

  return NextResponse.json({ target_year: targetYear })
}
