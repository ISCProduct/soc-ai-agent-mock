'use client'

import React, { useState, useRef, useEffect } from 'react'
import {
  Box,
  Paper,
  TextField,
  IconButton,
  Typography,
  Avatar,
  Chip,
  Stack,
  CircularProgress,
  Button,
  Card,
  CardContent,
  Divider,
} from '@mui/material'
import { Send, SmartToy, Person, Refresh, Business, LocationOn, People, TrendingUp as TrendingUpIcon } from '@mui/icons-material'
import { sendMessage, getChatHistory } from '@/lib/api'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}

interface Company {
  id: string
  name: string
  industry: string
  location: string
  employees: string
  description: string
  matchScore: number
  tags: string[]
  techStack: string[]
}

// ローディングメッセージコンポーネント
function TypingIndicator() {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <CircularProgress size={16} />
      <Typography variant="body2" color="text.secondary">
        AIが考えています
      </Typography>
      <Box sx={{ display: 'flex', gap: 0.5 }}>
        {[0, 0.16, 0.32].map((delay, i) => (
          <Box
            key={i}
            sx={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              bgcolor: 'text.secondary',
              animation: 'bounce 1.4s infinite ease-in-out',
              animationDelay: `${delay}s`,
              '@keyframes bounce': {
                '0%, 80%, 100%': { transform: 'scale(0)' },
                '40%': { transform: 'scale(1)' },
              },
            }}
          />
        ))}
      </Box>
    </Box>
  )
}

// 企業情報表示コンポーネント
function CompanyResults({ onReset }: { onReset: () => void }) {
  const companies: Company[] = [
    {
      id: '1',
      name: '株式会社テックイノベーション',
      industry: 'Webサービス・AI開発',
      location: '東京都渋谷区',
      employees: '150名',
      description: '自社AIプロダクトを開発するベンチャー企業。最新技術を活用した開発環境で急成長中。',
      matchScore: 95,
      tags: ['リモートワーク', 'フレックス', '技術力重視'],
      techStack: ['Python', 'TypeScript', 'React', 'AWS'],
    },
    {
      id: '2',
      name: '日本システムソリューションズ株式会社',
      industry: 'SIer・受託開発',
      location: '東京都千代田区',
      employees: '2500名',
      description: '大手企業向けシステム開発を手がける老舗SIer。充実した研修制度と安定した環境。',
      matchScore: 88,
      tags: ['大手企業', '研修充実', '福利厚生'],
      techStack: ['Java', 'Oracle', 'Spring'],
    },
    {
      id: '3',
      name: 'クラウドテック株式会社',
      industry: 'クラウド・インフラ',
      location: '東京都港区',
      employees: '300名',
      description: 'クラウドインフラの設計・構築を専門とする企業。AWS/Azure/GCPの認定資格取得支援あり。',
      matchScore: 85,
      tags: ['インフラ特化', '資格支援', '技術研修'],
      techStack: ['AWS', 'Kubernetes', 'Terraform'],
    },
    {
      id: '4',
      name: 'データアナリティクス株式会社',
      industry: 'データ分析・BI',
      location: '東京都品川区',
      employees: '120名',
      description: 'ビッグデータ分析とBIツール開発を行う企業。データサイエンティストとして成長できる。',
      matchScore: 82,
      tags: ['データ分析', '成長企業', 'リモート可'],
      techStack: ['Python', 'SQL', 'Tableau', 'Spark'],
    },
    {
      id: '5',
      name: 'フィンテック株式会社',
      industry: '金融×IT',
      location: '東京都千代田区',
      employees: '250名',
      description: '金融業界向けのITソリューションを提供。高い技術力と金融知識を身につけられる。',
      matchScore: 80,
      tags: ['金融IT', '高給与', '成長分野'],
      techStack: ['Java', 'Python', 'Blockchain'],
    },
  ]

  return (
    <Box sx={{ 
      height: '100vh',
      display: 'flex',
      flexDirection: 'column',
      overflow: 'hidden',
      backgroundColor: '#fff',
    }}>
      {/* ヘッダー部分 */}
      <Box sx={{ 
        p: 3, 
        borderBottom: '1px solid #e0e0e0',
        backgroundColor: '#fff',
        flexShrink: 0,
      }}>
        <Box sx={{ textAlign: 'center' }}>
          <Typography variant="h4" fontWeight="bold" gutterBottom>
            🎉 分析完了！あなたに適した企業を5社に絞り込みました
          </Typography>
          <Typography variant="body1" color="text.secondary">
            4段階の分析に基づいて、最適なIT企業をマッチングしました
          </Typography>
        </Box>
      </Box>

      {/* スクロール可能なコンテンツエリア */}
      <Box sx={{ 
        flexGrow: 1,
        overflowY: 'auto',
        p: 3,
        backgroundColor: '#fafafa',
      }}>
        <Box sx={{ maxWidth: 1200, mx: 'auto' }}>
          <Stack spacing={3}>
            {companies.map((company, index) => (
              <Card key={company.id} elevation={3} sx={{ border: '2px solid', borderColor: 'primary.light' }}>
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Avatar sx={{ bgcolor: 'primary.main', width: 40, height: 40, fontWeight: 'bold' }}>
                        {index + 1}
                      </Avatar>
                      <Box>
                        <Typography variant="h6" fontWeight="bold">
                          {company.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {company.industry}
                        </Typography>
                      </Box>
                    </Box>
                    <Box sx={{ textAlign: 'right' }}>
                      <Typography variant="h4" color="primary.main" fontWeight="bold">
                        {company.matchScore}%
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        マッチ度
                      </Typography>
                    </Box>
                  </Box>

                  <Typography variant="body2" sx={{ mb: 2 }}>
                    {company.description}
                  </Typography>

                  <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LocationOn fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.location}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <People fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.employees}
                      </Typography>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <TrendingUpIcon fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.industry}
                      </Typography>
                    </Box>
                  </Stack>

                  <Box sx={{ mb: 2 }}>
                    <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                      技術スタック:
                    </Typography>
                    <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                      {company.techStack.map((tech, i) => (
                        <Chip key={i} label={tech} size="small" color="primary" variant="outlined" />
                      ))}
                    </Stack>
                  </Box>

                  <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    {company.tags.map((tag, i) => (
                      <Chip key={i} label={tag} size="small" />
                    ))}
                  </Stack>

                  <Box sx={{ mt: 2 }}>
                    <Button variant="contained" fullWidth>
                      詳細を見る
                    </Button>
                  </Box>
                </CardContent>
              </Card>
            ))}
          </Stack>

          <Box sx={{ textAlign: 'center', mt: 4, mb: 4 }}>
            <Button variant="outlined" size="large" startIcon={<Refresh />} onClick={onReset}>
              最初からやり直す
            </Button>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}

