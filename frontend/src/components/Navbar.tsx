import React from 'react';
import { AppBar, Toolbar, Typography, Box, Chip } from '@mui/material';
import ShareIcon from '@mui/icons-material/Share';
import ShieldOutlinedIcon from '@mui/icons-material/ShieldOutlined';
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import type { ConnectionStatus } from '../types';

interface NavbarProps {
  status: ConnectionStatus;
  myPeerId: string;
}

export const Navbar: React.FC<NavbarProps> = ({ status, myPeerId }) => {
  const getStatusChip = () => {
    switch (status) {
      case 'p2p_connected':
        return (
          <Chip
            icon={<ShieldOutlinedIcon style={{ color: '#16A34A' }} />}
            label="Direct P2P Connected"
            size="small"
            sx={{
              backgroundColor: '#DCFCE7',
              color: '#15803D',
              fontWeight: 600,
              border: '1px solid #86EFAC',
            }}
          />
        );
      case 'authenticating':
        return (
          <Chip
            icon={<LockOutlinedIcon style={{ color: '#D97706' }} />}
            label="P2P Authenticating..."
            size="small"
            sx={{
              backgroundColor: '#FEF3C7',
              color: '#B45309',
              fontWeight: 600,
              border: '1px solid #FDE68A',
            }}
          />
        );
      case 'in_room':
        return (
          <Chip
            icon={<DnsOutlinedIcon style={{ color: '#0284C7' }} />}
            label="In Room (Waiting Peer)"
            size="small"
            sx={{
              backgroundColor: '#E0F2FE',
              color: '#0369A1',
              fontWeight: 600,
              border: '1px solid #7DD3FC',
            }}
          />
        );
      case 'signaling_ready':
        return (
          <Chip
            label="Signal Broker Ready"
            size="small"
            sx={{
              backgroundColor: '#E8F5E9',
              color: '#2E7D32',
              fontWeight: 600,
              border: '1px solid #A5D6A7',
            }}
          />
        );
      case 'auth_failed':
        return (
          <Chip
            label="P2P Auth Failed (Wrong Pass)"
            size="small"
            color="error"
            variant="outlined"
            sx={{ fontWeight: 600 }}
          />
        );
      default:
        return (
          <Chip
            label="Connecting Broker..."
            size="small"
            variant="outlined"
            sx={{ color: '#64748B', borderColor: '#CBD5E1' }}
          />
        );
    }
  };

  return (
    <AppBar
      position="sticky"
      elevation={0}
      sx={{
        backgroundColor: '#FFFFFF',
        borderBottom: '1px solid #E2E8F0',
        color: '#0F172A',
        mb: 4,
      }}
    >
      <Toolbar sx={{ justifyContent: 'space-between', maxW: '1200px', width: '100%', mx: 'auto', px: 2 }}>
        {/* Brand Logo */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box
            sx={{
              width: 40,
              height: 40,
              borderRadius: 3,
              background: 'linear-gradient(135deg, #2E7D32 0%, #15803D 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#FFFFFF',
              boxShadow: '0 4px 12px rgba(46, 125, 50, 0.25)',
            }}
          >
            <ShareIcon />
          </Box>
          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography variant="h6" sx={{ fontWeight: 800, color: '#1B5E20', letterSpacing: '-0.02em' }}>
                EzyShare
              </Typography>
              <Chip
                label="Zero-Knowledge P2P"
                size="small"
                sx={{
                  backgroundColor: '#E8F5E9',
                  color: '#2E7D32',
                  fontSize: '0.65rem',
                  height: 20,
                  fontWeight: 700,
                }}
              />
            </Box>
            <Typography variant="caption" sx={{ color: '#64748B', display: { xs: 'none', sm: 'block' } }}>
              Encrypted Peer-to-Peer File Transfer (Web & Mobile)
            </Typography>
          </Box>
        </Box>

        {/* Status Indicators */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Typography
            variant="caption"
            sx={{
              display: { xs: 'none', md: 'block' },
              fontFamily: 'monospace',
              color: '#475569',
              bgcolor: '#F1F5F9',
              px: 1.5,
              py: 0.5,
              borderRadius: 2,
            }}
          >
            Peer ID: {myPeerId}
          </Typography>
          {getStatusChip()}
        </Box>
      </Toolbar>
    </AppBar>
  );
};
