"use client"

import { useState, useRef, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Send, Bot, User, CheckCircle2, Circle } from "lucide-react"
import { CompanyResults } from "@/components/company-results"
import { AnalysisLoading } from "@/components/analysis-loading"
import { sendChatMessage, getChatHistory, type ChatResponse } from "@/lib/api"

type Message = {
  id: string
  role: "agent" | "user"
  content: string
  options?: string[]
}

type UserData = {
  jobType?: string
  qualifications?: string
  programmingConfidence?: string
  programmingLanguages?: string
  interestField?: string
  projectType?: string
  salaryExpectation?: string
  workStyle?: string
  careerGoal?: string
  companySize?: string
}

type FlowStep = {
  message: string
  hint: string
  options?: string[]
  next: string
}

const conversationFlow: Record<string, FlowStep> = {
  jobType: {
    message:
        "こんにちは！IT業界専門のキャリアエージェントです。4万社余りのIT企業の中から、あなたに最適な企業を選定いたします。\n\nまず、どのような職種を希望されますか？",
    hint: "IT業界には様々な職種があります。開発系、インフラ系、それとも両方に興味がありますか？",
    options: ["開発系エンジニア", "インフラエンジニア", "両方に興味がある", "まだ決めていない"],
    next: "qualifications",
  },
  qualifications: {
    message: "{previous}ですね。では、現在までに合格した資格を教えてください。",
    hint: "ITパスポート、基本情報技術者試験、応用情報技術者試験など、お持ちの資格をお聞かせください。",
    next: "programmingConfidence",
  },
  programmingConfidence: {
    message: "ありがとうございます。プログラミングは得意ですか？",
    hint: "正直にお答えください。あなたのレベルに合った企業をご紹介します。",
    options: ["とても得意です", "ある程度できます", "あまり自信がありません", "これから学びたい"],
    next: "programmingLanguages",
  },
  programmingLanguages: {
    message: "承知しました。今までに学習したプログラミング言語を教えてください。",
    hint: "Java、Python、JavaScript、C++など、学習経験のある言語をお聞かせください。",
    next: "interestField",
  },
  interestField: {
    message: "ありがとうございます。ここからは興味分析に移ります。IT業界のどの分野に最も興味がありますか？",
    hint: "Web開発、AI・機械学習、クラウド、セキュリティなど、様々な分野があります。",
    options: ["Web・アプリ開発", "AI・機械学習", "クラウド・インフラ", "セキュリティ", "データ分析", "その他"],
    next: "projectType",
  },
  projectType: {
    message: "{previous}の分野ですね。どのようなプロジェクトに携わりたいですか？",
    hint: "自社サービス開発、受託開発、社内システムなど、働き方によって環境が大きく変わります。",
    options: ["自社サービス開発", "受託開発・SES", "社内システム開発", "研究開発", "まだ決めていない"],
    next: "salaryExpectation",
  },
  salaryExpectation: {
    message: "なるほど。それでは待遇面についてお伺いします。初年度の年収について、どのくらいを希望されますか？",
    hint: "IT業界の新卒平均は300-400万円程度ですが、企業によって大きく異なります。",
    options: ["300万円以上", "400万円以上", "500万円以上", "特にこだわらない"],
    next: "workStyle",
  },
  workStyle: {
    message: "承知しました。働き方について、最も重視することは何ですか？",
    hint: "リモートワーク、フレックス、残業の少なさなど、ワークライフバランスに関わる要素です。",
    options: [
      "リモートワーク可能",
      "フレックスタイム制",
      "残業が少ない",
      "オフィス勤務でチーム重視",
      "特にこだわらない",
    ],
    next: "careerGoal",
  },
  careerGoal: {
    message: "最後に、将来のキャリアについてお伺いします。5年後、どのような姿を目指していますか？",
    hint: "スペシャリスト、マネージャー、起業など、様々なキャリアパスがあります。",
    options: [
      "技術のスペシャリスト",
      "プロジェクトマネージャー",
      "テックリード・アーキテクト",
      "起業・フリーランス",
      "まだ考えていない",
    ],
    next: "companySize",
  },
  companySize: {
    message: "素晴らしい目標ですね。最後に、どのような規模の企業で働きたいですか？",
    hint: "大企業は安定性、ベンチャーは成長性が魅力です。それぞれに良さがあります。",
    options: ["大手企業（1000名以上）", "中堅企業（100-1000名）", "ベンチャー企業（100名未満）", "特にこだわらない"],
    next: "complete",
  },
}

