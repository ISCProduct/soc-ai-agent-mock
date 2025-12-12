'use client'

import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import {
  Box,
  Paper,
  Typography,
  Button,
  Card,
  CardContent,
  Stack,
  CircularProgress,
  Avatar,
  Chip,
  Tabs,
  Tab,
} from '@mui/material'
import { ArrowBack, LocationOn, People, TrendingUp as TrendingUpIcon, Refresh } from '@mui/icons-material'
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  MiniMap,
  MarkerType,
  EdgeTypes,
  ReactFlowProvider,
} from 'reactflow'
import 'reactflow/dist/style.css'
import {
  fetchCompanyRelations,
  fetchCompanyMarketInfo,
  marketColors,
  marketLabels,
  type CapitalRelation,
  type CompanyMarketInfo,
  type MarketType,
} from '@/lib/company-data'

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

const CustomEdge = ({ id, sourceX, sourceY, targetX, targetY, style, markerEnd, label }: any) => {
  const edgePath = `M ${sourceX} ${sourceY} L ${targetX} ${targetY}`
  
  // ラベルの位置を計算（中点）
  const labelX = (sourceX + targetX) / 2
  const labelY = (sourceY + targetY) / 2
  
  // エッジの角度を計算
  const angle = Math.atan2(targetY - sourceY, targetX - sourceX) * (180 / Math.PI)
  
  // テキストが逆さまにならないように調整（-90度〜90度の範囲に収める）
  const adjustedAngle = angle > 90 || angle < -90 ? angle + 180 : angle
  
  return (
    <>
      <path
        id={id}
        style={style}
        className="react-flow__edge-path"
        d={edgePath}
        markerEnd={markerEnd}
      />
      {label && (
        <text
          x={labelX}
          y={labelY}
          style={{
            fontSize: '13px',
            fill: '#333',
            fontWeight: 600,
            pointerEvents: 'none',
          }}
          textAnchor="middle"
          dominantBaseline="middle"
          transform={`rotate(${adjustedAngle}, ${labelX}, ${labelY})`}
        >
          {/* 白い縁取り（背景） */}
          <tspan
            x={labelX}
            dy="0"
            style={{
              fill: 'none',
              stroke: '#fff',
              strokeWidth: 4,
              strokeLinejoin: 'round',
              paintOrder: 'stroke',
            }}
          >
            {label}
          </tspan>
          {/* メインテキスト */}
          <tspan
            x={labelX}
            dy="0"
            style={{
              fill: '#333',
            }}
          >
            {label}
          </tspan>
        </text>
      )}
    </>
  )
}

const edgeTypes: EdgeTypes = {
  custom: CustomEdge,
}

function ResultsContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [companies, setCompanies] = useState<Company[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [mounted, setMounted] = useState(false)
  const [selectedCompany, setSelectedCompany] = useState<Company | null>(null)
  const [detailTab, setDetailTab] = useState(0)
  const [relations, setRelations] = useState<CapitalRelation[]>([])
  const [marketInfo, setMarketInfo] = useState<CompanyMarketInfo[]>([])
  const [diagramLoading, setDiagramLoading] = useState(false)

  const userId = searchParams.get('user_id')
  const sessionId = searchParams.get('session_id')

  useEffect(() => {
    setMounted(true)
  }, [])

  useEffect(() => {
    if (!mounted || !userId || !sessionId) {
      if (mounted && (!userId || !sessionId)) {
        setError('セッション情報が見つかりません')
        setLoading(false)
      }
      return
    }

    const fetchCompanies = async () => {
      try {
        setLoading(true)
        console.log('[Results] Fetching recommendations for user:', userId, 'session:', sessionId)
        const response = await fetch(`/api/chat/recommendations?user_id=${userId}&session_id=${sessionId}&limit=10`)
        
        if (!response.ok) {
          throw new Error('企業データの取得に失敗しました')
        }
        
        const data = await response.json()
        console.log('[Results] API Response:', data)
        
        if (!data || !data.recommendations || !Array.isArray(data.recommendations) || data.recommendations.length === 0) {
          console.error('[Results] No recommendations available')
          console.error('[Results] Debug info - User ID:', userId, 'Session ID:', sessionId)
          console.error('[Results] Response data:', data)
          setError(
            'まだ企業マッチングデータがありません。\n\n' +
            '診断を完了してから数秒待ち、このページを更新してください。\n' +
            '診断が未完了の場合は、チャット画面で最低15問に回答してください。\n\n' +
            `デバッグ情報:\n` +
            `- ユーザーID: ${userId}\n` +
            `- セッションID: ${sessionId}\n` +
            `- レスポンス: ${JSON.stringify(data)}`
          )
          setLoading(false)
          return
        }
        
        if (data && data.recommendations && Array.isArray(data.recommendations)) {
          // 企業データをマッピング
          const mappedCompanies = data.recommendations.map((rec: any, index: number) => {
            console.log('[Results] Mapping company data:', rec)
            return {
              id: String(rec.id || rec.ID || index + 1),
              name: rec.category_name || rec.name || `企業 ${index + 1}`,
              industry: rec.industry || 'IT・ソフトウェア',
              location: rec.location || '東京都',
              employees: rec.employees || '未定',
              description: rec.reason || '詳細情報は準備中です',
              matchScore: rec.score || 0,
              tags: rec.tags || [],
              techStack: rec.tech_stack || [],
            }
          })
          console.log('[Results] Mapped companies:', mappedCompanies)
          setCompanies(mappedCompanies)
        } else {
          console.error('[Results] Invalid data format:', data)
          setError('企業データの形式が正しくありません')
        }
      } catch (err) {
        console.error('[Results] 企業データ取得エラー:', err)
        setError(err instanceof Error ? err.message : '不明なエラー')
      } finally {
        setLoading(false)
      }
    }

    fetchCompanies()
  }, [mounted, userId, sessionId])

  // 企業詳細を開いたときに関係図データを取得
  useEffect(() => {
    if (selectedCompany && (detailTab === 1 || detailTab === 2)) {
      const loadDiagramData = async () => {
        if (relations.length === 0 || marketInfo.length === 0) {
          setDiagramLoading(true)
          console.log('[Relations] Fetching company relations and market info...')
          const [relationsData, marketData] = await Promise.all([
            fetchCompanyRelations(),
            fetchCompanyMarketInfo()
          ])
          console.log('[Relations] Fetched relations:', relationsData.length)
          console.log('[Relations] Business relations:', relationsData.filter(r => r.relation_type.startsWith('business')).length)
          console.log('[Relations] Capital relations:', relationsData.filter(r => r.relation_type.startsWith('capital')).length)
          setRelations(relationsData)
          setMarketInfo(marketData)
          setDiagramLoading(false)
        } else {
          console.log('[Relations] Using cached data - relations:', relations.length)
        }
      }
      loadDiagramData()
    }
  }, [selectedCompany, detailTab, relations.length, marketInfo.length])

  const handleBack = () => {
    router.push('/')
  }

  const handleReset = () => {
    localStorage.clear()
    sessionStorage.clear()
    router.push('/')
  }

  // 関係図のノードとエッジを生成
  const createDiagramData = (companyId: string, type: 'capital' | 'business') => {
    const compId = parseInt(companyId)
    const relatedIds = new Set([compId])
    
    console.log(`[Diagram] Creating ${type} diagram for company ${compId}`)
    console.log(`[Diagram] Total relations available:`, relations.length)
    
    let matchedRelations = 0
    relations.forEach(rel => {
      if (type === 'capital' && rel.relation_type.startsWith('capital')) {
        if (rel.parent_id === compId || rel.child_id === compId) {
          if (rel.parent_id) relatedIds.add(rel.parent_id)
          if (rel.child_id) relatedIds.add(rel.child_id)
          matchedRelations++
        }
      } else if (type === 'business' && rel.relation_type.startsWith('business')) {
        if (rel.from_id === compId || rel.to_id === compId) {
          if (rel.from_id === compId && rel.to_id) relatedIds.add(rel.to_id)
          if (rel.to_id === compId && rel.from_id) relatedIds.add(rel.from_id)
          matchedRelations++
        }
      }
    })
    
    console.log(`[Diagram] Matched ${matchedRelations} ${type} relations for company ${compId}`)
    console.log(`[Diagram] Related company IDs:`, Array.from(relatedIds))

    const getMarketType = (id: number): MarketType => {
      const info = marketInfo.find(m => m.company_id === id)
      return info?.market_type || 'unlisted'
    }

    const getCompanyName = (id: number): string => {
      for (const rel of relations) {
        if (rel.parent?.id === id) return rel.parent.name
        if (rel.child?.id === id) return rel.child.name
        if (rel.from?.id === id) return rel.from.name
        if (rel.to?.id === id) return rel.to.name
      }
      return `企業 ${id}`
    }

    // ノード生成
    const nodes: Node[] = []
    const ids = Array.from(relatedIds)
    const angle = (2 * Math.PI) / ids.length
    const radius = Math.max(200, ids.length * 40)

    ids.forEach((id, idx) => {
      const isFocusCompany = id === compId
      const marketType = getMarketType(id)

      nodes.push({
        id: String(id),
        type: 'default',
        position: {
          x: 400 + radius * Math.cos(idx * angle),
          y: 300 + radius * Math.sin(idx * angle),
        },
        data: {
          label: (
            <Box sx={{ textAlign: 'center', p: 1 }}>
              <Typography 
                variant="body2" 
                sx={{ fontWeight: isFocusCompany ? 'bold' : 'normal', mb: 0.5 }}
              >
                {getCompanyName(id)}
              </Typography>
              <Chip
                label={marketLabels[marketType]}
                size="small"
                sx={{
                  bgcolor: marketColors[marketType],
                  color: 'white',
                  fontSize: '10px',
                  height: '20px',
                }}
              />
            </Box>
          ),
        },
        style: {
          background: isFocusCompany ? '#FFF3CD' : '#fff',
          border: `3px solid ${isFocusCompany ? '#FFC107' : marketColors[marketType]}`,
          borderRadius: '8px',
          padding: '10px',
          minWidth: '200px',
          boxShadow: isFocusCompany ? '0 4px 12px rgba(255, 193, 7, 0.3)' : undefined,
        },
      })
    })

    // エッジ生成
    const edges: Edge[] = []
    relations.forEach((rel, idx) => {
      if (type === 'capital' && rel.relation_type.startsWith('capital') && rel.parent_id && rel.child_id) {
        if (relatedIds.has(rel.parent_id) && relatedIds.has(rel.child_id)) {
          edges.push({
            id: `capital-${idx}`,
            source: String(rel.parent_id),
            target: String(rel.child_id),
            type: 'custom',
            label: rel.ratio ? `${rel.ratio.toFixed(0)}%` : '',
            style: {
              stroke: '#555',
              strokeWidth: 2,
              strokeDasharray: rel.relation_type === 'capital_affiliate' ? '5,5' : 'none',
            },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: '#555',
            },
          })
        }
      } else if (type === 'business' && rel.relation_type.startsWith('business') && rel.from_id && rel.to_id) {
        if (relatedIds.has(rel.from_id) && relatedIds.has(rel.to_id)) {
          edges.push({
            id: `business-${idx}`,
            source: String(rel.from_id),
            target: String(rel.to_id),
            type: 'custom',
            label: rel.description || rel.relation_type,
            animated: true,
            style: {
              stroke: '#2196F3',
              strokeWidth: 2,
            },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: '#2196F3',
            },
          })
        }
      }
    })

    return { nodes, edges }
  }

  if (!mounted) {
    return null
  }

  if (loading) {
    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 2,
      }}>
        <CircularProgress size={60} />
        <Typography variant="h6" color="text.secondary">
          AIが企業を分析中...
        </Typography>
      </Box>
    )
  }

  if (error) {
    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 2,
        p: 3,
      }}>
        <Typography variant="h6" color="error" sx={{ whiteSpace: 'pre-line', textAlign: 'center' }}>
          {error}
        </Typography>
        <Stack direction="row" spacing={2}>
          <Button 
            variant="contained" 
            startIcon={<Refresh />}
            onClick={() => window.location.reload()}
          >
            ページを更新
          </Button>
          <Button 
            variant="outlined" 
            onClick={() => router.push('/chat')}
          >
            チャットに戻る
          </Button>
          <Button 
            variant="outlined" 
            color="error"
            onClick={handleReset}
          >
            最初からやり直す
          </Button>
        </Stack>
      </Box>
    )
  }

  if (companies.length === 0) {
    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexDirection: 'column',
        gap: 2,
      }}>
        <Typography variant="h6" color="text.secondary">
          適合する企業が見つかりませんでした
        </Typography>
        <Button variant="outlined" onClick={handleReset}>
          最初からやり直す
        </Button>
      </Box>
    )
  }

  // 企業詳細ダイアログを表示している場合
  if (selectedCompany) {
    const capitalDiagram = detailTab === 1 ? createDiagramData(selectedCompany.id, 'capital') : { nodes: [], edges: [] }
    const businessDiagram = detailTab === 2 ? createDiagramData(selectedCompany.id, 'business') : { nodes: [], edges: [] }

    return (
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        backgroundColor: '#fff',
      }}>
        {/* ヘッダー */}
        <Box sx={{ 
          p: 3, 
          borderBottom: '1px solid #e0e0e0',
          backgroundColor: '#fff',
          flexShrink: 0,
        }}>
          <Button variant="outlined" startIcon={<ArrowBack />} onClick={() => {
            setSelectedCompany(null)
            setDetailTab(0)
          }}>
            企業一覧に戻る
          </Button>
        </Box>

        {/* 企業詳細コンテンツ */}
        <Box sx={{ 
          flexGrow: 1,
          overflowY: 'auto',
          p: 3,
          backgroundColor: '#fafafa',
        }}>
          <Box sx={{ maxWidth: 1200, mx: 'auto' }}>
            <Card elevation={3}>
              <CardContent sx={{ p: 4 }}>
                {/* 企業名とマッチスコア */}
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
                  <Box>
                    <Typography variant="h4" fontWeight="bold" gutterBottom>
                      {selectedCompany.name}
                    </Typography>
                    <Typography variant="h6" color="text.secondary">
                      {selectedCompany.industry}
                    </Typography>
                  </Box>
                  <Box sx={{ textAlign: 'right' }}>
                    <Typography variant="h2" color="primary.main" fontWeight="bold">
                      {selectedCompany.matchScore}
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                      適合度
                    </Typography>
                  </Box>
                </Box>

                {/* タブ */}
                <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
                  <Tabs value={detailTab} onChange={(e, newValue) => setDetailTab(newValue)}>
                    <Tab label="基本情報" />
                    <Tab label="資本関連図" />
                    <Tab label="ビジネス関連図" />
                  </Tabs>
                </Box>

                {/* タブ0: 基本情報 */}
                {detailTab === 0 && (
                  <>
                    {/* 基本情報 */}
                    <Box sx={{ mb: 4 }}>
                      <Typography variant="h6" fontWeight="bold" gutterBottom>
                        📍 基本情報
                      </Typography>
                      <Stack spacing={2}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <LocationOn color="action" />
                          <Typography variant="body1">
                            <strong>所在地:</strong> {selectedCompany.location}
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <People color="action" />
                          <Typography variant="body1">
                            <strong>従業員数:</strong> {selectedCompany.employees}
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <TrendingUpIcon color="action" />
                          <Typography variant="body1">
                            <strong>業種:</strong> {selectedCompany.industry}
                          </Typography>
                        </Box>
                      </Stack>
                    </Box>

                    {/* マッチング理由 */}
                    <Box sx={{ mb: 4 }}>
                      <Typography variant="h6" fontWeight="bold" gutterBottom>
                        💡 マッチング理由
                      </Typography>
                      <Paper sx={{ p: 2, backgroundColor: '#f5f5f5' }}>
                        <Typography variant="body1">
                          {selectedCompany.description}
                        </Typography>
                      </Paper>
                    </Box>

                    {/* 技術スタック */}
                    {selectedCompany.techStack && selectedCompany.techStack.length > 0 && (
                      <Box sx={{ mb: 4 }}>
                        <Typography variant="h6" fontWeight="bold" gutterBottom>
                          🛠️ 技術スタック
                        </Typography>
                        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                          {selectedCompany.techStack.map((tech, i) => (
                            <Chip 
                              key={i} 
                              label={tech} 
                              color="primary" 
                              variant="filled"
                              sx={{ fontSize: '0.95rem', py: 2.5 }}
                            />
                          ))}
                        </Stack>
                      </Box>
                    )}

                    {/* タグ */}
                    {selectedCompany.tags && selectedCompany.tags.length > 0 && (
                      <Box sx={{ mb: 4 }}>
                        <Typography variant="h6" fontWeight="bold" gutterBottom>
                          🏷️ 企業タグ
                        </Typography>
                        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                          {selectedCompany.tags.map((tag, i) => (
                            <Chip key={i} label={tag} variant="outlined" />
                          ))}
                        </Stack>
                      </Box>
                    )}
                  </>
                )}

                {/* タブ1: 資本関連図 */}
                {detailTab === 1 && (
                  <Box sx={{ height: 600 }}>
                    {diagramLoading ? (
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                        <CircularProgress />
                      </Box>
                    ) : capitalDiagram.nodes.length === 0 ? (
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                        <Typography color="text.secondary">この企業の資本関連情報はありません</Typography>
                      </Box>
                    ) : (
                      <>
                        <Box sx={{ mb: 2, display: 'flex', gap: 3, fontSize: '12px', flexWrap: 'wrap' }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                            <Box sx={{ width: 40, height: 2, bgcolor: '#555' }} />
                            <span>子会社（実線）</span>
                          </Box>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                            <Box sx={{ width: 40, height: 2, borderTop: '2px dashed #555' }} />
                            <span>関連会社（破線）</span>
                          </Box>
                          {Object.entries(marketLabels).map(([key, label]) => (
                            <Box key={key} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                              <Box sx={{ width: 16, height: 16, bgcolor: marketColors[key as MarketType], borderRadius: '50%' }} />
                              <Typography variant="caption">{label}</Typography>
                            </Box>
                          ))}
                        </Box>
                        <Box sx={{ height: 'calc(100% - 40px)', border: '1px solid #e0e0e0', borderRadius: 1 }}>
                          <ReactFlow
                            nodes={capitalDiagram.nodes}
                            edges={capitalDiagram.edges}
                            edgeTypes={edgeTypes}
                            fitView
                            minZoom={0.05}
                            maxZoom={3}
                            defaultViewport={{ x: 0, y: 0, zoom: 0.8 }}
                            attributionPosition="bottom-right"
                            nodesDraggable={true}
                            nodesConnectable={false}
                            elementsSelectable={true}
                          >
                            <Background color="#aaa" gap={16} />
                            <Controls 
                              showZoom={true}
                              showFitView={true}
                              showInteractive={true}
                              position="top-right"
                            />
                            <MiniMap 
                              nodeColor={(node) => {
                                const border = node.style?.border as string
                                if (border?.includes('#FFA726')) return '#FFA726'
                                return '#2196F3'
                              }}
                              maskColor="rgba(0, 0, 0, 0.1)"
                              position="bottom-left"
                            />
                          </ReactFlow>
                        </Box>
                      </>
                    )}
                  </Box>
                )}

                {/* タブ2: ビジネス関連図 */}
                {detailTab === 2 && (
                  <Box sx={{ height: 600 }}>
                    {diagramLoading ? (
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                        <CircularProgress />
                      </Box>
                    ) : businessDiagram.nodes.length === 0 ? (
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                        <Typography color="text.secondary">この企業のビジネス関連情報はありません</Typography>
                      </Box>
                    ) : (
                      <>
                        <Box sx={{ mb: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                          {Object.entries(marketLabels).map(([key, label]) => (
                            <Box key={key} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                              <Box sx={{ width: 16, height: 16, bgcolor: marketColors[key as MarketType], borderRadius: '50%' }} />
                              <Typography variant="caption">{label}</Typography>
                            </Box>
                          ))}
                        </Box>
                        <Box sx={{ height: 'calc(100% - 40px)', border: '1px solid #e0e0e0', borderRadius: 1 }}>
                          <ReactFlow
                            nodes={businessDiagram.nodes}
                            edges={businessDiagram.edges}
                            edgeTypes={edgeTypes}
                            fitView
                            minZoom={0.05}
                            maxZoom={3}
                            defaultViewport={{ x: 0, y: 0, zoom: 0.8 }}
                            attributionPosition="bottom-right"
                            nodesDraggable={true}
                            nodesConnectable={false}
                            elementsSelectable={true}
                          >
                            <Background color="#aaa" gap={16} />
                            <Controls 
                              showZoom={true}
                              showFitView={true}
                              showInteractive={true}
                              position="top-right"
                            />
                            <MiniMap 
                              nodeColor={(node) => {
                                const border = node.style?.border as string
                                if (border?.includes('#FFA726')) return '#FFA726'
                                return '#2196F3'
                              }}
                              maskColor="rgba(0, 0, 0, 0.1)"
                              position="bottom-left"
                            />
                          </ReactFlow>
                        </Box>
                      </>
                    )}
                  </Box>
                )}
              </CardContent>
            </Card>
          </Box>
        </Box>
      </Box>
    )
  }

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
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
          <Button variant="outlined" startIcon={<ArrowBack />} onClick={handleBack}>
            チャットに戻る
          </Button>
        </Box>
        <Box sx={{ textAlign: 'center' }}>
          <Typography variant="h4" fontWeight="bold" gutterBottom>
            🎉 AI分析完了！適合企業を{companies.length}社に絞り込みました
          </Typography>
          <Typography variant="body1" color="text.secondary">
            AIによる詳細分析に基づいて、最適なIT企業をマッチングしました
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
              <Card 
                key={`${company.id}-${index}`} 
                elevation={3} 
                sx={{ 
                  border: '2px solid', 
                  borderColor: 'primary.light',
                  cursor: 'pointer',
                  transition: 'all 0.3s ease',
                  '&:hover': {
                    borderColor: 'primary.main',
                    transform: 'translateY(-4px)',
                    boxShadow: 6,
                  }
                }}
                onClick={() => setSelectedCompany(company)}
              >
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
                        {company.matchScore}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        適合度
                      </Typography>
                    </Box>
                  </Box>

                  <Typography variant="body2" sx={{ mb: 2 }}>
                    {company.description}
                  </Typography>

                  <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap', gap: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LocationOn fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.location}
                      </Typography>
                    </Box>
                    {company.employees && (
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <People fontSize="small" color="action" />
                        <Typography variant="body2" color="text.secondary">
                          {company.employees}
                        </Typography>
                      </Box>
                    )}
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <TrendingUpIcon fontSize="small" color="action" />
                      <Typography variant="body2" color="text.secondary">
                        {company.industry}
                      </Typography>
                    </Box>
                  </Stack>

                  {company.techStack && company.techStack.length > 0 && (
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
                  )}

                  {company.tags && company.tags.length > 0 && (
                    <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                      {company.tags.map((tag, i) => (
                        <Chip key={i} label={tag} size="small" />
                      ))}
                    </Stack>
                  )}
                  
                  <Box sx={{ mt: 2, textAlign: 'right' }}>
                    <Typography variant="caption" color="primary" sx={{ fontWeight: 'bold' }}>
                      クリックして詳細を見る →
                    </Typography>
                  </Box>
                </CardContent>
              </Card>
            ))}
          </Stack>

          <Box sx={{ textAlign: 'center', mt: 4, mb: 4 }}>
            <Button variant="outlined" size="large" startIcon={<Refresh />} onClick={handleReset}>
              最初からやり直す
            </Button>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}

export default function ResultsPage() {
  return (
    <Suspense fallback={
      <Box sx={{ 
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <CircularProgress size={60} />
      </Box>
    }>
      <ResultsContent />
    </Suspense>
  )
}
