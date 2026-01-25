'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Divider,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import { authService } from '@/lib/auth'

type AdminUser = {
  id: number
  email: string
  name: string
  is_guest: boolean
  is_admin: boolean
  target_level?: string
  school_name?: string
  created_at: string
  updated_at: string
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const user = authService.getStoredUser()
    if (!user?.is_admin) {
      window.location.href = '/'
    }
  }, [])

  const loadUsers = async () => {
    setError('')
    const response = await fetch('/api/admin/users')
    const data = await response.json()
    if (!response.ok) {
      setError(data?.error || 'ユーザー一覧の取得に失敗しました')
      return
    }
    setUsers(data?.users || [])
  }

  useEffect(() => {
    loadUsers()
  }, [])

  const filtered = useMemo(() => {
    if (!query) return users
    const q = query.toLowerCase()
    return users.filter((user) =>
      `${user.email} ${user.name} ${user.school_name || ''}`.toLowerCase().includes(q),
    )
  }, [users, query])

  const handleToggleAdmin = async (user: AdminUser) => {
    const admin = authService.getStoredUser()
    if (!admin?.email) return
    setLoading(true)
    const response = await fetch(`/api/admin/users/${user.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Email': admin.email,
      },
      body: JSON.stringify({ is_admin: !user.is_admin }),
    })
    const data = await response.json()
    if (!response.ok) {
      setError(data?.error || '権限更新に失敗しました')
      setLoading(false)
      return
    }
    setUsers((prev) => prev.map((item) => (item.id === user.id ? { ...item, ...data } : item)))
    setLoading(false)
  }

  return (
    <Box sx={{ p: 4, maxWidth: 1100, mx: 'auto' }}>
      <Typography variant="h4" fontWeight="bold" gutterBottom>
        ユーザー管理
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        管理者権限の付与やユーザー情報の確認を行います。
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Stack spacing={2}>
            <TextField
              label="検索 (メール/名前/学校名)"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              fullWidth
            />
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            ユーザー一覧
          </Typography>
          <Divider sx={{ mb: 2 }} />
          {filtered.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              該当するユーザーがいません。
            </Typography>
          ) : (
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>ユーザー</TableCell>
                    <TableCell>区分</TableCell>
                    <TableCell>学校名</TableCell>
                    <TableCell>権限</TableCell>
                    <TableCell>更新日時</TableCell>
                    <TableCell align="right">操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filtered.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell>
                        <Typography variant="subtitle2" fontWeight="bold">
                          {user.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {user.email}
                        </Typography>
                      </TableCell>
                      <TableCell>{user.target_level || '未設定'}</TableCell>
                      <TableCell>{user.school_name || '未設定'}</TableCell>
                      <TableCell>
                        <Chip
                          label={user.is_admin ? '管理者' : user.is_guest ? 'ゲスト' : '一般'}
                          size="small"
                          color={user.is_admin ? 'success' : 'default'}
                        />
                      </TableCell>
                      <TableCell>{user.updated_at}</TableCell>
                      <TableCell align="right">
                        <Button
                          size="small"
                          variant="outlined"
                          disabled={loading}
                          onClick={() => handleToggleAdmin(user)}
                        >
                          {user.is_admin ? '管理者権限を外す' : '管理者にする'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>
    </Box>
  )
}