const analysisPhases = [
  { id: 1, name: "職種分析", steps: ["jobType", "qualifications", "programmingConfidence", "programmingLanguages"] },
  { id: 2, name: "興味分析", steps: ["interestField", "projectType"] },
  { id: 3, name: "待遇分析", steps: ["salaryExpectation", "workStyle"] },
  { id: 4, name: "将来分析", steps: ["careerGoal", "companySize"] },
]

export function JobAgentChat() {
  const [sessionId] = useState(() => {
    // セッションIDをlocalStorageから取得または新規作成
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('chat_session_id')
      if (stored) {
        return stored
      }
    }
    const newId = `session_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`
    if (typeof window !== 'undefined') {
      localStorage.setItem('chat_session_id', newId)
    }
    return newId
  })
  const [userId] = useState(1)
  const [industryId] = useState(1)
  const [jobCategoryId] = useState(1)
  const [useBackend, setUseBackend] = useState(true)
  const [isLoadingFromBackend, setIsLoadingFromBackend] = useState(false)

  const [messages, setMessages] = useState<Message[]>(() => {
    // localStorageからメッセージ履歴を復元
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('chat_messages')
      if (stored) {
        try {
          return JSON.parse(stored)
        } catch (e) {
          console.error('Failed to parse stored messages:', e)
        }
      }
    }
    return [
      {
        id: "1",
        role: "agent",
        content: conversationFlow.jobType.message,
        options: conversationFlow.jobType.options,
      },
    ]
  })
  const [inputValue, setInputValue] = useState("")
  const [currentStep, setCurrentStep] = useState<string>("jobType")
  const [userData, setUserData] = useState<UserData>({})
  const [isComplete, setIsComplete] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('chat_is_complete') === 'true'
    }
    return false
  })
  const [isAnalyzing, setIsAnalyzing] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('chat_is_analyzing') === 'true'
    }
    return false
  })
  const [isTyping, setIsTyping] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }

  useEffect(() => {
    scrollToBottom()
    // メッセージが更新されたらlocalStorageに保存
    if (typeof window !== 'undefined' && messages.length > 0) {
      localStorage.setItem('chat_messages', JSON.stringify(messages))
    }
  }, [messages, isTyping])

  useEffect(() => {
    if (useBackend) {
      loadChatHistory()
    }
  }, [])

  const loadChatHistory = async () => {
    try {
      const history = await getChatHistory(sessionId)
      if (history.length > 0) {
        const loadedMessages: Message[] = history.map((msg) => ({
          id: msg.id.toString(),
          role: msg.role === "assistant" ? "agent" : "user",
          content: msg.content,
        }))
        setMessages(loadedMessages)
      }
    } catch (error) {
      console.error("[v0] Failed to load chat history:", error)
    }
  }

  const addAgentMessage = (content: string, hint?: string, options?: string[]) => {
    setIsTyping(true)
    setTimeout(() => {
      const fullContent = hint ? `${content}\n\n💡 ${hint}` : content
      setMessages((prev) => [
        ...prev,
        {
          id: Date.now().toString(),
          role: "agent",
          content: fullContent,
          options,
        },
      ])
      setIsTyping(false)
    }, 800)
  }

  const handleSend = async (message?: string) => {
    const textToSend = message || inputValue.trim()
    if (!textToSend) return

    setMessages((prev) => [
      ...prev,
      {
        id: Date.now().toString(),
        role: "user",
        content: textToSend,
      },
    ])
    setInputValue("")

    if (useBackend) {
      setIsLoadingFromBackend(true)
      setIsTyping(true)

      try {
        console.log("[v0] Sending message to backend:", { sessionId, userId, message: textToSend })

        const response: ChatResponse = await sendChatMessage({
          user_id: userId,
          session_id: sessionId,
          message: textToSend,
          industry_id: industryId,
          job_category_id: jobCategoryId,
        })

        console.log("[v0] Received response from backend:", response)

        setTimeout(() => {
          setMessages((prev) => [
            ...prev,
            {
              id: Date.now().toString(),
              role: "agent",
              content: response.response,
            },
          ])
          setIsTyping(false)
          setIsLoadingFromBackend(false)

          // 質問終了判定
          if (response.is_complete) {
            console.log("[v0] Analysis complete. Starting loading phase...")
            setTimeout(() => {
              setIsAnalyzing(true)
              if (typeof window !== 'undefined') {
                localStorage.setItem('chat_is_analyzing', 'true')
              }
            }, 1000)
          }
        }, 500)

        if (response.current_scores && response.current_scores.length > 0) {
          console.log("[v0] Current scores:", response.current_scores)
        }
      } catch (error) {
        console.error("[v0] Backend error:", error)
        setIsTyping(false)
        setIsLoadingFromBackend(false)

        setMessages((prev) => [
          ...prev,
          {
            id: Date.now().toString(),
            role: "agent",
            content: "申し訳ございません。接続エラーが発生しました。もう一度お試しください。",
          },
        ])
      }

      return
    }

    if (currentStep !== "complete") {
      setUserData((prev) => ({ ...prev, [currentStep]: textToSend }))

      const currentFlow = conversationFlow[currentStep]
      const nextStepKey = currentFlow.next

      if (nextStepKey === "complete") {
        setIsTyping(true)
        setTimeout(() => {
          setMessages((prev) => [
            ...prev,
            {
              id: Date.now().toString(),
              role: "agent",
              content: `ありがとうございました！\n\n4段階の分析が完了しました。あなたに適した企業を10社に絞り込んでいます...\n\n📊 職種分析 ✓\n🎯 興味分析 ✓\n💰 待遇分析 ✓\n🚀 将来分析 ✓`,
            },
          ])
          setIsTyping(false)
          setTimeout(() => {
            setIsComplete(true)
          }, 2000)
        }, 800)
      } else {
        const nextFlow = conversationFlow[nextStepKey]
        const messageWithContext = nextFlow.message.replace("{previous}", textToSend)
        addAgentMessage(messageWithContext, nextFlow.hint, nextFlow.options)
        setCurrentStep(nextStepKey)
      }
    }
  }

  const handleReset = () => {
    // localStorageをクリア
    if (typeof window !== 'undefined') {
      localStorage.removeItem('chat_session_id')
      localStorage.removeItem('chat_messages')
      localStorage.removeItem('chat_user_data')
      localStorage.removeItem('chat_is_complete')
      localStorage.removeItem('chat_is_analyzing')
    }
    
    // 新しいセッションID生成
    const newSessionId = `session_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`
    if (typeof window !== 'undefined') {
      localStorage.setItem('chat_session_id', newSessionId)
    }
    
    setMessages([
      {
        id: "1",
        role: "agent",
        content: conversationFlow.jobType.message,
        options: conversationFlow.jobType.options,
      },
    ])
    setCurrentStep("jobType")
    setUserData({})
    setIsComplete(false)
    setIsAnalyzing(false)
    setInputValue("")
    
    // ページをリロード
    window.location.reload()
  }

  const handleAnalysisComplete = () => {
    setIsAnalyzing(false)
    setIsComplete(true)
    if (typeof window !== 'undefined') {
      localStorage.setItem('chat_is_analyzing', 'false')
      localStorage.setItem('chat_is_complete', 'true')
    }
  }

  const getCurrentPhase = () => {
    for (let i = 0; i < analysisPhases.length; i++) {
      if (analysisPhases[i].steps.includes(currentStep)) {
        return i + 1
      }
    }
    return 1
  }

  const isPhaseCompleted = (phaseId: number) => {
    const currentPhaseId = getCurrentPhase()
    return phaseId < currentPhaseId
  }

  if (isAnalyzing) {
    return <AnalysisLoading onComplete={handleAnalysisComplete} />
  }

  if (isComplete) {
    return <CompanyResults userData={userData} onReset={handleReset} />
  }

  return (
      <div className="flex gap-4 h-[600px]">
        <Card className="w-64 border-2 p-6 flex flex-col gap-6">
          <div className="space-y-1">
            <h3 className="font-bold text-lg text-foreground">分析進捗</h3>
            <p className="text-xs text-muted-foreground">4段階の分析を実施中</p>
          </div>

          <div className="flex items-center gap-2 p-2 bg-muted rounded-lg">
            <input
                type="checkbox"
                id="backend-toggle"
                checked={useBackend}
                onChange={(e) => setUseBackend(e.target.checked)}
                className="w-4 h-4"
            />
            <label htmlFor="backend-toggle" className="text-xs text-muted-foreground cursor-pointer">
              バックエンド連携
            </label>
          </div>

          <div className="flex flex-col gap-4 flex-1">
            {analysisPhases.map((phase, index) => (
                <div key={phase.id} className="flex flex-col gap-2">
                  <div className="flex items-center gap-3">
                    {isPhaseCompleted(phase.id) ? (
                        <CheckCircle2 className="w-6 h-6 text-primary flex-shrink-0" />
                    ) : getCurrentPhase() === phase.id ? (
                        <Circle className="w-6 h-6 text-primary fill-primary flex-shrink-0" />
                    ) : (
                        <Circle className="w-6 h-6 text-muted-foreground flex-shrink-0" />
                    )}
                    <div className="flex flex-col">
                  <span
                      className={`text-sm font-semibold ${
                          getCurrentPhase() === phase.id
                              ? "text-primary"
                              : isPhaseCompleted(phase.id)
                                  ? "text-foreground"
                                  : "text-muted-foreground"
                      }`}
                  >
                    {phase.name}
                  </span>
                      <span className="text-xs text-muted-foreground">
                    {isPhaseCompleted(phase.id) ? "完了" : getCurrentPhase() === phase.id ? "進行中" : "待機中"}
                  </span>
                    </div>
                  </div>
                  {index < analysisPhases.length - 1 && (
                      <div
                          className={`w-0.5 h-8 ml-3 ${isPhaseCompleted(phase.id + 1) ? "bg-primary" : "bg-muted-foreground/30"}`}
                      />
                  )}
                </div>
            ))}
          </div>
        </Card>

        <Card className="flex flex-col flex-1 border-2">
          <div className="border-b bg-muted/50 p-4">
            <div className="flex items-center gap-3">
              <Avatar className="w-10 h-10 bg-primary">
                <AvatarFallback>
                  <Bot className="w-5 h-5 text-primary-foreground" />
                </AvatarFallback>
              </Avatar>
              <div>
                <h2 className="font-bold text-foreground">IT業界キャリアエージェント</h2>
                <p className="text-xs text-muted-foreground">
                  4万社から最適な企業を選定 {useBackend && "(バックエンド連携中)"}
                </p>
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto p-6 space-y-4">
            {messages.map((message) => (
                <div key={message.id} className={`flex gap-3 ${message.role === "user" ? "flex-row-reverse" : "flex-row"}`}>
                  <Avatar className={`w-10 h-10 ${message.role === "agent" ? "bg-primary" : "bg-accent"}`}>
                    <AvatarFallback>
                      {message.role === "agent" ? (
                          <Bot className="w-5 h-5 text-primary-foreground" />
                      ) : (
                          <User className="w-5 h-5 text-accent-foreground" />
                      )}
                    </AvatarFallback>
                  </Avatar>
                  <div
                      className={`flex flex-col gap-2 max-w-[80%] ${message.role === "user" ? "items-end" : "items-start"}`}
                  >
                    <div
                        className={`rounded-2xl px-4 py-3 ${
                            message.role === "agent" ? "bg-muted text-foreground" : "bg-primary text-primary-foreground"
                        }`}
                    >
                      <p className="text-sm leading-relaxed whitespace-pre-line">{message.content}</p>
                    </div>
                    {message.options && (
                        <div className="flex flex-wrap gap-2 mt-1">
                          {message.options.map((option, index) => (
                              <Button
                                  key={index}
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleSend(option)}
                                  className="text-xs hover:bg-primary hover:text-primary-foreground transition-colors"
                              >
                                {option}
                              </Button>
                          ))}
                        </div>
                    )}
                  </div>
                </div>
            ))}
            {isTyping && (
                <div className="flex gap-3">
                  <Avatar className="w-10 h-10 bg-primary">
                    <AvatarFallback>
                      <Bot className="w-5 h-5 text-primary-foreground" />
                    </AvatarFallback>
                  </Avatar>
                  <div className="bg-muted rounded-2xl px-4 py-3">
                    <div className="flex gap-1">
                  <span
                      className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce"
                      style={{ animationDelay: "0ms" }}
                  />
                      <span
                          className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce"
                          style={{ animationDelay: "150ms" }}
                      />
                      <span
                          className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce"
                          style={{ animationDelay: "300ms" }}
                      />
                    </div>
                  </div>
                </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          <div className="border-t bg-muted/30 p-4">
            <form
                onSubmit={(e) => {
                  e.preventDefault()
                  handleSend()
                }}
                className="flex gap-2"
            >
              <Input
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  placeholder="メッセージを入力..."
                  className="flex-1 bg-background"
                  disabled={isTyping || isLoadingFromBackend}
              />
              <Button type="submit" size="icon" disabled={!inputValue.trim() || isTyping || isLoadingFromBackend}>
                <Send className="w-4 h-4" />
              </Button>
            </form>
          </div>
        </Card>
      </div>
  )
}
