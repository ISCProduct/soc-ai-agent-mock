/** @type {import('next').NextConfig} */
const nextConfig = {
    // Dockerビルドで最小限の実行環境を構築するために必須の設定です
    output: 'standalone',

    // その他の設定をここに追加します
    // 例:
    // reactStrictMode: true,
    // basePath: '/soc-agent',
};

module.exports = nextConfig;