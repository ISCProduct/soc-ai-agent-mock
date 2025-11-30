'use client'

import React, { useState, useRef, useEffect } from 'react'
import { useRouter } from 'next/navigation'
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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material'
import { Send, SmartToy, Person, Refresh, Business, LocationOn, People, TrendingUp as TrendingUpIcon } from '@mui/icons-material'
import { sendChatMessage, getChatHistory, getUserScores, ChatRequest, ChatResponse } from '@/lib/api'
import { authService } from '@/lib/auth'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
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
        {[0, 0.16, 0.32].map((delay: any, i: any) => (
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

export function MuiChat() {
  const router = useRouter()
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [analysisComplete, setAnalysisComplete] = useState(false)
  const [sessionId, setSessionId] = useState('')
  const [userId, setUserId] = useState<number>(0)
  const [questionCount, setQuestionCount] = useState(0)
  const [totalQuestions, setTotalQuestions] = useState(15)
  const [mounted, setMounted] = useState(false)
  const [showCompletionModal, setShowCompletionModal] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages, isLoading])

  useEffect(() => {
    setMounted(true)
    
    const initializeChat = async () => {
      // ユーザー情報を初期化
      const user = authService.getStoredUser()
      const currentUserId = user ? user.user_id : 1
      setUserId(currentUserId)
      
      // セッションIDの生成または復元（sessionStorageのみ使用）
      let storedSessionId = sessionStorage.getItem('chatSessionId')
      if (!storedSessionId) {
        storedSessionId = `session_${Date.now()}_${Math.random().toString(36).substring(7)}`
        sessionStorage.setItem('chatSessionId', storedSessionId)
      }
      setSessionId(storedSessionId)
      
      // バックエンドからチャット履歴を取得
      try {
        console.log('[MUI Chat] Loading history for session:', storedSessionId)
        const history = await getChatHistory(storedSessionId)
        console.log('[MUI Chat] History loaded:', history?.length, 'messages')
        
        if (history && history.length > 0) {
          // 履歴が存在する場合は復元
          const restoredMessages: Message[] = history.map((msg) => ({
            id: String(msg.id),
            role: msg.role,
            content: msg.content,
            timestamp: new Date(msg.created_at),
          }))
          setMessages(restoredMessages)
          setQuestionCount(history.filter(msg => msg.role === 'user').length)
          
          // スコアを取得して分析完了状態を判定
          const scores = await getUserScores(currentUserId, storedSessionId)
          console.log('[MUI Chat] Scores loaded:', scores?.length)
          if (scores && scores.length > 0) {
            setAnalysisComplete(true)
          }
        } else {
          // 履歴がない場合: バックエンドでセッションを開始
          console.log('[MUI Chat] No history found, starting new session')
          const initialResponse = await sendChatMessage({
            user_id: currentUserId,
            session_id: storedSessionId,
            message: 'START_SESSION',
            industry_id: 1,
            job_category_id: 1,
          })
          
          const initialMessage: Message = {
            id: '0',
            role: 'assistant',
            content: initialResponse.response,
            timestamp: new Date(),
          }
          setMessages([initialMessage])
        }
      } catch (error) {
        console.error('[MUI Chat] Failed to load history:', error)
        // エラー時は初回メッセージを表示
        const initialMessage: Message = {
          id: '0',
          role: 'assistant',
          content: 'こんにちは！IT業界への就職をサポートする適性診断AIです。\n\nこれから約10-15問の質問を通じて、あなたの適性を分析し、最適な企業をご提案します。\n質問は**AIが動的に生成**するため、あなたの回答に応じて変化します。\n\nまず、どのようなIT職種に興味がありますか？\n\n例：\n- Webエンジニア\n- インフラエンジニア\n- データサイエンティスト\n- セキュリティエンジニア\n- モバイルアプリ開発者',
          timestamp: new Date(),
        }
        setMessages([initialMessage])
      }
    }
    
    initializeChat()
  }, [])

  const handleSend = async () => {
    if (!input.trim() || isLoading || !sessionId || !userId) return

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
      // バックエンドのAI機能を活用
      const chatRequest: ChatRequest = {
        user_id: userId,
        session_id: sessionId,
        message: input,
        industry_id: 1, // IT業界
        job_category_id: 1, // 開発職
      }
      
      const response: ChatResponse = await sendChatMessage(chatRequest)
      
      const assistantMessage: Message = {
        id: String(Date.now() + 1),
        role: 'assistant',
        content: response.response || 'エラーが発生しました',
        timestamp: new Date(),
      }
      
      setMessages((prev) => {
        const newMessages = [...prev, assistantMessage]
        
        // 質問カウントの更新
        const newCount = response.answered_questions ?? questionCount + 1
        setQuestionCount(newCount)
        setTotalQuestions(response.total_questions ?? 15)
        
        // 進捗状況を親コンポーネントに通知
        window.dispatchEvent(new CustomEvent('chatProgress', { 
          detail: { 
            messageCount: newMessages.length,
            questionCount: newCount,
            totalQuestions: response.total_questions ?? 15,
          } 
        }))
        
        // **重要: バックエンドのis_completeのみを信頼**
        // バックエンドがtrueを返した時は分析完了状態にする
        console.log('[MUI Chat] is_complete:', response.is_complete, 'type:', typeof response.is_complete)
        if (response.is_complete === true) {
          console.log('[MUI Chat] AI分析完了 - モーダルを表示します')
          setTimeout(() => {
            setAnalysisComplete(true)
            setShowCompletionModal(true)
          }, 1000)
        } else {
          console.log(`[MUI Chat] 質問継続中 (${newCount}/${response.total_questions ?? 15})`)
          // 明示的にfalseを設定
          setAnalysisComplete(false)
        }
        
        return newMessages
      })
    } catch (error) {
      console.error('[MUI Chat] Backend error:', error)
      const errorMessage: Message = {
        id: String(Date.now() + 1),
        role: 'assistant',
        content:
          'バックエンドとの接続に失敗しました。後ほど再試行してください。\n\nエラー: ' + (error as Error).message,
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, errorMessage])
    } finally {
      setIsLoading(false)
    }
  }

  const handleReset = () => {
    // すべての状態をクリア
    setMessages([])
    setAnalysisComplete(false)
    setQuestionCount(0)
    setTotalQuestions(15)
    
    // セッションIDも新しく生成
    const newSessionId = `session_${Date.now()}_${Math.random().toString(36).substring(7)}`
    setSessionId(newSessionId)
    sessionStorage.setItem('chatSessionId', newSessionId)
    
    // 初回メッセージを再設定
    const initialMessage: Message = {
      id: '0',
      role: 'assistant',
      content: 'こんにちは！IT業界への就職をサポートする適性診断AIです。\n\nこれから約10-15問の質問を通じて、あなたの適性を分析し、最適な企業をご提案します。\n質問は**AIが動的に生成**するため、あなたの回答に応じて変化します。\n\nまず、どのようなIT職種に興味がありますか？\n\n例：\n- Webエンジニア\n- インフラエンジニア\n- データサイエンティスト\n- セキュリティエンジニア\n- モバイルアプリ開発者',
      timestamp: new Date(),
    }
    setMessages([initialMessage])
    localStorage.setItem('chatMessages', JSON.stringify([initialMessage]))
    
    window.dispatchEvent(new CustomEvent('chatProgress', { 
      detail: { messageCount: 1, questionCount: 0, totalQuestions: 15 } 
    }))
  }

  const handleViewResults = () => {
    setShowCompletionModal(false)
    router.push(`/results?user_id=${userId}&session_id=${sessionId}`)
  }

  const handleContinueChat = () => {
    setShowCompletionModal(false)
  }

  const jobOptions = [
    '開発系エンジニア',
    'インフラエンジニア',
    '両方に興味がある',
    'まだ決めていない',
  ]

  if (!mounted) {
    return null
  }

  return (
    <>
      {/* 分析完了モーダル */}
      <Dialog
        open={showCompletionModal}
        onClose={handleContinueChat}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: 2,
            p: 2,
          }
        }}
      >
        <DialogTitle sx={{ textAlign: 'center', pb: 1 }}>
          <Typography variant="h5" component="div" sx={{ fontWeight: 'bold', color: 'primary.main' }}>
            🎉 分析が完了しました！
          </Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2 }}>
          <Typography variant="body1" sx={{ textAlign: 'center', mb: 2 }}>
            あなたの適性を分析し、最適な企業をマッチングしました。
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center' }}>
            結果ページで詳細な企業情報を確認できます。
          </Typography>
        </DialogContent>
        <DialogActions sx={{ justifyContent: 'center', gap: 2, pb: 2 }}>
          <Button
            onClick={handleContinueChat}
            variant="outlined"
            size="large"
            sx={{ minWidth: 140 }}
          >
            チャットを続ける
          </Button>
          <Button
            onClick={handleViewResults}
            variant="contained"
            size="large"
            sx={{ minWidth: 140 }}
          >
            結果を見る
          </Button>
        </DialogActions>
      </Dialog>

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
          AI適性診断 - {questionCount}/{totalQuestions} 問完了 
          {questionCount > 0 && ` (${Math.round((questionCount / totalQuestions) * 100)}%)`}
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
        {analysisComplete ? (
          <Box sx={{ textAlign: 'center' }}>
            <Button
              variant="contained"
              size="large"
              onClick={() => setShowCompletionModal(true)}
              sx={{
                py: 2,
                px: 4,
                fontSize: '1.1rem',
                fontWeight: 'bold',
              }}
            >
              🎉 分析完了！結果を見る
            </Button>
            <Typography variant="caption" display="block" sx={{ mt: 1 }} color="text.secondary">
              あなたに最適な企業をマッチングしました
            </Typography>
          </Box>
        ) : (
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
        )}
      </Box>
      </Box>
    </>
  )
}
