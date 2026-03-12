import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  // MUI emotion CSS-in-JS のSSR対応
  compiler: {
    emotion: true,
  },
}

export default nextConfig
