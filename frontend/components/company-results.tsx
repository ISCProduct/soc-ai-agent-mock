"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { RotateCcw, Building2, MapPin, Users, TrendingUp, Award, Code, Briefcase, Target, Eye } from "lucide-react"
import { CompanyDetailModal } from "@/components/company-detail-modal"

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

const generateITCompanies = (userData: UserData): Company[] => {
  const companies: Company[] = [
    {
      id: 1,
      name: "株式会社テックイノベーション",
      industry: "Webサービス・AI開発",
      location: "東京都渋谷区",
      employees: "150名",
      size: "ベンチャー企業（100-300名）",
      description: "自社AIプロダクトを開発するベンチャー企業。最新技術を活用した開発環境で急成長中。機械学習エンジニアとして最前線で活躍できる環境を提供しています。",
      matchScore: 0,
      tags: ["リモートワーク", "フレックス", "技術力重視"],
      techStack: ["Python", "TypeScript", "React", "AWS"],
      projectTypes: ["自社サービス", "AI開発"],
      salary: "400-650万円",
      benefits: ["リモートワーク可", "フレックスタイム", "書籍購入補助", "資格取得支援", "最新機器貸与"],
      culture: ["技術第一", "フラットな組織", "挑戦を推奨", "成果主義"],
      founded: "2018年",
      website: "https://tech-innovation.example.com",
      parentCompany: undefined,
      subsidiaries: ["TI Lab株式会社", "TI Consulting"],
      partnerships: ["Google Cloud", "AWS", "Microsoft Azure"],
      capitalStructure: {
        shareholders: [
          { name: "創業者グループ", percentage: 45 },
          { name: "ベンチャーキャピタルA", percentage: 30 },
          { name: "事業会社B", percentage: 15 },
          { name: "従業員持株会", percentage: 10 },
        ],
      },
    },
    {
      id: 2,
      name: "日本システムソリューションズ株式会社",
      industry: "SIer・受託開発",
      location: "東京都千代田区",
      employees: "2500名",
      size: "大手企業（1000名以上）",
      description: "大手企業向けシステム開発を手がける老舗SIer。充実した研修制度と安定した環境で、エンジニアとしての基礎をしっかり学べます。",
      matchScore: 0,
      tags: ["大手企業", "研修充実", "福利厚生"],
      techStack: ["Java", "Oracle", "Spring"],
      projectTypes: ["受託開発", "社内システム"],
      salary: "350-550万円",
      benefits: ["住宅手当", "家族手当", "退職金制度", "社員食堂", "保養所"],
      culture: ["安定重視", "チームワーク", "長期キャリア形成", "教育重視"],
      founded: "1985年",
      website: "https://nss.example.co.jp",
      parentCompany: "日本テクノロジーグループ",
      subsidiaries: ["NSS北海道", "NSS関西", "NSS九州"],
      partnerships: ["Oracle", "SAP", "IBM"],
      capitalStructure: {
        shareholders: [
          { name: "日本テクノロジーグループ", percentage: 60 },
          { name: "機関投資家", percentage: 25 },
          { name: "自社株", percentage: 10 },
          { name: "その他", percentage: 5 },
        ],
      },
    },
    {
      id: 3,
      name: "クラウドテック株式会社",
      industry: "クラウド・インフラ",
      location: "東京都港区",
      employees: "300名",
      description: "クラウドインフラの設計・構築を専門とする企業。AWS/Azure/GCPの認定資格取得支援あり。",
      matchScore: 0,
      tags: ["インフラ特化", "資格支援", "技術研修"],
      techStack: ["AWS", "Kubernetes", "Terraform"],
      projectTypes: ["インフラ構築", "クラウド移行"],
    },
    {
      id: 4,
      name: "株式会社セキュアネット",
      industry: "セキュリティ・コンサルティング",
      location: "東京都新宿区",
      employees: "180名",
      description: "サイバーセキュリティのスペシャリスト集団。高度な技術力と専門性を磨ける環境。",
      matchScore: 0,
      tags: ["セキュリティ", "高給与", "専門性"],
      techStack: ["Python", "Linux", "ネットワーク"],
      projectTypes: ["セキュリティ診断", "コンサルティング"],
    },
    {
      id: 5,
      name: "データアナリティクス株式会社",
      industry: "データ分析・BI",
      location: "東京都品川区",
      employees: "120名",
      description: "ビッグデータ分析とBIツール開発を行う企業。データサイエンティストとして成長できる。",
      matchScore: 0,
      tags: ["データ分析", "成長企業", "リモート可"],
      techStack: ["Python", "SQL", "Tableau", "Spark"],
      projectTypes: ["データ分析", "BI開発"],
    },
    {
      id: 6,
      name: "モバイルアプリ開発株式会社",
      industry: "モバイルアプリ開発",
      location: "東京都渋谷区",
      employees: "80名",
      description: "iOS/Androidアプリの受託開発を行うベンチャー。自由な社風とチーム開発重視。",
      matchScore: 0,
      tags: ["モバイル", "ベンチャー", "チーム開発"],
      techStack: ["Swift", "Kotlin", "Flutter"],
      projectTypes: ["アプリ開発", "受託開発"],
    },
    {
      id: 7,
      name: "エンタープライズシステムズ株式会社",
      industry: "社内システム開発",
      location: "神奈川県横浜市",
      employees: "500名",
      description: "大手メーカーのグループ会社。安定した環境で社内システムの開発・運用を担当。",
      matchScore: 0,
      tags: ["安定性", "ワークライフバランス", "福利厚生"],
      techStack: ["Java", "C#", ".NET"],
      projectTypes: ["社内システム", "業務システム"],
    },
    {
      id: 8,
      name: "株式会社ゲームテクノロジー",
      industry: "ゲーム開発",
      location: "東京都中野区",
      employees: "200名",
      description: "スマホゲームの開発・運営を行う企業。エンタメ×技術で楽しく働ける環境。",
      matchScore: 0,
      tags: ["ゲーム", "クリエイティブ", "自社サービス"],
      techStack: ["Unity", "C#", "Go"],
      projectTypes: ["ゲーム開発", "自社サービス"],
    },
    {
      id: 9,
      name: "フィンテック株式会社",
      industry: "金融×IT",
      location: "東京都千代田区",
      employees: "250名",
      description: "金融業界向けのITソリューションを提供。高い技術力と金融知識を身につけられる。",
      matchScore: 0,
      tags: ["金融IT", "高給与", "成長分野"],
      techStack: ["Java", "Python", "Blockchain"],
      projectTypes: ["金融システム", "フィンテック"],
    },
    {
      id: 10,
      name: "株式会社AIリサーチラボ",
      industry: "AI研究開発",
      location: "東京都文京区",
      employees: "60名",
      description: "最先端のAI研究を行うスタートアップ。論文執筆や学会発表の機会も豊富。",
      matchScore: 0,
      tags: ["AI研究", "スタートアップ", "最先端技術"],
      techStack: ["Python", "TensorFlow", "PyTorch"],
      projectTypes: ["研究開発", "AI開発"],
    },
  ]

  return companies
    .map((company) => {
      let score = 50 // ベーススコア

      // 1. 職種分析
      if (userData.jobType?.includes("開発系") && company.projectTypes.some((p) => p.includes("開発"))) {
        score += 10
      }
      if (userData.jobType?.includes("インフラ") && company.industry.includes("インフラ")) {
        score += 10
      }
      if (userData.programmingConfidence?.includes("得意") && company.tags.includes("技術力重視")) {
        score += 5
      }

      // 2. 興味分析
      if (userData.interestField?.includes("Web") && company.industry.includes("Web")) {
        score += 10
      }
      if (userData.interestField?.includes("AI") && company.industry.includes("AI")) {
        score += 10
      }
      if (userData.interestField?.includes("クラウド") && company.industry.includes("クラウド")) {
        score += 10
      }
      if (userData.interestField?.includes("セキュリティ") && company.industry.includes("セキュリティ")) {
        score += 10
      }
      if (userData.projectType?.includes("自社") && company.projectTypes.includes("自社サービス")) {
        score += 8
      }
      if (userData.projectType?.includes("受託") && company.projectTypes.includes("受託開発")) {
        score += 8
      }

      // 3. 待遇分析
      if (userData.salaryExpectation?.includes("500万") && company.tags.includes("高給与")) {
        score += 8
      }
      if (userData.workStyle?.includes("リモート") && company.tags.includes("リモートワーク")) {
        score += 8
      }
      if (userData.workStyle?.includes("フレックス") && company.tags.includes("フレックス")) {
        score += 5
      }
      if (userData.workStyle?.includes("残業が少ない") && company.tags.includes("ワークライフバランス")) {
        score += 8
      }

      // 4. 将来分析
      if (userData.careerGoal?.includes("スペシャリスト") && company.tags.includes("専門性")) {
        score += 8
      }
      if (userData.careerGoal?.includes("起業") && company.tags.includes("ベンチャー")) {
        score += 8
      }
      if (userData.companySize?.includes("大手") && company.tags.includes("大手企業")) {
        score += 8
      }
      if (userData.companySize?.includes("ベンチャー") && company.tags.includes("ベンチャー")) {
        score += 8
      }

      return { ...company, matchScore: Math.min(score, 99) }
    })
    .sort((a, b) => b.matchScore - a.matchScore)
    .slice(0, 10) // 上位10社に絞り込み
}

