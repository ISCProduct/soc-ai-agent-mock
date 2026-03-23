// /api/healthz は ECS ターゲットグループ・ALB・Kubernetes の標準ヘルスチェックパス
// Next.js App Router の API Route として実装
import { NextResponse } from 'next/server'

export async function GET() {
  return NextResponse.json({ status: 'ok' })
}
