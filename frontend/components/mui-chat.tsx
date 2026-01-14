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

interface PhaseProgress {
  phase_name: string
  display_name: string
  questions_asked: number
  valid_answers: number
  completion_score: number
  is_completed: boolean
  min_questions: number
  max_questions: number
}

const makeMessageId = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`

interface ChoiceOption {
  value: string
  label: string
  text: string
}

function extractChoices(content: string): ChoiceOption[] {
  const lines = content.split('\n')
  const choices: ChoiceOption[] = []
  for (const line of lines) {
    const trimmedLine = line.trim()
    if (!trimmedLine) {
      continue
    }
    let match = trimmedLine.match(/^([A-E])\)\s*(.+)$/)
    if (!match) {
      match = trimmedLine.match(/^([A-E])[：、.．]\s*(.+)$/)
    }
    if (match) {
      choices.push({ value: match[1], label: match[1], text: match[2].trim() })
      continue
    }
    match = trimmedLine.match(/^(\d+)[\.\)．]\s*(.+)$/)
    if (match) {
      choices.push({ value: match[1], label: match[1], text: match[2].trim() })
    }
  }
  return choices
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
  const [allPhasesCompleted, setAllPhasesCompleted] = useState(false)
  const [sessionId, setSessionId] = useState('')
  const [userId, setUserId] = useState<number>(0)
  const [questionCount, setQuestionCount] = useState(0)
  const [totalQuestions, setTotalQuestions] = useState(15)
  const [mounted, setMounted] = useState(false)
  const [showCompletionModal, setShowCompletionModal] = useState(false)
  const [showEndChatModal, setShowEndChatModal] = useState(false)
  const [showTerminationModal, setShowTerminationModal] = useState(false)
  const [otherChoiceActive, setOtherChoiceActive] = useState(false)
  const [phaseProgresses, setPhaseProgresses] = useState<PhaseProgress[] | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const progressTotals = (() => {
    if (!phaseProgresses || phaseProgresses.length === 0) return null
    let valid = 0
    let asked = 0
    for (const phase of phaseProgresses) {
      asked += phase.questions_asked || 0
      valid += phase.valid_answers || 0
    }
    if (asked <= 0) return null
    return {
      valid,
      required: asked,
      percent: Math.round((valid / asked) * 100),
    }
  })()

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
      if (!user || !user.name || !user.target_level) {
        router.replace('/onboarding')
        return
      }
      const currentUserId = user.user_id
      setUserId(currentUserId)
      
      // セッションIDの取得優先順位:
      // 1. localStorageから（履歴ページから選択した場合）
      // 2. sessionStorageから（ページリロード時の復元）
      // 3. 新規生成
      let storedSessionId = localStorage.getItem('currentSessionId')
      if (storedSessionId) {
        console.log('[MUI Chat] Loading session from localStorage:', storedSessionId)
        // localStorageから読み込んだ後は削除
        localStorage.removeItem('currentSessionId')
      } else {
        storedSessionId = sessionStorage.getItem('chatSessionId')
      }
      
      if (!storedSessionId) {
        storedSessionId = `session_${Date.now()}_${Math.random().toString(36).substring(7)}`
        console.log('[MUI Chat] Created new session:', storedSessionId)
      }
      
      sessionStorage.setItem('chatSessionId', storedSessionId)
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
          const userQuestionCount = history.filter(msg => msg.role === 'user').length
          setQuestionCount(userQuestionCount)
          
          // sessionStorageから総質問数を復元（なければデフォルト15）
          const savedTotalQuestions = sessionStorage.getItem('totalQuestions')
          const restoredTotalQuestions = savedTotalQuestions ? parseInt(savedTotalQuestions) : 15
          setTotalQuestions(restoredTotalQuestions)
          const savedPhases = sessionStorage.getItem('phaseProgress')
          const restoredPhases = savedPhases ? JSON.parse(savedPhases) : null
          if (Array.isArray(restoredPhases)) {
            setPhaseProgresses(restoredPhases)
          }
          
          // 進捗状況を通知（履歴復元時）
          setTimeout(() => {
            window.dispatchEvent(new CustomEvent('chatProgress', { 
              detail: { 
                messageCount: restoredMessages.length,
                questionCount: userQuestionCount,
                totalQuestions: restoredTotalQuestions,
                phases: restoredPhases,
              } 
            }))
            console.log('[MUI Chat] Progress restored:', userQuestionCount, '/', restoredTotalQuestions)
          }, 100)
          
          // 最後のメッセージが完了メッセージかチェック
          const lastMessage = history[history.length - 1]
          const isCompletionMessage = lastMessage?.content?.includes('分析が完了しました') || 
                                     lastMessage?.content?.includes('全てのフェーズが完了') ||
                                     lastMessage?.content?.includes('最適な企業をマッチング')
          
          if (isCompletionMessage) {
            console.log('[MUI Chat] Session already completed, showing completion state')
            setAnalysisComplete(true)
            setAllPhasesCompleted(true)
          }
        } else {
          // 履歴がない場合: AIのあいさつメッセージを表示（バックエンドには送信しない）
          console.log('[MUI Chat] No history found, displaying initial greeting')
          
          // AIのあいさつメッセージ（フロントエンドのみで表示）
          const greetingMessage = 'こんにちは！IT業界専門のキャリアエージェントです。\n\nこれから約10-15問の質問を通じて、あなたの適性を分析し、最適な企業をご提案します。\n質問は動的に生成されるため、あなたの回答に応じて変化します。\n\nまず、どのようなIT職種に興味がありますか？\n\n例：\n- Webエンジニア\n- インフラエンジニア\n- データサイエンティスト\n- セキュリティエンジニア\n- モバイルアプリ開発者'
          
          const initialMessage: Message = {
            id: '0',
            role: 'assistant',
            content: greetingMessage,
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
          content: 'こんにちは！IT業界専門のキャリアエージェントです。\n\nこれから約10-15問の質問を通じて、あなたの適性を分析し、最適な企業をご提案します。\n質問は動的に生成されるため、あなたの回答に応じて変化します。\n\nまず、どのようなIT職種に興味がありますか？\n\n例：\n- Webエンジニア\n- インフラエンジニア\n- データサイエンティスト\n- セキュリティエンジニア\n- モバイルアプリ開発者',
          timestamp: new Date(),
        }
        setMessages([initialMessage])
      }
    }
    
    initializeChat()
  }, [])

  const handleSend = async (overrideMessage?: string) => {
    const messageText = (overrideMessage ?? input).trim()
    if (!messageText || isLoading || !sessionId || !userId) return
    
    // 分析完了後はメッセージ送信を無効化
    if (analysisComplete) {
      console.log('[MUI Chat] Analysis already complete, ignoring message')
      return
    }

    const userMessage: Message = {
      id: makeMessageId(),
      role: 'user',
      content: messageText,
      timestamp: new Date(),
    }

    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setOtherChoiceActive(false)
    setIsLoading(true)

    try {
      // バックエンドのAI機能を活用
      const chatRequest: ChatRequest = {
        user_id: userId,
        session_id: sessionId,
        message: messageText,
        industry_id: 1, // IT業界
        job_category_id: 0, // 未設定（バックエンドで判定）
      }
      
      const response: ChatResponse = await sendChatMessage(chatRequest)
      
      const assistantMessage: Message = {
        id: makeMessageId(),
        role: 'assistant',
        content: response.response || 'エラーが発生しました',
        timestamp: new Date(),
      }
      
      // バリデーションエラーかどうかをチェック
      const isValidationError = response.response?.includes('書かれた内容にはお答えできません') || 
                                response.response?.includes('質問に回答してください') ||
                                response.response?.includes('質問と関係のない内容が3回続いた')
      
      // セッション終了チェック
      const isTerminated = response.is_terminated === true || 
                          response.response?.includes('チャットを終了させていただきます')
      
      setMessages((prev) => {
        const newMessages = [...prev, assistantMessage]
        
        // セッション終了の場合 - 専用モーダルを表示
        if (isTerminated) {
          console.log('[MUI Chat] Session terminated due to invalid answers')
          setAnalysisComplete(true)
          setShowTerminationModal(true)  // 終了専用モーダル
          return newMessages
        }
        
        // バリデーションエラーの場合は質問カウントを進めない
        if (!isValidationError) {
          // 質問カウントの更新
          const newCount = response.answered_questions ?? questionCount + 1
          setQuestionCount(newCount)
          const newTotalQuestions = response.total_questions ?? 15
          setTotalQuestions(newTotalQuestions)
          
          // totalQuestionsをsessionStorageに保存
          sessionStorage.setItem('totalQuestions', String(newTotalQuestions))
          if (response.all_phases) {
            sessionStorage.setItem('phaseProgress', JSON.stringify(response.all_phases))
            setPhaseProgresses(response.all_phases)
          }
          
          // 進捗状況を親コンポーネントに通知（非同期で実行）
          setTimeout(() => {
            window.dispatchEvent(new CustomEvent('chatProgress', { 
              detail: { 
                messageCount: newMessages.length,
                questionCount: newCount,
                totalQuestions: newTotalQuestions,
                phases: response.all_phases ?? null,
              } 
            }))
          }, 0)
          
          // **重要: バックエンドのis_completeのみを信頼**
          console.log('[MUI Chat] is_complete:', response.is_complete, 'type:', typeof response.is_complete)
          console.log('[MUI Chat] evaluated_categories:', response.evaluated_categories, 'total:', response.total_categories)
          
          const allCompleted = response.all_phases?.every((phase: any) => {
            const required = phase.max_questions > 0 ? phase.max_questions : phase.min_questions
            return required > 0 && phase.valid_answers >= required
          }) ?? false

          const completionText =
            response.response?.includes('分析が完了しました') ||
            response.response?.includes('最適な企業をマッチング')
          if (response.is_complete === true && !completionText) {
            const completionMessage: Message = {
              id: makeMessageId(),
              role: 'assistant',
              content: '分析が完了しました！あなたに最適な企業をマッチングしました。「結果を見る」ボタンから詳細をご確認ください。',
              timestamp: new Date(),
            }
            newMessages.push(completionMessage)
          }

          if (response.is_complete === true) {
            console.log('[MUI Chat] AI分析完了 - モーダルを表示します')
            console.log('[MUI Chat] All phases completed:', allCompleted)
            setTimeout(() => {
              setAnalysisComplete(true)
              setAllPhasesCompleted(allCompleted)
              setShowCompletionModal(true)
            }, 300)
          } else {
            console.log(`[MUI Chat] 質問継続中 (${newCount}/${response.total_questions ?? 15})`)
            // 明示的にfalseを設定
            setAnalysisComplete(false)
            setAllPhasesCompleted(false)
          }
        } else {
          // バリデーションエラーの場合は質問カウントを進めないが、完了状態はリセット
          console.log('[MUI Chat] Validation error detected, not updating question count')
          // バリデーションエラー後も質問を継続できるように、完了状態を解除
          setAnalysisComplete(false)
          setAllPhasesCompleted(false)
        }
        
        return newMessages
      })
    } catch (error) {
      console.error('[MUI Chat] Backend error:', error)
      
      // "all phases completed"エラーの場合は分析完了として扱う
      const errorMessage = (error as Error).message
      if (errorMessage.includes('all phases completed')) {
        console.log('[MUI Chat] All phases completed - showing completion modal')
        setAnalysisComplete(true)
        setAllPhasesCompleted(true)
        setShowCompletionModal(true)
        
        // 完了メッセージを表示
        const completionMessage: Message = {
          id: makeMessageId(),
          role: 'assistant',
          content: '分析が完了しました！あなたに最適な企業をマッチングしました。「結果を見る」ボタンから詳細をご確認ください。',
          timestamp: new Date(),
        }
        setMessages((prev) => [...prev, completionMessage])
      } else {
        // その他のエラー
        const errorMsg: Message = {
          id: makeMessageId(),
          role: 'assistant',
          content:
            'バックエンドとの接続に失敗しました。後ほど再試行してください。\n\nエラー: ' + errorMessage,
          timestamp: new Date(),
        }
        setMessages((prev) => [...prev, errorMsg])
      }
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
    
    // 進捗状況を親コンポーネントに通知（非同期で実行）
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent('chatProgress', { 
        detail: { messageCount: 1, questionCount: 0, totalQuestions: 15 } 
      }))
    }, 0)
  }

  const handleEndChat = () => {
    setShowEndChatModal(true)
  }

  const handleConfirmEndChat = () => {
    // セッションとキャッシュを完全にクリア
    sessionStorage.removeItem('chatSessionId')
    sessionStorage.removeItem('chatMessages')
    localStorage.removeItem('chatMessages')
    
    // チャット履歴のキャッシュも削除
    const currentSessionId = sessionStorage.getItem('chatSessionId')
    if (currentSessionId) {
      localStorage.removeItem(`chat_cache_${currentSessionId}`)
    }
    localStorage.removeItem('chat_session_id')
    
    // ページをリロードして新しいセッションを開始
    window.location.reload()
  }

  const handleCancelEndChat = () => {
    setShowEndChatModal(false)
  }

  const handleViewResults = () => {
    setShowCompletionModal(false)
    router.push(`/results?user_id=${userId}&session_id=${sessionId}`)
  }

  const handleContinueChat = () => {
    console.log('[MUI Chat] Continuing chat after completion')
    console.log('[MUI Chat] Before reset - analysisComplete:', analysisComplete)
    setShowCompletionModal(false)
    setAnalysisComplete(false)
    console.log('[MUI Chat] After reset - modal closed, analysisComplete set to false')
    // 入力フィールドを有効化するためにフォーカス
    setTimeout(() => {
      const inputElement = document.querySelector('input[type="text"]') as HTMLInputElement
      if (inputElement) {
        console.log('[MUI Chat] Input field found, focusing')
        inputElement.focus()
      } else {
        console.log('[MUI Chat] Input field not found')
      }
    }, 100)
  }

  const jobOptions = [
    '開発系エンジニア',
    'インフラエンジニア',
    '両方に興味がある',
    'まだ決めていない',
  ]
  const lastAssistantMessage = [...messages].reverse().find((msg) => msg.role === 'assistant')
  const choiceOptions = lastAssistantMessage ? extractChoices(lastAssistantMessage.content) : []
  const showChoiceButtons = choiceOptions.length >= 2 && !analysisComplete
  const inputPlaceholder = otherChoiceActive ? 'その他の内容を入力...' : 'メッセージを入力...'

  useEffect(() => {
    if (!showChoiceButtons) {
      setOtherChoiceActive(false)
    }
  }, [showChoiceButtons])

  if (!mounted) {
    return null
  }

  return (
    <>
      {/* 分析完了モーダル */}
      <Dialog
        open={showCompletionModal}
        onClose={allPhasesCompleted ? undefined : handleContinueChat}
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
            {allPhasesCompleted 
              ? 'すべての分析が完了しました！あなたに最適な企業をマッチングしました。'
              : 'あなたの適性を分析し、最適な企業をマッチングしました。'}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center' }}>
            結果ページで詳細な企業情報を確認できます。
          </Typography>
        </DialogContent>
        <DialogActions sx={{ justifyContent: 'center', gap: 2, pb: 2 }}>
          {!allPhasesCompleted && (
            <Button
              onClick={handleContinueChat}
              variant="outlined"
              size="large"
              sx={{ minWidth: 140 }}
            >
              チャットを続ける
            </Button>
          )}
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

      {/* チャット終了確認モーダル */}
      <Dialog
        open={showEndChatModal}
        onClose={handleCancelEndChat}
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
          <Typography variant="h5" component="div" sx={{ fontWeight: 'bold', color: 'warning.main' }}>
            ⚠️ チャットを終了しますか？
          </Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2 }}>
          <Typography variant="body1" sx={{ textAlign: 'center', mb: 2 }}>
            チャットを終了すると、現在の会話履歴が削除されます。
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center' }}>
            新しいセッションで最初からやり直すことになりますが、よろしいですか？
          </Typography>
        </DialogContent>
        <DialogActions sx={{ justifyContent: 'center', gap: 2, pb: 2 }}>
          <Button
            onClick={handleCancelEndChat}
            variant="outlined"
            size="large"
            sx={{ minWidth: 140 }}
          >
            キャンセル
          </Button>
          <Button
            onClick={handleConfirmEndChat}
            variant="contained"
            color="error"
            size="large"
            sx={{ minWidth: 140 }}
          >
            終了する
          </Button>
        </DialogActions>
      </Dialog>

      {/* 強制終了モーダル（3回の無効回答） */}
      <Dialog
        open={showTerminationModal}
        onClose={() => {}} // 閉じられないようにする
        disableEscapeKeyDown // Escキーでも閉じられない
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
          <Typography variant="h5" component="div" sx={{ fontWeight: 'bold', color: 'error.main' }}>
            ⚠️ チャットを終了します
          </Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 2, pb: 2 }}>
          <Typography variant="body1" sx={{ textAlign: 'center', mb: 2, color: 'error.main', fontWeight: 'bold' }}>
            質問と関係のない内容が3回続いたため、チャットを終了しました。
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', mb: 1 }}>
            新しいセッションで最初からやり直してください。
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center' }}>
            現在の回答内容は保存されていません。
          </Typography>
        </DialogContent>
        <DialogActions sx={{ justifyContent: 'center', pb: 2 }}>
          <Button
            onClick={() => {
              // 新しいセッションを開始（ページリロード）
              sessionStorage.removeItem('chatSessionId')
              localStorage.removeItem('currentSessionId')
              window.location.reload()
            }}
            variant="contained"
            color="error"
            size="large"
            sx={{ minWidth: 180 }}
            autoFocus={false}
            tabIndex={-1}
          >
            新しいセッションを開始
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
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Box>
          <Typography variant="h5" sx={{ fontWeight: 600 }}>
            IT業界キャリアエージェント
          </Typography>
          <Typography variant="body2" color="text.secondary">
            AI適性診断 - {(progressTotals?.valid ?? questionCount)}/{(progressTotals?.required ?? totalQuestions)} 問完了 
            {((progressTotals?.valid ?? questionCount) > 0) && ` (${progressTotals?.percent ?? Math.round((questionCount / totalQuestions) * 100)}%)`}
          </Typography>
        </Box>
        <Button
          variant="outlined"
          size="small"
          onClick={handleEndChat}
          sx={{ minWidth: '120px' }}
        >
          チャットを終了
        </Button>
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

        {messages.map((message) => {
          const isValidationError = message.role === 'assistant' && 
            (message.content.includes('書かれた内容にはお答えできません') || 
             message.content.includes('質問に回答してください') ||
             message.content.includes('質問と関係のない内容が3回続いた'))
          
          const isTerminationMessage = message.role === 'assistant' && 
            message.content.includes('チャットを終了させていただきます')
          
          return (
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
                  bgcolor: isTerminationMessage ? '#d32f2f' : (isValidationError ? '#f57c00' : '#1976d2'),
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
                  message.role === 'user' 
                    ? '#1976d2' 
                    : isTerminationMessage
                      ? '#ffebee'
                      : isValidationError 
                        ? '#fff3e0' 
                        : '#f5f5f5',
                color: message.role === 'user' ? '#fff' : '#000',
                border: isTerminationMessage ? '2px solid #d32f2f' : (isValidationError ? '2px solid #f57c00' : 'none'),
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
          )
        })}

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
              onClick={() => {
                console.log('[MUI Chat] Rendering completion button (analysisComplete=true)')
                setShowCompletionModal(true)
              }}
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
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {showChoiceButtons && (
              <Paper
                elevation={0}
                sx={{
                  p: 1.5,
                  borderRadius: 2,
                  border: '1px solid #e0e0e0',
                  backgroundColor: '#fafafa',
                }}
              >
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                  選択肢を選んでください
                </Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" gap={1}>
                  {choiceOptions.map((choice) => {
                    const isOtherChoice = choice.text.includes('その他')
                    return (
                      <Button
                        key={`${choice.label}-${choice.text}`}
                        variant="outlined"
                        onClick={() => {
                          if (isOtherChoice) {
                            setOtherChoiceActive(true)
                            setInput('')
                            setTimeout(() => inputRef.current?.focus(), 0)
                            return
                          }
                          handleSend(choice.value)
                        }}
                        disabled={isLoading}
                        sx={{ borderRadius: 2 }}
                      >
                        {choice.label}. {choice.text}
                      </Button>
                    )
                  })}
                </Stack>
            </Paper>
          )}
            <Box sx={{ display: 'flex', gap: 1 }}>
              <TextField
                fullWidth
                placeholder={inputPlaceholder}
                value={input}
                onChange={(e) => {
                  console.log('[MUI Chat] Rendering input field (analysisComplete=false)')
                  setInput(e.target.value)
                }}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault()
                    handleSend()
                  }
                }}
                disabled={isLoading}
                variant="outlined"
                size="small"
                inputRef={inputRef}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    borderRadius: 2,
                  },
                }}
              />
              <IconButton
                color="primary"
                onClick={() => handleSend()}
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
        )}
      </Box>
      </Box>
    </>
  )
}
