# AI-就活エージェント

就活に悩んでいる我が校の生徒（2年学科1年生4年学科3年生）向け就活AI<br>
自分が何をしたいかを決めて企業選定をしてくれます。

## 特徴

- 機能1 4段階企業分析による企業選定
- 機能2　自分に合ったさまざまな会社を見つけられる
- 機能3　自分のしたいことを見つけられる

## 技術スタック

- Frontend: Next.js / React / TypeScript
- Backend: Echo/Go
- Database: MySQL 
- Infrastructure: Docker / AWS など

## ディレクトリ構成
### forntend
<pre>
front/
├── app/                    # Next.js App Router pages
│   ├── layout.tsx
│   ├── page.tsx
│   └── (feature directories)
│
├── components/              # UI コンポーネント
│   ├── ui/
│   └── common/
│
├── lib/                     # API / util 関数
│   ├── api/
│   └── helpers/
│
├── hooks/                   # カスタムフック
│
├── styles/
│   ├── globals.css
│   └── variables.css
│
├── public/                  # 画像・静的ファイル
│   ├── favicon.ico
│   ├── logo.png
│   └── assets/
│
├── types/                   # 型定義（Optional）
│   └── index.d.ts
│
├── .env.example             # フロント環境変数
├── next.config.js
├── package.json
├── tsconfig.json
└── README.md
</pre>