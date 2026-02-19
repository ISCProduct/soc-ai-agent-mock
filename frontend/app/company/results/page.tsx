'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Box, Typography } from '@mui/material'

export default function CompanyResultsRedirectPage() {
  const router = useRouter()

  useEffect(() => {
    router.replace('/results')
  }, [router])

  return (
    <Box sx={{ p: 4 }}>
      <Typography variant="body2" color="text.secondary">
        結果ページへ移動中です...
      </Typography>
    </Box>
  )
}
