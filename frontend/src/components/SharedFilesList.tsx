import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Button,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  Alert,
} from '@mui/material';
import DownloadOutlinedIcon from '@mui/icons-material/DownloadOutlined';
import InsertDriveFileOutlinedIcon from '@mui/icons-material/InsertDriveFileOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import type { FileMetadata, ConnectionStatus } from '../types';

interface SharedFilesListProps {
  files: FileMetadata[];
  onDownloadFile: (fileId: string) => void;
  status: ConnectionStatus;
  isHost: boolean;
}

export const SharedFilesList: React.FC<SharedFilesListProps> = ({
  files,
  onDownloadFile,
  status,
  isHost,
}) => {
  if (isHost || status !== 'p2p_connected') return null;

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <Card sx={{ mb: 4, border: '2px solid #A5D6A7' }}>
      <CardContent sx={{ p: 3.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Typography variant="h6" sx={{ color: '#1B5E20', fontWeight: 700 }}>
              Available Shared Files ({files.length})
            </Typography>
            <Chip
              icon={<LockOutlinedIcon style={{ color: '#16A34A' }} />}
              label="E2E Authenticated P2P Stream"
              size="small"
              sx={{ bgcolor: '#DCFCE7', color: '#15803D', fontWeight: 600 }}
            />
          </Box>
        </Box>

        {files.length === 0 ? (
          <Alert severity="info" sx={{ borderRadius: 3 }}>
            Connected to Sender! The Host has not added any files to share yet.
          </Alert>
        ) : (
          <List sx={{ bgcolor: '#F8FAFC', borderRadius: 3, border: '1px solid #E2E8F0', p: 1 }}>
            {files.map((file) => (
              <ListItem
                key={file.id}
                secondaryAction={
                  <Button
                    variant="contained"
                    color="primary"
                    size="small"
                    startIcon={<DownloadOutlinedIcon />}
                    onClick={() => onDownloadFile(file.id)}
                  >
                    Download File
                  </Button>
                }
              >
                <ListItemIcon>
                  <Box
                    sx={{
                      width: 44,
                      height: 44,
                      borderRadius: 2.5,
                      bgcolor: '#E8F5E9',
                      color: '#2E7D32',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <InsertDriveFileOutlinedIcon />
                  </Box>
                </ListItemIcon>
                <ListItemText
                  primary={<Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#0F172A' }}>{file.name}</Typography>}
                  secondary={formatFileSize(file.size)}
                />
              </ListItem>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  );
};
