'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Grid,
  Stack,
  Typography,
} from '@mui/material'
import { authService } from '@/lib/auth'

export default function AdminDashboardPage() {
  const [companyCount, setCompanyCount] = useState<number | null>(null)
  const [crawlCount, setCrawlCount] = useState<number | null>(null)

  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  useEffect(() => {
    const loadCounts = async () => {
      try {
        const [companiesRes, crawlRes] = await Promise.all([
          fetch('/api/admin/companies'),
          fetch('/api/admin/crawl-sources'),
        ])
        const companiesData = await companiesRes.json()
        const crawlData = await crawlRes.json()
        setCompanyCount(companiesData?.companies?.length ?? 0)
        setCrawlCount(crawlData?.sources?.length ?? 0)
      } catch {
        setCompanyCount(null)
        setCrawlCount(null)
      }
    }
    loadCounts()
  }, [])

  return (
    <Box sx={{ p: 4, maxWidth: 960, mx: 'auto' }}>
      <Typography variant="h4" fontWeight="bold" gutterBottom>
        管理者ダッシュボード
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        管理者向けの操作メニューです。権限がない場合は表示されません。
      </Typography>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                企業データ
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                企業プロフィールの追加・更新、公開ステータスの管理を行います。
              </Typography>
              <Chip
                label={companyCount === null ? '読み込み中' : `登録数 ${companyCount}`}
                size="small"
                sx={{ mb: 2 }}
              />
              <Divider sx={{ mb: 2 }} />
              <Button variant="contained" component={Link} href="/admin/companies">
                企業管理へ
              </Button>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                自動クローリング
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                週次・月次のスケジュールを設定し、企業データを自動更新します。
              </Typography>
              <Chip
                label={crawlCount === null ? '読み込み中' : `スケジュール ${crawlCount}`}
                size="small"
                sx={{ mb: 2 }}
              />
              <Divider sx={{ mb: 2 }} />
              <Button variant="contained" component={Link} href="/admin/crawling">
                クローリング管理へ
              </Button>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                ユーザー管理
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                ユーザー情報の確認と管理者権限の付与を行います。
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <Button variant="contained" component={Link} href="/admin/users">
                ユーザー管理へ
              </Button>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                監査ログ
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                管理者操作の履歴を確認できます。
              </Typography>
              <Divider sx={{ mb: 2 }} />
              <Button variant="contained" component={Link} href="/admin/audit-logs">
                監査ログへ
              </Button>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Stack spacing={2}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              近日追加予定
            </Typography>
            <Typography variant="body2" color="text.secondary">
              ユーザー権限管理、監査ログ、通知設定などを順次追加します。
            </Typography>
          </CardContent>
        </Card>
      </Stack>
    </Box>
  )
}
