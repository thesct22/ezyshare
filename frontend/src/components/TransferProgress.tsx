import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Button,
  LinearProgress,
  Chip,
} from '@mui/material';
import DownloadIcon from '@mui/icons-material/Download';
import CheckCircleOutlinedIcon from '@mui/icons-material/CheckCircleOutlined';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import type { TransferProgressState } from '../types';

interface TransferProgressProps {
  transfers: TransferProgressState[];
}

export const TransferProgress: React.FC<TransferProgressProps> = ({ transfers }) => {
  if (transfers.length === 0) return null;

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const formatSpeed = (bytesPerSec: number): string => {
    if (bytesPerSec <= 0) return '';
    return `${formatFileSize(bytesPerSec)}/s`;
  };

  return (
    <Card sx={{ mb: 4 }}>
      <CardContent sx={{ p: 3.5 }}>
        <Typography variant="h6" sx={{ color: '#1E293B', mb: 2.5, fontWeight: 700 }}>
          Active & Recent Transfers
        </Typography>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {transfers.map((item) => (
            <Box
              key={item.id}
              sx={{
                p: 2.5,
                bgcolor: '#F8FAFC',
                borderRadius: 3,
                border: '1px solid #E2E8F0',
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', justify: 'space-between', gap: 2, mb: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, overflow: 'hidden' }}>
                  <Box
                    sx={{
                      p: 1,
                      borderRadius: 2,
                      bgcolor: item.direction === 'upload' ? '#E8F5E9' : '#E0F2FE',
                      color: item.direction === 'upload' ? '#2E7D32' : '#0284C7',
                      display: 'flex',
                    }}
                  >
                    {item.direction === 'upload' ? <ArrowUpwardIcon fontSize="small" /> : <ArrowDownwardIcon fontSize="small" />}
                  </Box>

                  <Box sx={{ overflow: 'hidden' }}>
                    <Typography noWrap variant="subtitle2" sx={{ fontWeight: 700, color: '#0F172A' }}>
                      {item.fileName}
                    </Typography>
                    <Typography variant="caption" sx={{ color: '#64748B' }}>
                      {formatFileSize(item.bytesTransferred)} of {formatFileSize(item.fileSize)}
                      {item.speedBps > 0 && item.status === 'transferring' && (
                        <span style={{ color: '#2E7D32', fontFamily: 'monospace', marginLeft: 8 }}>
                          ({formatSpeed(item.speedBps)})
                        </span>
                      )}
                    </Typography>
                  </Box>
                </Box>

                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                  {item.status === 'completed' ? (
                    <Chip
                      icon={<CheckCircleOutlinedIcon style={{ color: '#16A34A' }} />}
                      label="Completed"
                      size="small"
                      sx={{ bgcolor: '#DCFCE7', color: '#15803D', fontWeight: 700 }}
                    />
                  ) : (
                    <Typography variant="caption" sx={{ fontWeight: 700, color: '#2E7D32' }}>
                      {item.percentage}%
                    </Typography>
                  )}

                  {item.fileUrl && (
                    <Button
                      href={item.fileUrl}
                      download={item.fileName}
                      variant="contained"
                      color="success"
                      size="small"
                      startIcon={<DownloadIcon />}
                    >
                      Save File
                    </Button>
                  )}
                </Box>
              </Box>

              {/* Progress Bar */}
              <LinearProgress
                variant="determinate"
                value={item.percentage}
                color={item.status === 'completed' ? 'success' : 'primary'}
                sx={{ height: 8, borderRadius: 4, bgcolor: '#E2E8F0' }}
              />
            </Box>
          ))}
        </Box>
      </CardContent>
    </Card>
  );
};
