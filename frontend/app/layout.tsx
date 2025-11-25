import './globals.css'
import { Analytics } from '@vercel/analytics/next'
import type { Metadata } from 'next'
import { MuiProvider } from '@/components/mui-provider'

export const metadata: Metadata = {
  title: 'IT企業エージェント',
  description: 'AI-powered IT Career Agent',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="ja">
      <body style={{ margin: 0, padding: 0 }}>
        <MuiProvider>
          {children}
          <Analytics />
        </MuiProvider>
      </body>
    </html>
  )
}
