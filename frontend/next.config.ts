import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Docker開発環境でのホットリロード設定（Turbopack対応）
  turbopack: {
    // Turbopackは自動的にファイル変更を検知するため、空オブジェクトで設定
  },
  // MUI emotion CSS-in-JS のSSR対応
  compiler: {
    emotion: true,
  },
}

export default nextConfig
