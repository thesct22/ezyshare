import React from 'react';
import { Card, CardContent, Typography, Box } from '@mui/material';
import VerifiedUserOutlinedIcon from '@mui/icons-material/VerifiedUserOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';

export const SecurityBadge: React.FC = () => {
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' }, gap: 3, mb: 6 }}>
      <Card sx={{ height: '100%', borderLeft: '4px solid #2E7D32' }}>
        <CardContent sx={{ display: 'flex', items: 'flex-start', gap: 2, p: 3 }}>
          <Box
            sx={{
              p: 1.5,
              borderRadius: 3,
              bgcolor: '#E8F5E9',
              color: '#2E7D32',
              display: 'flex',
            }}
          >
            <VerifiedUserOutlinedIcon />
          </Box>
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#0F172A', mb: 0.5 }}>
              Zero-Knowledge Backend
            </Typography>
            <Typography variant="body2" sx={{ color: '#64748B' }}>
              Backend server has zero knowledge of file names, file sizes, or room passwords.
            </Typography>
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ height: '100%', borderLeft: '4px solid #10B981' }}>
        <CardContent sx={{ display: 'flex', items: 'flex-start', gap: 2, p: 3 }}>
          <Box
            sx={{
              p: 1.5,
              borderRadius: 3,
              bgcolor: '#ECFDF5',
              color: '#059669',
              display: 'flex',
            }}
          >
            <LockOutlinedIcon />
          </Box>
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#0F172A', mb: 0.5 }}>
              PBKDF2 & DTLS 1.2 Encrypted
            </Typography>
            <Typography variant="body2" sx={{ color: '#64748B' }}>
              Keys derived locally via PBKDF2 (100,000 iterations). WebRTC streams are DTLS encrypted.
            </Typography>
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ height: '100%', borderLeft: '4px solid #0284C7' }}>
        <CardContent sx={{ display: 'flex', items: 'flex-start', gap: 2, p: 3 }}>
          <Box
            sx={{
              p: 1.5,
              borderRadius: 3,
              bgcolor: '#E0F2FE',
              color: '#0284C7',
              display: 'flex',
            }}
          >
            <DnsOutlinedIcon />
          </Box>
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#0F172A', mb: 0.5 }}>
              STUN & Ephemeral TURN Relaying
            </Typography>
            <Typography variant="body2" sx={{ color: '#64748B' }}>
              Supports mobile 4G/5G carriers & symmetric NATs via ephemeral Coturn relay credentials.
            </Typography>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
};
