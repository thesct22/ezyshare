import React, { useState, useEffect, useRef } from 'react';
import { ThemeProvider, CssBaseline, Container, Box, Typography } from '@mui/material';
import ShieldIcon from '@mui/icons-material/Shield';
import { leafGreenTheme } from './theme';
import { Navbar } from './components/Navbar';
import { RoomCard } from './components/RoomCard';
import { FileDropzone } from './components/FileDropzone';
import { SharedFilesList } from './components/SharedFilesList';
import { TransferProgress } from './components/TransferProgress';
import { SecurityBadge } from './components/SecurityBadge';
import { WebRTCManager } from './services/webrtc';
import type { ConnectionStatus, FileMetadata, TransferProgressState } from './types';

export const App: React.FC = () => {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [currentRoomId, setCurrentRoomId] = useState('');
  const [sharedFilesList, setSharedFilesList] = useState<FileMetadata[]>([]);
  const [transfers, setTransfers] = useState<TransferProgressState[]>([]);
  const rtcRef = useRef<WebRTCManager | null>(null);

  useEffect(() => {
    const rtc = new WebRTCManager();
    rtcRef.current = rtc;

    rtc.setCallbacks(
      (newStatus) => setStatus(newStatus),
      (progress) => {
        setTransfers((prev) => {
          const index = prev.findIndex((t) => t.id === progress.id);
          if (index >= 0) {
            const updated = [...prev];
            updated[index] = progress;
            return updated;
          }
          return [progress, ...prev];
        });
      },
      (files) => setSharedFilesList(files),
      (roomId) => setCurrentRoomId(roomId)
    );

    rtc.connectSignaling();

    return () => {
      rtc.disconnectAll();
    };
  }, []);

  const handleCreateRoom = (customRoomId?: string, password?: string) => {
    if (rtcRef.current) {
      rtcRef.current.createRoom(customRoomId, password);
    }
  };

  const handleJoinRoom = (roomId: string, password?: string) => {
    if (rtcRef.current) {
      rtcRef.current.joinRoom(roomId, password);
    }
  };

  const handleLeaveRoom = () => {
    if (rtcRef.current) {
      rtcRef.current.leaveRoom();
      setCurrentRoomId('');
      setSharedFilesList([]);
      setTransfers([]);
    }
  };

  const handleRemovePeer = () => {
    if (rtcRef.current) {
      rtcRef.current.removePeer();
    }
  };

  const handleAddFiles = (files: File[]) => {
    if (rtcRef.current) {
      const newMetaList: FileMetadata[] = [];
      files.forEach((file) => {
        const meta = rtcRef.current!.addFileToShare(file);
        newMetaList.push(meta);
      });
      setSharedFilesList((prev) => [...prev, ...newMetaList]);
    }
  };

  const handleRemoveFile = (fileId: string) => {
    if (rtcRef.current) {
      rtcRef.current.removeFileFromShare(fileId);
      setSharedFilesList((prev) => prev.filter((f) => f.id !== fileId));
    }
  };

  const handleDownloadFile = (fileId: string) => {
    if (rtcRef.current) {
      rtcRef.current.requestFileDownload(fileId);
    }
  };

  const myPeerId = rtcRef.current?.myPeerId || 'ez-init...';
  const isHost = rtcRef.current?.isHost ?? true;

  return (
    <ThemeProvider theme={leafGreenTheme}>
      <CssBaseline />
      <Box sx={{ minHeight: '100vh', bgcolor: '#F8FAFC', color: '#0F172A', pb: 10 }}>
        <Navbar status={status} myPeerId={myPeerId} />

        <Container maxWidth="lg">
          {/* Hero Headline */}
          <Box sx={{ textAlign: 'center', mb: 6, mt: 2 }}>
            <Box
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 1,
                bgcolor: '#E8F5E9',
                color: '#2E7D32',
                px: 2,
                py: 0.75,
                borderRadius: 4,
                fontSize: '0.85rem',
                fontWeight: 700,
                mb: 2,
                border: '1px solid #A5D6A7',
              }}
            >
              <ShieldIcon fontSize="small" />
              <span>Zero-Knowledge Peer-to-Peer Protocol (Web & Mobile)</span>
            </Box>

            <Typography variant="h3" component="h1" sx={{ fontWeight: 800, color: '#0F172A', letterSpacing: '-0.03em', mb: 2 }}>
              Share Files Directly Device-to-Device.{' '}
              <span style={{ color: '#2E7D32' }}>Zero Cloud Uploads.</span>
            </Typography>

            <Typography variant="h6" sx={{ color: '#475569', fontWeight: 400, maxW: '700px', mx: 'auto' }}>
              Create a custom password-protected room or generate a QR code for instant mobile file transfers. The backend never sees your files or passwords.
            </Typography>
          </Box>

          {/* Room Controls (Create/Join/QR Code) */}
          <RoomCard
            myPeerId={myPeerId}
            currentRoomId={currentRoomId}
            onCreateRoom={handleCreateRoom}
            onJoinRoom={handleJoinRoom}
            onLeaveRoom={handleLeaveRoom}
            onRemovePeer={handleRemovePeer}
            status={status}
          />

          {/* Host File Upload Dropzone */}
          <FileDropzone
            onAddFiles={handleAddFiles}
            onRemoveFile={handleRemoveFile}
            sharedFilesList={sharedFilesList}
            status={status}
            isHost={isHost}
          />

          {/* Receiver Shared Files List & One-Click Downloads */}
          <SharedFilesList
            files={sharedFilesList}
            onDownloadFile={handleDownloadFile}
            status={status}
            isHost={isHost}
          />

          {/* Active Transfers & Progress */}
          <TransferProgress transfers={transfers} />

          {/* Security Features */}
          <SecurityBadge />
        </Container>

        {/* Footer */}
        <Box
          component="footer"
          sx={{
            py: 4,
            borderTop: '1px solid #E2E8F0',
            bgcolor: '#FFFFFF',
            textAlign: 'center',
            color: '#64748B',
            fontSize: '0.85rem',
            mt: 'auto',
          }}
        >
          <Container maxWidth="lg">
            <Typography variant="body2" sx={{ color: '#64748B' }}>
              © {new Date().getFullYear()} EzyShare P2P Protocol. Built with Go, WebRTC, React & Material UI.
            </Typography>
          </Container>
        </Box>
      </Box>
    </ThemeProvider>
  );
};

export default App;
