'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Box, Typography } from '@mui/material';
import styles from '../app/Correlation-diagram/CorrelationDiagram.module.css';

// ----------------------------------------------------------------------
// NTTグループのデータ
// ----------------------------------------------------------------------
export const companyNodes = [
    { id: 'n_ntt', name: '日本電信電話株式会社 (NTT)' },
    { id: 'n_docomo', name: 'NTTドコモ' },
    { id: 'n_east', name: 'NTT東日本' },
    { id: 'n_west', name: 'NTT西日本' },
    { id: 'n_data', name: 'NTTデータグループ' },
    { id: 'n_comms', name: 'NTTコミュニケーションズ' },
];

export const capitalLinks = [
    { source: 'n_ntt', target: 'n_docomo', ratio: 100 },
    { source: 'n_ntt', target: 'n_east', ratio: 100 },
    { source: 'n_ntt', target: 'n_west', ratio: 100 },
    { source: 'n_ntt', target: 'n_data', ratio: 100 },
    { source: 'n_ntt', target: 'n_comms', ratio: 100 },
];
// ----------------------------------------------------------------------

interface Position {
    x: number;
    y: number;
}

const MIN_NODE_RADIUS = 30;

export default function CorrelationDiagram() {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const [nodePositions, setNodePositions] = useState<Record<string, Position>>({});
    const [tooltip, setTooltip] = useState<{ x: number; y: number; name: string } | null>(null);

    const dragNodeRef = useRef<string | null>(null);
    const nodePositionsRef = useRef<Record<string, Position>>({});
    const nodeRadiusMap = useRef<Record<string, number>>({}); // 🔹各ノードの半径を保持

    useEffect(() => {
        nodePositionsRef.current = nodePositions;
    }, [nodePositions]);

    // 初期位置設定
    const initPositions = useCallback((width: number, height: number) => {
        const positions: Record<string, Position> = {};
        const centerX = width / 2;
        const topY = 60;

        positions['n_ntt'] = { x: centerX, y: topY };

        const childNodes = companyNodes.filter(node => node.id !== 'n_ntt');
        const numChildren = childNodes.length;
        const levelY = height * 0.75;
        const padding = 60;
        const totalWidth = width - 2 * padding;
        const spacing = totalWidth / (numChildren - 1);

        childNodes.forEach((node, i) => {
            positions[node.id] = { x: padding + i * spacing, y: levelY };
        });

        setNodePositions(positions);
    }, []);

    const draw = useCallback(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;
        const ctx = canvas.getContext('2d');
        if (!ctx) return;

        ctx.clearRect(0, 0, canvas.width, canvas.height);

        // リンク描画
        capitalLinks.forEach(link => {
            const s = nodePositions[link.source];
            const t = nodePositions[link.target];
            if (!s || !t) return;

            ctx.beginPath();
            ctx.moveTo(s.x, s.y);
            ctx.lineTo(t.x, t.y);
            ctx.strokeStyle = '#888';
            ctx.lineWidth = 2;
            ctx.stroke();

            if (link.ratio) {
                const midX = (s.x + t.x) / 2;
                const midY = (s.y + t.y) / 2;
                const text = `${link.ratio}%`;
                ctx.font = '14px sans-serif';
                const textMetrics = ctx.measureText(text);
                const textWidth = textMetrics.width;
                const textHeight = 16;

                ctx.fillStyle = 'rgba(255,255,255,0.8)';
                ctx.fillRect(midX - textWidth / 2 - 2, midY - textHeight / 2 - 6, textWidth + 4, textHeight);

                ctx.fillStyle = 'black';
                ctx.textAlign = 'center';
                ctx.fillText(text, midX, midY - 6);
            }
        });

        // ノード描画（テキスト幅に応じて半径を設定）
        nodeRadiusMap.current = {}; // 毎回リセット
        companyNodes.forEach(node => {
            const pos = nodePositions[node.id];
            if (!pos) return;

            const isParent = node.id === 'n_ntt';
            ctx.font = '12px sans-serif';
            const textWidth = ctx.measureText(node.name).width;
            const radius = Math.max(MIN_NODE_RADIUS, textWidth / 2 + 10);
            nodeRadiusMap.current[node.id] = radius; // 🔹Mouse判定用に保存

            ctx.beginPath();
            ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
            ctx.fillStyle = isParent ? '#f0f0ff' : 'white';
            ctx.fill();
            ctx.strokeStyle = isParent ? '#3a7d9b' : 'black';
            ctx.lineWidth = 2;
            ctx.stroke();

            ctx.fillStyle = 'black';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(node.name, pos.x, pos.y);
        });
    }, [nodePositions]);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;
        canvas.width = canvas.clientWidth;
        canvas.height = canvas.clientHeight;
        initPositions(canvas.width, canvas.height);
    }, [initPositions]);

    useEffect(() => {
        draw();
    }, [draw, nodePositions]);

    // マウス操作
    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const handleMouseDown = (e: MouseEvent) => {
            const rect = canvas.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;

            for (const node of companyNodes) {
                const pos = nodePositionsRef.current[node.id];
                if (!pos) continue;
                const radius = nodeRadiusMap.current[node.id] || MIN_NODE_RADIUS;
                if (Math.hypot(pos.x - x, pos.y - y) <= radius) {
                    dragNodeRef.current = node.id; // 🔹ドラッグ開始
                    break;
                }
            }
        };

        const handleMouseMove = (e: MouseEvent) => {
            const rect = canvas.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;

            if (dragNodeRef.current) {
                setNodePositions(prev => ({
                    ...prev,
                    [dragNodeRef.current!]: { x, y },
                }));
            } else {
                let found = false;
                for (const node of companyNodes) {
                    const pos = nodePositionsRef.current[node.id];
                    if (!pos) continue;
                    const radius = nodeRadiusMap.current[node.id] || MIN_NODE_RADIUS;
                    if (Math.hypot(pos.x - x, pos.y - y) <= radius) {
                        setTooltip({ x: e.clientX, y: e.clientY, name: node.name });
                        found = true;
                        break;
                    }
                }
                if (!found) setTooltip(null);
            }
        };

        const handleMouseUp = () => {
            dragNodeRef.current = null; // 🔹ドラッグ終了
        };

        canvas.addEventListener('mousedown', handleMouseDown);
        canvas.addEventListener('mousemove', handleMouseMove);
        window.addEventListener('mouseup', handleMouseUp);

        return () => {
            canvas.removeEventListener('mousedown', handleMouseDown);
            canvas.removeEventListener('mousemove', handleMouseMove);
            window.removeEventListener('mouseup', handleMouseUp);
        };
    }, []);

    return (
        <Box className={styles.container}>
            <Card className={styles.card}>
                <CardContent className={styles.card}>
                    <canvas ref={canvasRef} className={styles.canvas} />
                </CardContent>
            </Card>

            {tooltip && (
                <Box
                    sx={{
                        position: 'fixed',
                        left: tooltip.x + 10,
                        top: tooltip.y + 10,
                        bgcolor: 'background.paper',
                        px: 1,
                        py: 0.5,
                        borderRadius: 1,
                        boxShadow: 2,
                        pointerEvents: 'none',
                        zIndex: 1000,
                    }}
                >
                    <Typography variant="body2">{tooltip.name}</Typography>
                </Box>
            )}
        </Box>
    );
}
