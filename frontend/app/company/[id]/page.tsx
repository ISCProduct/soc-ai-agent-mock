"use client"

import { useParams, useRouter } from "next/navigation"
import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { 
  ArrowLeft, Building2, MapPin, Users, Calendar, Globe, 
  TrendingUp, Award, Code, Briefcase, Heart, ExternalLink,
  DollarSign, Star, Clock, Target
} from "lucide-react"

type Company = {
  id: number
  name: string
  industry: string
  location: string
  employees: string
  description: string
  matchScore: number
  tags: string[]
  techStack: string[]
  projectTypes: string[]
  salary: string
  benefits: string[]
  culture: string[]
  founded: string
  website: string
  size: string
  parentCompany?: string
  subsidiaries?: string[]
  partnerships?: string[]
  capitalStructure?: {
    shareholders: { name: string; percentage: number }[]
  }
}

export default function CompanyDetailPage() {
  const params = useParams()
  const router = useRouter()
  const [company, setCompany] = useState<Company | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchCompanyDetail = async () => {
      try {
        // バックエンドから企業詳細を取得
        const response = await fetch(`/api/companies/${params.id}`)
        
        if (response.ok) {
          const data = await response.json()
          setCompany(data)
        } else {
          console.error('Failed to fetch company details:', response.statusText)
          setCompany(null)
        }
      } catch (error) {
        console.error('Failed to fetch company details:', error)
        setCompany(null)
      } finally {
        setLoading(false)
      }
    }

    fetchCompanyDetail()
  }, [params.id])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">読み込み中...</p>
        </div>
      </div>
    )
  }

  if (!company) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold mb-4">企業が見つかりません</h1>
          <Button onClick={() => router.back()}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            戻る
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      {/* ヘッダー */}
      <div className="border-b bg-muted/50">
        <div className="container mx-auto px-4 py-4">
          <Button variant="ghost" onClick={() => router.back()}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            一覧に戻る
          </Button>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8 max-w-5xl">
        {/* 企業ヘッダー */}
        <Card className="mb-6">
          <CardHeader>
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <Building2 className="w-8 h-8 text-primary" />
                  <CardTitle className="text-3xl">{company.name}</CardTitle>
                </div>
                <p className="text-muted-foreground">{company.industry}</p>
              </div>
              <div className="text-right">
                <div className="flex items-center gap-2 mb-2">
                  <Star className="w-5 h-5 text-yellow-500 fill-yellow-500" />
                  <span className="text-2xl font-bold text-primary">{company.matchScore}%</span>
                </div>
                <p className="text-sm text-muted-foreground">マッチ度</p>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
              <div className="flex items-center gap-2">
                <MapPin className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm">{company.location}</span>
              </div>
              <div className="flex items-center gap-2">
                <Users className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm">{company.employees}</span>
              </div>
              <div className="flex items-center gap-2">
                <Calendar className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm">設立: {company.founded}</span>
              </div>
            </div>

            {/* タグ */}
            <div className="flex flex-wrap gap-2 mb-4">
              {company.tags.map((tag, index) => (
                <Badge key={index} variant="secondary">{tag}</Badge>
              ))}
            </div>

            {/* ウェブサイト */}
            <Button variant="outline" asChild>
              <a href={company.website} target="_blank" rel="noopener noreferrer">
                <Globe className="w-4 h-4 mr-2" />
                公式サイトを見る
                <ExternalLink className="w-3 h-3 ml-2" />
              </a>
            </Button>
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* 企業概要 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Building2 className="w-5 h-5" />
                企業概要
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm leading-relaxed">{company.description}</p>
            </CardContent>
          </Card>

          {/* 技術スタック */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Code className="w-5 h-5" />
                技術スタック
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {company.techStack.map((tech, index) => (
                  <Badge key={index} variant="outline">{tech}</Badge>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* プロジェクトタイプ */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Target className="w-5 h-5" />
                プロジェクトタイプ
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2">
                {company.projectTypes.map((type, index) => (
                  <li key={index} className="flex items-center gap-2 text-sm">
                    <div className="w-1.5 h-1.5 rounded-full bg-primary" />
                    {type}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>

          {/* 給与・待遇 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <DollarSign className="w-5 h-5" />
                給与・待遇
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-lg font-semibold mb-3">{company.salary}</p>
              <div className="space-y-1">
                {company.benefits.map((benefit, index) => (
                  <div key={index} className="flex items-center gap-2 text-sm">
                    <Heart className="w-3 h-3 text-primary" />
                    {benefit}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* 企業文化 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Award className="w-5 h-5" />
                企業文化・働き方
              </CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2">
                {company.culture.map((item, index) => (
                  <li key={index} className="flex items-center gap-2 text-sm">
                    <div className="w-1.5 h-1.5 rounded-full bg-green-500" />
                    {item}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>

          {/* 企業規模 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="w-5 h-5" />
                企業規模・関連情報
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-sm font-semibold mb-1">企業規模</p>
                <p className="text-sm text-muted-foreground">{company.size}</p>
              </div>
              
              {company.parentCompany && (
                <div>
                  <p className="text-sm font-semibold mb-1">親会社</p>
                  <p className="text-sm text-muted-foreground">{company.parentCompany}</p>
                </div>
              )}

              {company.subsidiaries && company.subsidiaries.length > 0 && (
                <div>
                  <p className="text-sm font-semibold mb-1">子会社</p>
                  <ul className="text-sm text-muted-foreground space-y-1">
                    {company.subsidiaries.map((sub, index) => (
                      <li key={index}>• {sub}</li>
                    ))}
                  </ul>
                </div>
              )}

              {company.partnerships && company.partnerships.length > 0 && (
                <div>
                  <p className="text-sm font-semibold mb-1">主要パートナー</p>
                  <ul className="text-sm text-muted-foreground space-y-1">
                    {company.partnerships.map((partner, index) => (
                      <li key={index}>• {partner}</li>
                    ))}
                  </ul>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* アクションボタン */}
        <div className="mt-8 flex gap-4 justify-center">
          <Button size="lg" onClick={() => router.back()}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            一覧に戻る
          </Button>
          <Button size="lg" variant="outline" asChild>
            <a href={company.website} target="_blank" rel="noopener noreferrer">
              <Globe className="w-4 h-4 mr-2" />
              公式サイトへ
              <ExternalLink className="w-3 h-3 ml-2" />
            </a>
          </Button>
        </div>
      </div>
    </div>
  )
}
