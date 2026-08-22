import React, { useState, useRef } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  IconButton,
  Chip,
} from '@mui/material';
import CloudUploadOutlinedIcon from '@mui/icons-material/CloudUploadOutlined';
import InsertDriveFileOutlinedIcon from '@mui/icons-material/InsertDriveFileOutlined';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import type { FileMetadata, ConnectionStatus } from '../types';

interface FileDropzoneProps {
  onAddFiles: (files: File[]) => void;
  onRemoveFile: (fileId: string) => void;
  sharedFilesList: FileMetadata[];
  status: ConnectionStatus;
  isHost: boolean;
}

export const FileDropzone: React.FC<FileDropzoneProps> = ({
  onAddFiles,
  onRemoveFile,
  sharedFilesList,
  status,
  isHost,
}) => {
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isHost && status !== 'p2p_connected') return null;

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = () => {
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      onAddFiles(Array.from(e.dataTransfer.files));
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      onAddFiles(Array.from(e.target.files));
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <Card sx={{ mb: 4 }}>
      <CardContent sx={{ p: 3.5 }}>
        <Typography variant="h6" sx={{ color: '#1E293B', mb: 1, fontWeight: 700 }}>
          {isHost ? 'Host Shared Files Dropzone' : 'Files Shared by Host'}
        </Typography>
        <Typography variant="body2" sx={{ color: '#64748B', mb: 3 }}>
          {isHost
            ? 'Drag & drop files below to make them available to authenticated peers in this room. Files stream directly device-to-device.'
            : 'Authenticated Room: You can inspect and download any file below.'}
        </Typography>

        {isHost && (
          <>
            <input
              type="file"
              multiple
              ref={fileInputRef}
              onChange={handleFileChange}
              style={{ display: 'none' }}
            />

            <Box
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              sx={{
                border: '2px dashed',
                borderColor: isDragging ? '#2E7D32' : '#CBD5E1',
                bgcolor: isDragging ? '#E8F5E9' : '#F8FAFC',
                borderRadius: 4,
                p: 5,
                textAlign: 'center',
                cursor: 'pointer',
                transition: 'all 0.2s ease',
                '&:hover': {
                  borderColor: '#2E7D32',
                  bgcolor: '#F4FBF7',
                },
                mb: 3,
              }}
            >
              <Box
                sx={{
                  width: 64,
                  height: 64,
                  borderRadius: 4,
                  bgcolor: '#E8F5E9',
                  color: '#2E7D32',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  mx: 'auto',
                  mb: 2,
                }}
              >
                <CloudUploadOutlinedIcon sx={{ fontSize: 36 }} />
              </Box>
              <Typography variant="h6" sx={{ color: '#1E293B', fontWeight: 600, mb: 0.5 }}>
                Drag & Drop files here or click to browse
              </Typography>
              <Typography variant="body2" sx={{ color: '#64748B', mb: 1.5 }}>
                Supports any file type, unlimited file size
              </Typography>
              <Chip
                icon={<LockOutlinedIcon style={{ color: '#2E7D32' }} />}
                label="Zero Server Storage: Files stay on your device"
                size="small"
                sx={{ bgcolor: '#E8F5E9', color: '#1B5E20', fontWeight: 600 }}
              />
            </Box>
          </>
        )}

        {/* List of Files Currently Offered */}
        {sharedFilesList.length > 0 && (
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#334155', mb: 1.5 }}>
              Files Hosted in Room ({sharedFilesList.length}):
            </Typography>

            <List sx={{ bgcolor: '#F8FAFC', borderRadius: 3, border: '1px solid #E2E8F0', p: 1 }}>
              {sharedFilesList.map((file) => (
                <ListItem
                  key={file.id}
                  secondaryAction={
                    isHost && (
                      <IconButton edge="end" color="error" onClick={() => onRemoveFile(file.id)}>
                        <DeleteOutlinedIcon />
                      </IconButton>
                    )
                  }
                >
                  <ListItemIcon>
                    <Box
                      sx={{
                        width: 40,
                        height: 40,
                        borderRadius: 2,
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
                    primary={<Typography variant="body1" sx={{ fontWeight: 600, color: '#0F172A' }}>{file.name}</Typography>}
                    secondary={formatFileSize(file.size)}
                  />
                </ListItem>
              ))}
            </List>
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
