'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { Button, Box } from '@mui/material';
import { Suspense } from 'react';
import CorrelationDiagram from '@/components/Correlation-diagram';

function CorrelationDiagramContent() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const companyIdParam = searchParams.get('company_id');
    const initialCompanyId = companyIdParam ? parseInt(companyIdParam, 10) : null;

    return (
        <Box sx={{ p: 2 }}>
            <Button variant="contained" onClick={() => router.back()} sx={{ mb: 2 }}>
                戻る
            </Button>

            <CorrelationDiagram initialCompanyId={initialCompanyId} />
        </Box>
    );
}

export default function Page() {
    return (
        <Suspense fallback={null}>
            <CorrelationDiagramContent />
        </Suspense>
    );
}
