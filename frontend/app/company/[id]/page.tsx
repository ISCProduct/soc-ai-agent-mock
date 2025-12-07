"use client"

import { useParams, useRouter } from "next/navigation"
import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { 
  ArrowLeft, Building2, MapPin, Users, Calendar, Globe, 
  TrendingUp, Award, Code, Briefcase, Heart, ExternalLink,
  DollarSign, Star, Clock, Target, GitBranch, Network
} from "lucide-react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import CompanyDiagram from "@/components/company-diagram"

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

// 企業名からIDにマッピング（暫定的に企業ID 1-3 を使用）
function getMockCompanyId(companyName: string): number {
  // 実際の企業データから取得する場合はAPIを使用
  // ここでは既存の3社のいずれかを返す
  const nameMap: Record<string, number> = {
    '株式会社テクノシステム': 1,
    '日本ソフトウェア株式会社': 2,
    '株式会社クラウドワークス': 3,
  };
  
  return nameMap[companyName] || 1; // デフォルトは企業ID 1
}

export default function CompanyDetailPage() {
  const params = useParams()
  const router = useRouter()
  const [company, setCompany] = useState<Company | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('capital')

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

        {/* 企業関連図 */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Network className="w-5 h-5" />
              企業関連図
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
              <TabsList className="grid w-full max-w-md grid-cols-2">
                <TabsTrigger value="capital" className="flex items-center gap-2">
                  <GitBranch className="w-4 h-4" />
                  資本関連図
                </TabsTrigger>
                <TabsTrigger value="business" className="flex items-center gap-2">
                  <Network className="w-4 h-4" />
                  ビジネス関連図
                </TabsTrigger>
              </TabsList>
              
              <TabsContent value="capital" className="mt-4">
                <div className="mb-3 text-sm text-muted-foreground space-y-1">
                  <p>• 黄色でハイライトされた企業が現在表示中の企業です</p>
                  <p>• 実線：子会社（出資比率50%以上）、破線：関連会社（出資比率50%未満）</p>
                  <div className="flex gap-4 mt-2">
                    <span className="flex items-center gap-2">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: '#4169E1' }}></span>
                      プライム
                    </span>
                    <span className="flex items-center gap-2">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: '#32CD32' }}></span>
                      スタンダード
                    </span>
                    <span className="flex items-center gap-2">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: '#FF6347' }}></span>
                      グロース
                    </span>
                    <span className="flex items-center gap-2">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: '#9E9E9E' }}></span>
                      非上場
                    </span>
                  </div>
                </div>
                <CompanyDiagram 
                  companyId={getMockCompanyId(company.name)} 
                  diagramType="capital" 
                />
              </TabsContent>
              
              <TabsContent value="business" className="mt-4">
                <div className="mb-3 text-sm text-muted-foreground space-y-1">
                  <p>• 黄色でハイライトされた企業が現在表示中の企業です</p>
                  <p>• 青い矢印：ビジネス取引関係、灰色の点線：資本関係（親会社）</p>
                </div>
                <CompanyDiagram 
                  companyId={getMockCompanyId(company.name)} 
                  diagramType="business" 
                />
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

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
