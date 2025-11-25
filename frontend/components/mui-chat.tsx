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
} from '@mui/material'
import { Send, SmartToy, Person } from '@mui/icons-material'
import { sendMessage, getChatHistory } from '@/lib/api'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}

export function MuiChat() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    const loadHistory = async () => {
      try {
        const sessionId = `session_${Date.now()}_${Math.random().toString(36).substring(7)}`
        const history = await getChatHistory(sessionId)
        if (history && history.length > 0) {
          setMessages(
            history.map((msg: any) => ({
              id: msg.id || String(Date.now()),
              role: msg.role,
              content: msg.content,
              timestamp: new Date(msg.timestamp || Date.now()),
            }))
          )
        }
      } catch (error) {
        console.log('[MUI Chat] Failed to load chat history:', error)
      }
    }
    loadHistory()
  }, [])

  const handleSend = async () => {
    if (!input.trim() || isLoading) return

    const userMessage: Message = {
      id: String(Date.now()),
      role: 'user',
      content: input,
      timestamp: new Date(),
    }

    setMessages((prev) => [...prev, userMessage])
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
      setMessages((prev) => [...prev, assistantMessage])
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

  const jobOptions = [
    '開発系エンジニア',
    'インフラエンジニア',
    '両方に興味がある',
    'まだ決めていない',
  ]

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
          4万社から最適な企業を選定 (バックエンド連携中)
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
