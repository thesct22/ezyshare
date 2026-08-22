import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  TextField,
  Button,
  Tabs,
  Tab,
  FormControlLabel,
  Checkbox,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  IconButton,
  Alert,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CheckIcon from '@mui/icons-material/Check';
import QrCode2Icon from '@mui/icons-material/QrCode2';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import CloseIcon from '@mui/icons-material/Close';
import MeetingRoomIcon from '@mui/icons-material/MeetingRoom';
import { QRCodeSVG } from 'qrcode.react';
import type { ConnectionStatus } from '../types';

interface RoomCardProps {
  myPeerId: string;
  currentRoomId: string;
  onCreateRoom: (customRoomId?: string, password?: string) => void;
  onJoinRoom: (roomId: string, password?: string) => void;
  onLeaveRoom: () => void;
  status: ConnectionStatus;
}

export const RoomCard: React.FC<RoomCardProps> = ({
  currentRoomId,
  onCreateRoom,
  onJoinRoom,
  onLeaveRoom,
  status,
}) => {
  const [tabIndex, setTabIndex] = useState(0);
  const [customIDInput, setCustomIDInput] = useState('');
  const [joinRoomIDInput, setJoinRoomIDInput] = useState('');
  const [passwordInput, setPasswordInput] = useState('');
  const [usePassword, setUsePassword] = useState(false);
  const [copied, setCopied] = useState(false);
  const [qrOpen, setQrOpen] = useState(false);

  useEffect(() => {
    const hash = window.location.hash;
    if (hash && hash.includes('room=')) {
      const roomIdFromUrl = hash.split('room=')[1];
      if (roomIdFromUrl) {
        setJoinRoomIDInput(roomIdFromUrl);
        setTabIndex(1);
      }
    }
  }, []);

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onCreateRoom(customIDInput.trim() || undefined, usePassword ? passwordInput.trim() : undefined);
  };

  const handleJoinSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (joinRoomIDInput.trim()) {
      onJoinRoom(joinRoomIDInput.trim(), passwordInput.trim() || undefined);
    }
  };

  const getShareUrl = () => {
    const roomId = currentRoomId || customIDInput.trim() || 'room-demo';
    return `${window.location.origin}${window.location.pathname}#room=${roomId}`;
  };

  const handleCopyLink = () => {
    navigator.clipboard.writeText(getShareUrl());
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Card sx={{ mb: 4, overflow: 'visible' }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: '#F8FAFC', px: 2, pt: 1 }}>
        <Tabs
          value={tabIndex}
          onChange={(_, val) => setTabIndex(val)}
          textColor="primary"
          indicatorColor="primary"
          sx={{
            '& .MuiTab-root': {
              fontWeight: 600,
              fontSize: '0.95rem',
            },
          }}
        >
          <Tab icon={<MeetingRoomIcon />} iconPosition="start" label="Create Sharing Room" />
          <Tab icon={<LockOutlinedIcon />} iconPosition="start" label="Join Existing Room" />
        </Tabs>
      </Box>

      <CardContent sx={{ p: 3.5 }}>
        {/* TAB 0: CREATE ROOM */}
        {tabIndex === 0 && (
          <form onSubmit={handleCreateSubmit}>
            <Typography variant="h6" sx={{ color: '#1E293B', mb: 1, fontWeight: 700 }}>
              Create a Zero-Knowledge Sharing Room
            </Typography>
            <Typography variant="body2" sx={{ color: '#64748B', mb: 3 }}>
              Choose a custom room name or leave empty for an auto-generated UUID. Set an optional password for end-to-end zero-knowledge P2P authentication.
            </Typography>

            <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, gap: 2, mb: 2.5 }}>
              <TextField
                fullWidth
                label="Custom Room ID (Optional)"
                placeholder="e.g. my-awesome-room"
                value={customIDInput}
                onChange={(e) => setCustomIDInput(e.target.value)}
                disabled={status === 'in_room' || status === 'p2p_connected'}
                helperText="4–64 characters (letters, numbers, hyphens)"
              />

              {usePassword && (
                <TextField
                  fullWidth
                  type="password"
                  label="Room Password"
                  placeholder="Enter secret room password"
                  value={passwordInput}
                  onChange={(e) => setPasswordInput(e.target.value)}
                  disabled={status === 'in_room' || status === 'p2p_connected'}
                />
              )}
            </Box>

            <Box sx={{ display: 'flex', alignItems: 'center', justify: 'space-between', flexWrap: 'wrap', gap: 2, mb: 3 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={usePassword}
                    onChange={(e) => setUsePassword(e.target.checked)}
                    color="primary"
                  />
                }
                label={
                  <Typography variant="body2" sx={{ fontWeight: 600, color: '#334155' }}>
                    Protect Room with Password (Zero-Knowledge PBKDF2)
                  </Typography>
                }
              />

              <Button
                type="submit"
                variant="contained"
                size="large"
                disabled={status === 'in_room' || status === 'p2p_connected'}
                startIcon={<MeetingRoomIcon />}
              >
                {status === 'in_room' || status === 'p2p_connected' ? 'Room Active' : 'Create Room'}
              </Button>

              {(status === 'in_room' || status === 'p2p_connected') && tabIndex === 0 && (
                <Button
                  variant="outlined"
                  color="error"
                  size="large"
                  onClick={onLeaveRoom}
                  startIcon={<CloseIcon />}
                >
                  Close Room
                </Button>
              )}
            </Box>

            {/* Room Active Details & Sharing Controls */}
            {currentRoomId && (
              <Alert severity="success" sx={{ borderRadius: 3, bgcolor: '#ECFDF5', border: '1px solid #A7F3D0' }}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#065F46' }}>
                    Room Active! Share this link or QR Code with your receiver:
                  </Typography>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
                    <Box
                      sx={{
                        bgcolor: '#FFFFFF',
                        px: 2,
                        py: 1,
                        borderRadius: 2,
                        border: '1px solid #A7F3D0',
                        fontFamily: 'monospace',
                        fontWeight: 700,
                        color: '#047857',
                        flex: 1,
                      }}
                    >
                      {getShareUrl()}
                    </Box>

                    <Button
                      variant="contained"
                      color="secondary"
                      size="small"
                      onClick={handleCopyLink}
                      startIcon={copied ? <CheckIcon /> : <ContentCopyIcon />}
                    >
                      {copied ? 'Copied Link!' : 'Copy Link'}
                    </Button>

                    <Button
                      variant="outlined"
                      color="primary"
                      size="small"
                      onClick={() => setQrOpen(true)}
                      startIcon={<QrCode2Icon />}
                    >
                      Generate QR Code
                    </Button>
                  </Box>
                </Box>
              </Alert>
            )}
          </form>
        )}

        {/* TAB 1: JOIN ROOM */}
        {tabIndex === 1 && (
          <form onSubmit={handleJoinSubmit}>
            <Typography variant="h6" sx={{ color: '#1E293B', mb: 1, fontWeight: 700 }}>
              Join Room to Inspect & Download Shared Files
            </Typography>
            <Typography variant="body2" sx={{ color: '#64748B', mb: 3 }}>
              Enter the Room ID and Password provided by the Sender to establish a direct WebRTC P2P channel.
            </Typography>

            <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, gap: 2, mb: 3 }}>
              <TextField
                fullWidth
                label="Room ID"
                placeholder="e.g. my-awesome-room or room-x89f2a"
                value={joinRoomIDInput}
                onChange={(e) => setJoinRoomIDInput(e.target.value)}
                disabled={status === 'p2p_connected'}
                required
              />

              <TextField
                fullWidth
                type="password"
                label="Room Password (if protected)"
                placeholder="Enter password"
                value={passwordInput}
                onChange={(e) => setPasswordInput(e.target.value)}
                disabled={status === 'p2p_connected'}
              />

              <Button
                type="submit"
                variant="contained"
                size="large"
                disabled={!joinRoomIDInput.trim() || status === 'p2p_connected' || status === 'authenticating'}
                endIcon={<ArrowForwardIcon />}
                sx={{ minWidth: 160 }}
              >
                {status === 'p2p_connected' ? 'Connected' : 'Join Room'}
              </Button>
              
              {status === 'p2p_connected' && tabIndex === 1 && (
                <Button
                  variant="outlined"
                  color="error"
                  size="large"
                  onClick={onLeaveRoom}
                  startIcon={<CloseIcon />}
                >
                  Leave Room
                </Button>
              )}
            </Box>

            {status === 'auth_failed' && (
              <Alert severity="error" sx={{ borderRadius: 3, mt: 2 }}>
                Authentication Failed: The room password you entered is incorrect. Please check the password and try again.
              </Alert>
            )}

            {status === 'join_error' && (
              <Alert severity="error" sx={{ borderRadius: 3, mt: 2 }}>
                Join Failed: The room has reached maximum capacity or does not exist.
              </Alert>
            )}
          </form>
        )}

        {/* QR CODE DIALOG */}
        <Dialog open={qrOpen} onClose={() => setQrOpen(false)} maxWidth="xs" fullWidth>
          <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', pb: 1 }}>
            <Typography variant="h6" sx={{ fontWeight: 700, color: '#1B5E20' }}>
              Scan QR Code to Join Room
            </Typography>
            <IconButton onClick={() => setQrOpen(false)}>
              <CloseIcon />
            </IconButton>
          </DialogTitle>
          <DialogContent sx={{ textAlign: 'center', py: 4 }}>
            <Box
              sx={{
                p: 3,
                bgcolor: '#FFFFFF',
                borderRadius: 4,
                display: 'inline-block',
                border: '2px solid #A5D6A7',
                boxShadow: '0 8px 24px rgba(46, 125, 50, 0.12)',
                mb: 2,
              }}
            >
              <QRCodeSVG value={getShareUrl()} size={220} level="H" includeMargin />
            </Box>
            <Typography variant="body2" sx={{ color: '#475569', fontWeight: 500 }}>
              Scan with your mobile camera or barcode app to open room directly on iOS/Android.
            </Typography>
          </DialogContent>
          <DialogActions sx={{ p: 2, pt: 0 }}>
            <Button fullWidth variant="contained" onClick={() => setQrOpen(false)}>
              Close
            </Button>
          </DialogActions>
        </Dialog>
      </CardContent>
    </Card>
  );
};
