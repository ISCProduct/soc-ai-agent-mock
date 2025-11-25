'use client'

import { Box } from '@mui/material'
import { AnalysisSidebar } from '@/components/analysis-sidebar'
import { MuiChat } from '@/components/mui-chat'

export default function Home() {
  return (
    <Box sx={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      <AnalysisSidebar />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          height: '100vh',
          overflow: 'hidden',
        }}
      >
        <MuiChat />
      </Box>
    </Box>
  )
}
