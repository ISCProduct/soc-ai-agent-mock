export const CERTIFICATION_OPTIONS = [
  'なし',
  'ITパスポート',
  '基本情報技術者',
  '応用情報技術者',
  '情報処理安全確保支援士',
  'AWS Certified Cloud Practitioner',
  'AWS Certified Solutions Architect - Associate',
  'AWS Certified Developer - Associate',
  'Microsoft Azure AZ-900',
  'Microsoft Azure AZ-204',
  'Google Cloud Associate Cloud Engineer',
  'Google Cloud Professional Cloud Architect',
  'Oracle Java SE 11 Silver',
  'Oracle Java SE 11 Gold',
  'Python 3 エンジニア認定基礎',
  'Python 3 エンジニア認定データ分析',
  'LPIC-1',
  'CCNA',
  'TOEIC 600+',
  'TOEIC 700+',
  'TOEIC 800+',
]

export function splitCertifications(value?: string): string[] {
  if (!value) return []
  return value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

export function joinCertifications(values: string[]): string {
  return values
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
    .join(', ')
}
