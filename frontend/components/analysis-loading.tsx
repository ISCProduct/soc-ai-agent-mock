"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { CheckCircle2, Loader2 } from "lucide-react"

type AnalysisLoadingProps = {
  onComplete: () => void
}

const analysisSteps = [
  { id: 1, name: "回答データの集計", duration: 1000 },
  { id: 2, name: "適性スコアの算出", duration: 1500 },
  { id: 3, name: "企業データベースの検索", duration: 2000 },
  { id: 4, name: "マッチング分析", duration: 1500 },
  { id: 5, name: "最適企業の選定", duration: 1000 },
]

export function AnalysisLoading({ onComplete }: AnalysisLoadingProps) {
  const [currentStep, setCurrentStep] = useState(0)
  const [completedSteps, setCompletedSteps] = useState<number[]>([])

  useEffect(() => {
    if (currentStep >= analysisSteps.length) {
      setTimeout(() => {
        onComplete()
      }, 500)
      return
    }

    const step = analysisSteps[currentStep]
    const timer = setTimeout(() => {
      setCompletedSteps((prev) => [...prev, step.id])
      setCurrentStep((prev) => prev + 1)
    }, step.duration)

    return () => clearTimeout(timer)
  }, [currentStep, onComplete])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background to-muted/20 p-4">
      <Card className="w-full max-w-2xl p-8 space-y-6">
        <div className="text-center space-y-2">
          <h2 className="text-3xl font-bold text-foreground">適性分析完了</h2>
          <p className="text-muted-foreground">
            あなたに最適な企業を選定しています...
          </p>
        </div>

        <div className="space-y-4">
          {analysisSteps.map((step, index) => {
            const isCompleted = completedSteps.includes(step.id)
            const isCurrent = currentStep === index
            const isPending = currentStep < index

            return (
              <div
                key={step.id}
                className={`flex items-center gap-4 p-4 rounded-lg transition-all duration-300 ${
                  isCurrent
                    ? "bg-primary/10 border-2 border-primary"
                    : isCompleted
                    ? "bg-muted/50"
                    : "bg-muted/20"
                }`}
              >
                <div className="flex-shrink-0">
                  {isCompleted ? (
                    <CheckCircle2 className="w-6 h-6 text-primary" />
                  ) : isCurrent ? (
                    <Loader2 className="w-6 h-6 text-primary animate-spin" />
                  ) : (
                    <div className="w-6 h-6 rounded-full border-2 border-muted-foreground/30" />
                  )}
                </div>

                <div className="flex-1">
                  <p
                    className={`font-medium ${
                      isCurrent
                        ? "text-primary"
                        : isCompleted
                        ? "text-foreground"
                        : "text-muted-foreground"
                    }`}
                  >
                    {step.name}
                  </p>
                </div>

                {isCurrent && (
                  <div className="flex-shrink-0">
                    <div className="text-xs text-primary font-medium">処理中...</div>
                  </div>
                )}

                {isCompleted && (
                  <div className="flex-shrink-0">
                    <div className="text-xs text-muted-foreground">完了</div>
                  </div>
                )}
              </div>
            )
          })}
        </div>

        <div className="space-y-2">
          <div className="flex justify-between text-sm text-muted-foreground">
            <span>進捗状況</span>
            <span>
              {completedSteps.length} / {analysisSteps.length}
            </span>
          </div>
          <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
            <div
              className="bg-primary h-full transition-all duration-500 ease-out"
              style={{
                width: `${(completedSteps.length / analysisSteps.length) * 100}%`,
              }}
            />
          </div>
        </div>
      </Card>
    </div>
  )
}
