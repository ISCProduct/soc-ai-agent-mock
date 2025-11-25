'use client'

import React from 'react'
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
} from '@mui/material'
import {
  CheckCircle,
  RadioButtonUnchecked,
  Work,
  Psychology,
  TrendingUp,
  Speed,
  EmojiEvents,
} from '@mui/icons-material'

const DRAWER_WIDTH = 280

interface AnalysisStep {
  id: string
  label: string
  icon: React.ReactNode
  completed: boolean
  progress?: number
}

export function AnalysisSidebar() {
  const analysisSteps: AnalysisStep[] = [
    {
      id: 'backend',
      label: 'バックエンド連携',
      icon: <Speed />,
      completed: true,
    },
    {
      id: 'job',
      label: '職種分析進行中',
      icon: <Work />,
      completed: false,
      progress: 45,
    },
    {
      id: 'interest',
      label: '興味分析待機中',
      icon: <Psychology />,
      completed: false,
    },
    {
      id: 'aptitude',
      label: '適性分析待機中',
      icon: <TrendingUp />,
      completed: false,
    },
    {
      id: 'future',
      label: '将来分析待機中',
      icon: <EmojiEvents />,
      completed: false,
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
