"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Building2,
  Users,
  MapPin,
  Globe,
  TrendingUp,
  Calendar,
  Briefcase,
  DollarSign,
  Network,
  ChevronRight,
  X,
} from "lucide-react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

type CompanyData = {
  id: number
  name: string
  industry: string
  size: string
  location: string
  founded: string
  website: string
  description: string
  matchScore: number
  salary: string
  benefits: string[]
  culture: string[]
  techStack?: string[]
  parentCompany?: string
  subsidiaries?: string[]
  partnerships?: string[]
  capitalStructure?: {
    shareholders: { name: string; percentage: number }[]
  }
}

type CompanyDetailModalProps = {
  company: CompanyData | null
  isOpen: boolean
  onClose: () => void
}

export function CompanyDetailModal({ company, isOpen, onClose }: CompanyDetailModalProps) {
  const [activeTab, setActiveTab] = useState("overview")

  if (!company) return null

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-start justify-between">
            <div className="space-y-2">
              <DialogTitle className="text-2xl">{company.name}</DialogTitle>
              <div className="flex items-center gap-2">
                <Badge variant="secondary">{company.industry}</Badge>
                <Badge variant="outline">{company.size}</Badge>
                <div className="flex items-center gap-1 text-sm text-muted-foreground">
                  <MapPin className="w-4 h-4" />
                  {company.location}
                </div>
              </div>
            </div>
            <div className="text-right">
              <div className="text-sm text-muted-foreground">適合度</div>
              <div className="text-3xl font-bold text-primary">{company.matchScore}%</div>
            </div>
          </div>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="overview">概要</TabsTrigger>
            <TabsTrigger value="details">詳細情報</TabsTrigger>
            <TabsTrigger value="relations">企業相関図</TabsTrigger>
            <TabsTrigger value="capital">資本関係</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6 mt-6">
            <div>
              <h3 className="font-semibold text-lg mb-2 flex items-center gap-2">
                <Building2 className="w-5 h-5" />
                企業概要
              </h3>
              <p className="text-muted-foreground leading-relaxed">{company.description}</p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <Card className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <DollarSign className="w-5 h-5 text-primary" />
                  <h4 className="font-semibold">想定年収</h4>
                </div>
                <p className="text-2xl font-bold text-primary">{company.salary}</p>
              </Card>

              <Card className="p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Calendar className="w-5 h-5 text-primary" />
                  <h4 className="font-semibold">設立年</h4>
                </div>
                <p className="text-2xl font-bold">{company.founded}</p>
              </Card>
            </div>

            <div>
              <h3 className="font-semibold text-lg mb-3 flex items-center gap-2">
                <Briefcase className="w-5 h-5" />
                福利厚生
              </h3>
              <div className="flex flex-wrap gap-2">
                {company.benefits.map((benefit, index) => (
                  <Badge key={index} variant="secondary">
                    {benefit}
                  </Badge>
                ))}
              </div>
            </div>

            <div>
              <h3 className="font-semibold text-lg mb-3 flex items-center gap-2">
                <Users className="w-5 h-5" />
                企業文化
              </h3>
              <div className="flex flex-wrap gap-2">
                {company.culture.map((item, index) => (
                  <Badge key={index} variant="outline">
                    {item}
                  </Badge>
                ))}
              </div>
            </div>
          </TabsContent>

          <TabsContent value="details" className="space-y-6 mt-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="space-y-4">
                <div>
                  <div className="text-sm text-muted-foreground mb-1">業界</div>
                  <div className="font-medium">{company.industry}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground mb-1">従業員数</div>
                  <div className="font-medium">{company.size}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground mb-1">本社所在地</div>
                  <div className="font-medium">{company.location}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground mb-1">設立</div>
                  <div className="font-medium">{company.founded}</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="text-sm text-muted-foreground mb-1">ウェブサイト</div>
                  <a
                    href={company.website}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium text-primary hover:underline flex items-center gap-1"
                  >
                    <Globe className="w-4 h-4" />
                    {company.website}
                  </a>
                </div>
                {company.techStack && company.techStack.length > 0 && (
                  <div>
                    <div className="text-sm text-muted-foreground mb-2">技術スタック</div>
                    <div className="flex flex-wrap gap-2">
                      {company.techStack.map((tech, index) => (
                        <Badge key={index} variant="secondary">
                          {tech}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </TabsContent>

          <TabsContent value="relations" className="mt-6">
            <CompanyRelationDiagram company={company} />
          </TabsContent>

          <TabsContent value="capital" className="mt-6">
            <CapitalStructure company={company} />
          </TabsContent>
        </Tabs>

        <div className="flex justify-end gap-2 pt-4 border-t">
          <Button variant="outline" onClick={onClose}>
            閉じる
          </Button>
          <Button>
            <ChevronRight className="w-4 h-4 mr-1" />
            この企業に応募する
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function CompanyRelationDiagram({ company }: { company: CompanyData }) {
  return (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="font-semibold text-lg mb-2">企業相関図</h3>
        <p className="text-sm text-muted-foreground">
          {company.name}の関連企業とパートナーシップ
        </p>
      </div>

      <div className="relative p-8">
        {/* 中央の企業 */}
        <div className="flex justify-center mb-12">
          <Card className="p-6 bg-primary text-primary-foreground max-w-xs">
            <div className="text-center">
              <Building2 className="w-8 h-8 mx-auto mb-2" />
              <div className="font-bold text-lg">{company.name}</div>
              <div className="text-xs opacity-80 mt-1">対象企業</div>
            </div>
          </Card>
        </div>

        {/* 親会社 */}
        {company.parentCompany && (
          <div className="mb-8">
            <div className="text-sm text-muted-foreground mb-3 text-center">親会社</div>
            <div className="flex justify-center">
              <Card className="p-4 max-w-xs border-2 border-primary/50">
                <div className="text-center">
                  <Network className="w-6 h-6 mx-auto mb-1 text-primary" />
                  <div className="font-medium">{company.parentCompany}</div>
                </div>
              </Card>
            </div>
            <div className="w-0.5 h-8 bg-border mx-auto" />
          </div>
        )}

        {/* 子会社 */}
        {company.subsidiaries && company.subsidiaries.length > 0 && (
          <div className="mb-8">
            <div className="text-sm text-muted-foreground mb-3 text-center">子会社・関連会社</div>
            <div className="grid grid-cols-3 gap-4">
              {company.subsidiaries.map((sub, index) => (
                <Card key={index} className="p-3">
                  <div className="text-center">
                    <Building2 className="w-5 h-5 mx-auto mb-1 text-muted-foreground" />
                    <div className="text-sm font-medium">{sub}</div>
                  </div>
                </Card>
              ))}
            </div>
          </div>
        )}

        {/* パートナー企業 */}
        {company.partnerships && company.partnerships.length > 0 && (
          <div>
            <div className="text-sm text-muted-foreground mb-3 text-center">パートナーシップ</div>
            <div className="grid grid-cols-3 gap-4">
              {company.partnerships.map((partner, index) => (
                <Card key={index} className="p-3 border-dashed">
                  <div className="text-center">
                    <Network className="w-5 h-5 mx-auto mb-1 text-primary/70" />
                    <div className="text-sm font-medium">{partner}</div>
                  </div>
                </Card>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function CapitalStructure({ company }: { company: CompanyData }) {
  if (!company.capitalStructure) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        資本構成情報は現在準備中です
      </div>
    )
  }

  const maxPercentage = Math.max(...company.capitalStructure.shareholders.map((s) => s.percentage))

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="font-semibold text-lg mb-2">資本構成</h3>
        <p className="text-sm text-muted-foreground">主要株主の保有比率</p>
      </div>

      <div className="space-y-3">
        {company.capitalStructure.shareholders.map((shareholder, index) => (
          <div key={index} className="space-y-2">
            <div className="flex justify-between items-center">
              <div className="font-medium">{shareholder.name}</div>
              <div className="text-sm font-bold text-primary">{shareholder.percentage}%</div>
            </div>
            <div className="w-full bg-muted rounded-full h-3 overflow-hidden">
              <div
                className="bg-primary h-full transition-all duration-500 rounded-full"
                style={{
                  width: `${(shareholder.percentage / maxPercentage) * 100}%`,
                }}
              />
            </div>
          </div>
        ))}
      </div>

      <Card className="p-4 bg-muted/50">
        <div className="text-sm text-muted-foreground">
          <strong>注:</strong> 上記の資本構成は最新の有価証券報告書に基づくものです。
        </div>
      </Card>
    </div>
  )
}
