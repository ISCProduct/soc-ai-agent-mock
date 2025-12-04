'use client';

import { useRouter } from 'next/navigation';
import { Button, Box } from '@mui/material';
import CorrelationDiagram from '@/components/Correlation-diagram';

export default function Page() {
    const router = useRouter();

    return (
        <Box sx={{ p: 2 }}>
            <Button variant="contained" onClick={() => router.back()} sx={{ mb: 2 }}>
                戻る
            </Button>

            <CorrelationDiagram />
        </Box>
    );
}