export function MuiChat() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [showResults, setShowResults] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages, isLoading])

  useEffect(() => {
    // ローカルストレージから復元
    const savedMessages = localStorage.getItem('chatMessages')
    const savedShowResults = localStorage.getItem('showResults')
    
    if (savedMessages) {
      try {
        const parsed = JSON.parse(savedMessages)
        setMessages(parsed.map((msg: any) => ({
          ...msg,
          timestamp: new Date(msg.timestamp)
        })))
      } catch (error) {
        console.log('[MUI Chat] Failed to parse saved messages:', error)
      }
    }
    
    if (savedShowResults === 'true') {
      setShowResults(true)
    }
  }, [])

  const handleSend = async () => {
    if (!input.trim() || isLoading) return

    const userMessage: Message = {
      id: String(Date.now()),
      role: 'user',
      content: input,
      timestamp: new Date(),
    }

    setMessages((prev) => {
      const newMessages = [...prev, userMessage]
      // ユーザーメッセージもローカルストレージに保存
      localStorage.setItem('chatMessages', JSON.stringify(newMessages))
      return newMessages
    })
    setInput('')
    setIsLoading(true)

    try {
      const response = await sendMessage(input)
      const assistantMessage: Message = {
        id: String(Date.now() + 1),
        role: 'assistant',
        content: response.message || 'エラーが発生しました',
        timestamp: new Date(),
      }
      setMessages((prev) => {
        const newMessages = [...prev, assistantMessage]
        
        // ローカルストレージに保存
        localStorage.setItem('chatMessages', JSON.stringify(newMessages))
        
        // 進捗状況を親コンポーネントに通知
        window.dispatchEvent(new CustomEvent('chatProgress', { 
          detail: { messageCount: newMessages.length } 
        }))
        
        // 20メッセージ（10往復）で企業情報表示
        if (newMessages.length >= 20) {
          setTimeout(() => {
            setShowResults(true)
            localStorage.setItem('showResults', 'true')
          }, 1000)
        }
        
        return newMessages
      })
    } catch (error) {
      console.error('[MUI Chat] Backend error:', error)
      const errorMessage: Message = {
        id: String(Date.now() + 1),
        role: 'assistant',
        content:
          'バックエンドとの接続に失敗しました。後ほど再試行してください。',
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, errorMessage])
    } finally {
      setIsLoading(false)
    }
  }

  const handleReset = () => {
    setMessages([])
    setShowResults(false)
    // ローカルストレージをクリア
    localStorage.removeItem('chatMessages')
    localStorage.removeItem('showResults')
    window.dispatchEvent(new CustomEvent('chatProgress', { 
      detail: { messageCount: 0 } 
    }))
  }

  const jobOptions = [
    '開発系エンジニア',
    'インフラエンジニア',
    '両方に興味がある',
    'まだ決めていない',
  ]

  if (showResults) {
    return <CompanyResults onReset={handleReset} />
  }

  return (
    <Box
      sx={{
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: '#fff',
      }}
    >
      <Box
        sx={{
          p: 2,
          borderBottom: '1px solid #e0e0e0',
          backgroundColor: '#fff',
        }}
      >
        <Typography variant="h5" sx={{ fontWeight: 600 }}>
          IT業界キャリアエージェント
        </Typography>
        <Typography variant="body2" color="text.secondary">
          4段階の分析を実施中 - {Math.min(4, Math.ceil(messages.length / 5))}/4 段階完了
        </Typography>
      </Box>

      <Box
        sx={{
          flexGrow: 1,
          overflowY: 'auto',
          p: 3,
          backgroundColor: '#fff',
        }}
      >
        {messages.length === 0 && (
          <Box sx={{ textAlign: 'center', mt: 8 }}>
            <SmartToy sx={{ fontSize: 64, color: '#9e9e9e', mb: 2 }} />
            <Typography variant="h6" color="text.secondary" gutterBottom>
              こんにちは！IT業界専門のキャリアエージェントです。
            </Typography>
            <Typography variant="body2" color="text.secondary">
              4万社余りのIT企業の中から、あなたに最適な企業を選定いたします。
              <br />
              まず、どのような職種を希望されますか？
            </Typography>
          </Box>
        )}

        {messages.map((message) => (
          <Box
            key={message.id}
            sx={{
              display: 'flex',
              mb: 3,
              justifyContent:
                message.role === 'user' ? 'flex-end' : 'flex-start',
            }}
          >
            {message.role === 'assistant' && (
              <Avatar
                sx={{
                  bgcolor: '#1976d2',
                  width: 36,
                  height: 36,
                  mr: 2,
                }}
              >
                <SmartToy sx={{ fontSize: 20 }} />
              </Avatar>
            )}
            <Paper
              elevation={1}
              sx={{
                p: 2,
                maxWidth: '70%',
                backgroundColor:
                  message.role === 'user' ? '#1976d2' : '#f5f5f5',
                color: message.role === 'user' ? '#fff' : '#000',
              }}
            >
              <Typography variant="body1">{message.content}</Typography>
            </Paper>
            {message.role === 'user' && (
              <Avatar
                sx={{
                  bgcolor: '#757575',
                  width: 36,
                  height: 36,
                  ml: 2,
                }}
              >
                <Person sx={{ fontSize: 20 }} />
              </Avatar>
            )}
          </Box>
        ))}

        {/* ローディングインジケーター */}
        {isLoading && (
          <Box
            sx={{
              display: 'flex',
              mb: 3,
              justifyContent: 'flex-start',
            }}
          >
            <Avatar
              sx={{
                bgcolor: '#1976d2',
                width: 36,
                height: 36,
                mr: 2,
              }}
            >
              <SmartToy sx={{ fontSize: 20 }} />
            </Avatar>
            <Paper
              elevation={1}
              sx={{
                p: 2,
                maxWidth: '70%',
                backgroundColor: '#f5f5f5',
              }}
            >
              <TypingIndicator />
            </Paper>
          </Box>
        )}

        {messages.length === 0 && (
          <Box sx={{ mt: 4 }}>
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ mb: 2, textAlign: 'center' }}
            >
              クイック選択：
            </Typography>
            <Stack
              direction="row"
              spacing={1}
              justifyContent="center"
              flexWrap="wrap"
              gap={1}
            >
              {jobOptions.map((option) => (
                <Chip
                  key={option}
                  label={option}
                  onClick={() => setInput(option)}
                  sx={{ cursor: 'pointer' }}
                />
              ))}
            </Stack>
          </Box>
        )}

        <div ref={messagesEndRef} />
      </Box>

      <Box
        sx={{
          p: 2,
          borderTop: '1px solid #e0e0e0',
          backgroundColor: '#fff',
        }}
      >
        <Box sx={{ display: 'flex', gap: 1 }}>
          <TextField
            fullWidth
            placeholder="メッセージを入力..."
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyPress={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSend()
              }
            }}
            disabled={isLoading}
            variant="outlined"
            size="small"
            sx={{
              '& .MuiOutlinedInput-root': {
                borderRadius: 2,
              },
            }}
          />
          <IconButton
            color="primary"
            onClick={handleSend}
            disabled={!input.trim() || isLoading}
            sx={{
              bgcolor: '#1976d2',
              color: '#fff',
              '&:hover': {
                bgcolor: '#1565c0',
              },
              '&.Mui-disabled': {
                bgcolor: '#e0e0e0',
              },
            }}
          >
            <Send />
          </IconButton>
        </Box>
      </Box>
    </Box>
  )
}
