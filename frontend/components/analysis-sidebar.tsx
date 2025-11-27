'use client'

import React, { useState, useEffect } from 'react'
import {
  Box,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
  LinearProgress,
  Divider,
  Chip,
  Avatar,
  IconButton,
} from '@mui/material'
import {
  CheckCircle,
  RadioButtonUnchecked,
  Work,
  Psychology,
  TrendingUp,
  Speed,
  EmojiEvents,
  Logout,
} from '@mui/icons-material'
import { User } from '@/lib/auth'

const DRAWER_WIDTH = 280

interface AnalysisStep {
  id: string
  label: string
  icon: React.ReactNode
  completed: boolean
  progress?: number
}

interface AnalysisSidebarProps {
  user: User
  onLogout: () => void
}

export function AnalysisSidebar({ user, onLogout }: AnalysisSidebarProps) {
  const [messageCount, setMessageCount] = useState(0)
  
  // チャットの進行状況をリスンして進捗を更新
  useEffect(() => {
    const handleChatProgress = (event: CustomEvent) => {
      setMessageCount(event.detail.messageCount || 0)
    }
    
    window.addEventListener('chatProgress', handleChatProgress as EventListener)
    return () => {
      window.removeEventListener('chatProgress', handleChatProgress as EventListener)
    }
  }, [])

  // メッセージ数に応じて進捗を計算
  const calculateProgress = () => {
    // 職種分析: メッセージ1-5 (0% → 100%)
    // 興味分析: メッセージ6-10 (0% → 100%)
    // 適性分析: メッセージ11-15 (0% → 100%)
    // 将来分析: メッセージ16-20 (0% → 100%)
    
    const jobProgress = Math.min(100, Math.floor((messageCount / 5) * 100))
    const interestProgress = messageCount > 5 ? Math.min(100, Math.floor(((messageCount - 5) / 5) * 100)) : 0
    const aptitudeProgress = messageCount > 10 ? Math.min(100, Math.floor(((messageCount - 10) / 5) * 100)) : 0
    const futureProgress = messageCount > 15 ? Math.min(100, Math.floor(((messageCount - 15) / 5) * 100)) : 0
    
    return {
      job: jobProgress,
      interest: interestProgress,
      aptitude: aptitudeProgress,
      future: futureProgress,
    }
  }

  const progress = calculateProgress()

  const analysisSteps: AnalysisStep[] = [
    {
      id: 'backend',
      label: 'バックエンド連携',
      icon: <Speed />,
      completed: true,
    },
    {
      id: 'job',
      label: progress.job === 100 ? '職種分析完了' : '職種分析進行中',
      icon: <Work />,
      completed: progress.job === 100,
      progress: progress.job < 100 ? progress.job : undefined,
    },
    {
      id: 'interest',
      label: progress.interest === 100 ? '興味分析完了' : progress.interest > 0 ? '興味分析進行中' : '興味分析待機中',
      icon: <Psychology />,
      completed: progress.interest === 100,
      progress: progress.interest > 0 && progress.interest < 100 ? progress.interest : undefined,
    },
    {
      id: 'aptitude',
      label: progress.aptitude === 100 ? '適性分析完了' : progress.aptitude > 0 ? '適性分析進行中' : '適性分析待機中',
      icon: <TrendingUp />,
      completed: progress.aptitude === 100,
      progress: progress.aptitude > 0 && progress.aptitude < 100 ? progress.aptitude : undefined,
    },
    {
      id: 'future',
      label: progress.future === 100 ? '将来分析完了' : progress.future > 0 ? '将来分析進行中' : '将来分析待機中',
      icon: <EmojiEvents />,
      completed: progress.future === 100,
      progress: progress.future > 0 && progress.future < 100 ? progress.future : undefined,
    },
  ]

  return (
    <Drawer
      variant="permanent"
      sx={{
        width: DRAWER_WIDTH,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
          backgroundColor: '#f7f7f8',
          borderRight: '1px solid #e0e0e0',
        },
      }}
    >
      <Box sx={{ p: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2, gap: 1 }}>
          <Avatar sx={{ bgcolor: user.is_guest ? 'grey.500' : 'primary.main' }}>
            {user.name.charAt(0).toUpperCase()}
          </Avatar>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="subtitle2" noWrap sx={{ fontWeight: 600 }}>
              {user.name}
            </Typography>
            {user.is_guest && (
              <Chip label="ゲスト" size="small" sx={{ height: 18, fontSize: '0.7rem' }} />
            )}
            {user.oauth_provider && (
              <Chip 
                label={user.oauth_provider} 
                size="small" 
                sx={{ height: 18, fontSize: '0.7rem', textTransform: 'capitalize' }} 
              />
            )}
          </Box>
          <IconButton size="small" onClick={onLogout} title="ログアウト">
            <Logout fontSize="small" />
          </IconButton>
        </Box>

        <Divider sx={{ mb: 2 }} />

        <Typography variant="h6" sx={{ fontWeight: 600, mb: 1 }}>
          分析進捗
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          4段階の分析を実施中
        </Typography>

        <List sx={{ p: 0 }}>
          {analysisSteps.map((step, index) => (
            <React.Fragment key={step.id}>
              <ListItem
                sx={{
                  borderRadius: 1,
                  mb: 0.5,
                  backgroundColor: step.completed ? '#e8f5e9' : 'transparent',
                  '&:hover': {
                    backgroundColor: step.completed ? '#e8f5e9' : '#f0f0f0',
                  },
                }}
              >
                <ListItemIcon sx={{ minWidth: 36 }}>
                  {step.completed ? (
                    <CheckCircle color="success" />
                  ) : (
                    <RadioButtonUnchecked color="action" />
                  )}
                </ListItemIcon>
                <ListItemText
                  primary={step.label}
                  primaryTypographyProps={{
                    fontSize: '0.875rem',
                    fontWeight: step.completed ? 500 : 400,
                  }}
                />
              </ListItem>
              {step.progress !== undefined && (
                <Box sx={{ px: 2, pb: 1 }}>
                  <LinearProgress
                    variant="determinate"
                    value={step.progress}
                    sx={{ height: 6, borderRadius: 3 }}
                  />
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ mt: 0.5, display: 'block' }}
                  >
                    {step.progress}% 完了
                  </Typography>
                </Box>
              )}
            </React.Fragment>
          ))}
        </List>

        <Divider sx={{ my: 2 }} />

        <Box>
          <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
            IT業界キャリアエージェント
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            4万社から最適な企業を選定 (バックエンド連携中)
          </Typography>
          <Chip
            label="AI分析中"
            color="primary"
            size="small"
            sx={{ fontSize: '0.75rem' }}
          />
        </Box>
      </Box>
    </Drawer>
  )
}