export function CompanyResults({ userData, onReset }: { userData: UserData; onReset: () => void }) {
  const companies = generateITCompanies(userData)
  const [selectedCompany, setSelectedCompany] = useState<Company | null>(null)
  const [isDetailOpen, setIsDetailOpen] = useState(false)

  const handleShowDetail = (company: Company) => {
    setSelectedCompany(company)
    setIsDetailOpen(true)
  }

  return (
    <div className="max-w-5xl mx-auto">
      <CompanyDetailModal
        company={selectedCompany}
        isOpen={isDetailOpen}
        onClose={() => setIsDetailOpen(false)}
      />
      <div className="mb-8 text-center">
        <h2 className="text-3xl font-bold text-foreground mb-3 text-balance">
          あなたに適した企業を10社に絞り込みました
        </h2>
        <p className="text-muted-foreground text-pretty">4段階の分析に基づいて、最適なIT企業をマッチングしました</p>
      </div>

      <Card className="mb-6 border-2 border-primary/20 bg-primary/5">
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Award className="w-5 h-5" />
            分析レポート - 企業選定条件
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* 職種分析 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 font-semibold text-sm">
                <Code className="w-4 h-4 text-primary" />
                職種分析
              </div>
              <div className="pl-6 space-y-1 text-sm">
                {userData.jobType && (
                  <div>
                    <span className="text-muted-foreground">希望職種:</span> {userData.jobType}
                  </div>
                )}
                {userData.qualifications && (
                  <div>
                    <span className="text-muted-foreground">資格:</span> {userData.qualifications}
                  </div>
                )}
                {userData.programmingLanguages && (
                  <div>
                    <span className="text-muted-foreground">言語:</span> {userData.programmingLanguages}
                  </div>
                )}
              </div>
            </div>

            {/* 興味分析 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 font-semibold text-sm">
                <Target className="w-4 h-4 text-primary" />
                興味分析
              </div>
              <div className="pl-6 space-y-1 text-sm">
                {userData.interestField && (
                  <div>
                    <span className="text-muted-foreground">興味分野:</span> {userData.interestField}
                  </div>
                )}
                {userData.projectType && (
                  <div>
                    <span className="text-muted-foreground">プロジェクト:</span> {userData.projectType}
                  </div>
                )}
              </div>
            </div>

            {/* 待遇分析 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 font-semibold text-sm">
                <Briefcase className="w-4 h-4 text-primary" />
                待遇分析
              </div>
              <div className="pl-6 space-y-1 text-sm">
                {userData.salaryExpectation && (
                  <div>
                    <span className="text-muted-foreground">希望年収:</span> {userData.salaryExpectation}
                  </div>
                )}
                {userData.workStyle && (
                  <div>
                    <span className="text-muted-foreground">働き方:</span> {userData.workStyle}
                  </div>
                )}
              </div>
            </div>

            {/* 将来分析 */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 font-semibold text-sm">
                <TrendingUp className="w-4 h-4 text-primary" />
                将来分析
              </div>
              <div className="pl-6 space-y-1 text-sm">
                {userData.careerGoal && (
                  <div>
                    <span className="text-muted-foreground">キャリア目標:</span> {userData.careerGoal}
                  </div>
                )}
                {userData.companySize && (
                  <div>
                    <span className="text-muted-foreground">企業規模:</span> {userData.companySize}
                  </div>
                )}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="mb-4">
        <h3 className="text-xl font-bold text-foreground mb-4">選定企業一覧（マッチ度順）</h3>
      </div>

      <div className="space-y-4 mb-8">
        {companies.map((company, index) => (
          <Card key={company.id} className="border-2 hover:border-primary/50 transition-all">
            <CardHeader>
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <div className="flex items-center justify-center w-8 h-8 rounded-full bg-primary text-primary-foreground font-bold text-sm">
                      {index + 1}
                    </div>
                    <div className="p-2 rounded-lg bg-primary/10">
                      <Building2 className="w-5 h-5 text-primary" />
                    </div>
                    <CardTitle className="text-xl">{company.name}</CardTitle>
                  </div>
                  <CardDescription className="text-base">{company.industry}</CardDescription>
                </div>
                <div className="text-right">
                  <div className="text-3xl font-bold text-primary">{company.matchScore}%</div>
                  <div className="text-xs text-muted-foreground">マッチ度</div>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-card-foreground mb-4 leading-relaxed">{company.description}</p>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <MapPin className="w-4 h-4" />
                  {company.location}
                </div>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Users className="w-4 h-4" />
                  {company.employees}
                </div>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <TrendingUp className="w-4 h-4" />
                  {company.industry}
                </div>
              </div>

              <div className="mb-3">
                <div className="text-xs text-muted-foreground mb-1">技術スタック:</div>
                <div className="flex flex-wrap gap-1">
                  {company.techStack.map((tech, techIndex) => (
                    <Badge key={techIndex} variant="secondary" className="text-xs">
                      {tech}
                    </Badge>
                  ))}
                </div>
              </div>

              <div className="flex flex-wrap gap-2 mb-4">
                {company.tags.map((tag, tagIndex) => (
                  <Badge key={tagIndex} variant="outline">
                    {tag}
                  </Badge>
                ))}
              </div>

              <div className="flex gap-2">
                <Button className="flex-1 md:flex-initial" onClick={() => handleShowDetail(company)}>
                  <Eye className="w-4 h-4 mr-2" />
                  詳細を表示
                </Button>
                <Button variant="outline" className="flex-1 md:flex-initial">
                  応募する
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="mb-6 border-2 border-dashed">
        <CardHeader>
          <CardTitle className="text-lg">選定企業の業界マップ（開発中）</CardTitle>
          <CardDescription>資本関連図とビジネス関連図を表示予定</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            将来的に、選定企業の資本関係やビジネス関係を可視化した業界マップを表示します。
          </p>
        </CardContent>
      </Card>


      <div className="text-center">
        <Button variant="outline" onClick={onReset} size="lg">
          <RotateCcw className="w-4 h-4 mr-2" />
          最初からやり直す
        </Button>
      </div>
    </div>
  )
}
