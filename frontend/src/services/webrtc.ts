import type { SignalMessage, FileMetadata, TransferProgressState, ConnectionStatus } from '../types';
import { generateAuthHash, verifyAuthHash } from '../utils/crypto';

const CHUNK_SIZE = 64 * 1024; // 64 KB per chunk

export class WebRTCManager {
  private ws: WebSocket | null = null;
  private pc: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;
  private reconnectTimer: number | null = null;
  private isDisposed = false;
  private authFailed = false;
  public myPeerId: string;
  public currentRoomId: string = '';
  public roomPassword: string = '';
  public isHost: boolean = false;
  public targetPeerId: string = '';

  private onStatusChangeCB?: (status: ConnectionStatus) => void;
  private onTransferProgressCB?: (progress: TransferProgressState) => void;
  private onFileListReceivedCB?: (files: FileMetadata[]) => void;
  private onRoomCreatedCB?: (roomId: string) => void;

  public sharedFiles: Map<string, File> = new Map();

  // Receiving buffer state
  private rxMetadata: FileMetadata | null = null;
  private rxChunks: ArrayBuffer[] = [];
  private rxBytesReceived = 0;
  private rxLastSpeedCalcTime = 0;
  private rxLastBytesCalculated = 0;

  constructor(myPeerId?: string) {
    this.myPeerId = myPeerId || this.generatePeerId();
  }

  private generatePeerId(): string {
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    let id = '';
    for (let i = 0; i < 6; i++) {
      id += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return `ez-${id}`;
  }

  public setCallbacks(
    onStatusChange: (status: ConnectionStatus) => void,
    onTransferProgress: (progress: TransferProgressState) => void,
    onFileListReceived?: (files: FileMetadata[]) => void,
    onRoomCreated?: (roomId: string) => void
  ) {
    this.onStatusChangeCB = onStatusChange;
    this.onTransferProgressCB = onTransferProgress;
    this.onFileListReceivedCB = onFileListReceived;
    this.onRoomCreatedCB = onRoomCreated;
  }

  private getDefaultAPIUrl(): string {
    if (import.meta.env.VITE_API_URL) {
      return import.meta.env.VITE_API_URL;
    }
    if (import.meta.env.VITE_WS_URL) {
      return import.meta.env.VITE_WS_URL
        .replace(/^wss:/, 'https:')
        .replace(/^ws:/, 'http:')
        .replace(/\/ws\/?$/, '');
    }
    const protocol = window.location.protocol;
    const host = window.location.hostname;
    const isDevPort = window.location.port !== '' && window.location.port !== '8080';
    const targetPort = isDevPort ? '8080' : window.location.port;
    return `${protocol}//${host}${targetPort ? `:${targetPort}` : ''}`;
  }

  public async fetchICEServers(): Promise<RTCConfiguration> {
    try {
      const baseUrl = this.getDefaultAPIUrl();
      const response = await fetch(`${baseUrl}/api/v1/ice-servers`);
      if (response.ok) {
        const data = await response.json();
        return { iceServers: data.iceServers };
      }
    } catch (err) {
      console.warn('Failed to fetch custom ICE servers from backend, falling back to Google STUN:', err);
    }
    return {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' },
      ],
    };
  }

  public connectSignaling(wsUrl?: string) {
    this.isDisposed = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    const url = wsUrl || this.getDefaultWSUrl();
    this.onStatusChangeCB?.('connecting');

    try {
      const ws = new WebSocket(url);
      this.ws = ws;

      ws.onopen = () => {
        if (this.isDisposed) return;
        this.onStatusChangeCB?.('signaling_ready');
        if (this.currentRoomId) {
          if (this.isHost) {
            this.createRoom(this.currentRoomId, this.roomPassword);
          } else {
            this.joinRoom(this.currentRoomId, this.roomPassword);
          }
        }
      };

      ws.onmessage = async (event) => {
        if (this.isDisposed) return;
        try {
          const msg: SignalMessage = JSON.parse(event.data);
          await this.handleSignalMessage(msg);
        } catch (err) {
          console.error('Failed to parse signaling message:', err);
        }
      };

      ws.onclose = () => {
        if (this.isDisposed) return;
        this.onStatusChangeCB?.('disconnected');
        this.scheduleReconnect(url);
      };

      ws.onerror = (err) => {
        if (this.isDisposed) return;
        console.error('WebSocket error:', err);
        this.onStatusChangeCB?.('disconnected');
      };
    } catch (err) {
      if (!this.isDisposed) {
        console.error('WebSocket instantiation error:', err);
        this.scheduleReconnect(url);
      }
    }
  }

  private scheduleReconnect(url: string) {
    if (this.isDisposed) return;
    if (!this.reconnectTimer) {
      this.reconnectTimer = window.setTimeout(() => {
        if (!this.isDisposed) {
          this.connectSignaling(url);
        }
      }, 3000);
    }
  }

  private getDefaultWSUrl(): string {
    if (import.meta.env.VITE_WS_URL) {
      return import.meta.env.VITE_WS_URL;
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.hostname;
    // In dev/preview environments (e.g. port 5173, 5174, 3000), target backend port 8080
    const isDevPort = window.location.port !== '' && window.location.port !== '8080';
    const targetPort = isDevPort ? '8080' : window.location.port;
    return `${protocol}//${host}${targetPort ? `:${targetPort}` : ''}/ws`;
  }

  private sendSignal(msg: SignalMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  public createRoom(customRoomId?: string, password?: string) {
    this.isHost = true;
    this.authFailed = false;
    this.roomPassword = password || '';
    this.sendSignal({
      type: 'create_room',
      sender_id: this.myPeerId,
      room_id: customRoomId || '',
    });
  }

  public joinRoom(roomId: string, password?: string) {
    this.isHost = false;
    this.authFailed = false;
    this.currentRoomId = roomId;
    this.roomPassword = password || '';
    this.sendSignal({
      type: 'join_room',
      sender_id: this.myPeerId,
      room_id: roomId,
    });
  }

  public leaveRoom() {
    if (this.currentRoomId) {
      this.sendSignal({
        type: 'leave_room',
        sender_id: this.myPeerId,
        room_id: this.currentRoomId,
      });
    }
    this.currentRoomId = '';
    this.roomPassword = '';
    this.targetPeerId = '';
    this.isHost = false;
    this.closePeerConnection();
    this.onStatusChangeCB?.('signaling_ready');
  }

  private async handleSignalMessage(msg: SignalMessage) {
    switch (msg.type) {
      case 'room_created':
        this.currentRoomId = msg.room_id || '';
        this.onStatusChangeCB?.('in_room');
        this.onRoomCreatedCB?.(this.currentRoomId);
        break;

      case 'peer_joined':
        if (this.isHost) {
          this.targetPeerId = msg.sender_id;
          await this.initiateP2PConnection(msg.sender_id);
        }
        break;

      case 'offer':
        this.targetPeerId = msg.sender_id;
        await this.handleOffer(msg.sender_id, msg.payload);
        break;

      case 'answer':
        await this.handleAnswer(msg.payload);
        break;

      case 'candidate':
        await this.handleCandidate(msg.payload);
        break;

      case 'peer_left':
        if (msg.sender_id === this.targetPeerId) {
          this.closePeerConnection();
          this.targetPeerId = '';
          this.onStatusChangeCB?.(this.isHost ? 'in_room' : 'signaling_ready');
          if (!this.isHost) {
            this.currentRoomId = '';
          }
        }
        break;

      case 'error':
        console.error('Room signaling error:', msg.payload);
        if (msg.payload && (msg.payload.includes('maximum peer capacity') || msg.payload.includes('room full') || msg.payload.includes('not found'))) {
          this.onStatusChangeCB?.('join_error');
        } else {
          this.onStatusChangeCB?.('auth_failed');
        }
        break;
    }
  }

  private async initiateP2PConnection(targetPeerId: string) {
    const config = await this.fetchICEServers();
    this.createPeerConnection(config);

    this.dataChannel = this.pc!.createDataChannel('fileSharingChannel', { ordered: true });
    this.setupDataChannel(this.dataChannel);

    const offer = await this.pc!.createOffer();
    await this.pc!.setLocalDescription(offer);

    this.sendSignal({
      type: 'offer',
      sender_id: this.myPeerId,
      target_id: targetPeerId,
      room_id: this.currentRoomId,
      payload: offer,
    });
  }

  private async handleOffer(senderId: string, offer: RTCSessionDescriptionInit) {
    const config = await this.fetchICEServers();
    this.createPeerConnection(config);

    this.pc!.ondatachannel = (event) => {
      this.dataChannel = event.channel;
      this.setupDataChannel(this.dataChannel);
    };

    await this.pc!.setRemoteDescription(new RTCSessionDescription(offer));
    const answer = await this.pc!.createAnswer();
    await this.pc!.setLocalDescription(answer);

    this.sendSignal({
      type: 'answer',
      sender_id: this.myPeerId,
      target_id: senderId,
      room_id: this.currentRoomId,
      payload: answer,
    });
  }

  private async handleAnswer(answer: RTCSessionDescriptionInit) {
    if (this.pc) {
      await this.pc.setRemoteDescription(new RTCSessionDescription(answer));
    }
  }

  private async handleCandidate(candidate: RTCIceCandidateInit) {
    if (this.pc && candidate) {
      await this.pc.addIceCandidate(new RTCIceCandidate(candidate));
    }
  }

  private createPeerConnection(config: RTCConfiguration) {
    this.closePeerConnection();
    this.pc = new RTCPeerConnection(config);

    this.pc.onicecandidate = (event) => {
      if (event.candidate && this.targetPeerId) {
        this.sendSignal({
          type: 'candidate',
          sender_id: this.myPeerId,
          target_id: this.targetPeerId,
          room_id: this.currentRoomId,
          payload: event.candidate,
        });
      }
    };
  }

  private setupDataChannel(channel: RTCDataChannel) {
    channel.binaryType = 'arraybuffer';

    channel.onopen = async () => {
      if (!this.isHost) {
        this.onStatusChangeCB?.('authenticating');
        const authProof = await generateAuthHash(this.roomPassword, this.currentRoomId);
        channel.send(JSON.stringify({ type: 'auth_request', authProof }));
      }
    };

    channel.onmessage = async (event) => {
      await this.handleIncomingData(event.data);
    };

    channel.onclose = () => {
      if (!this.authFailed) {
        this.onStatusChangeCB?.('in_room');
      }
    };
  }

  private async handleIncomingData(data: string | ArrayBuffer) {
    if (typeof data === 'string') {
      try {
        const payload = JSON.parse(data);

        if (payload.type === 'auth_request' && this.isHost) {
          const isValid = await verifyAuthHash(payload.authProof, this.roomPassword, this.currentRoomId);
          if (isValid) {
            this.dataChannel?.send(JSON.stringify({ type: 'auth_success' }));
            this.onStatusChangeCB?.('p2p_connected');
            this.sendSharedFileList();
          } else {
            this.dataChannel?.send(JSON.stringify({ type: 'auth_failed', error: 'Invalid room password' }));
            this.closePeerConnection();
          }
        } else if (payload.type === 'auth_success' && !this.isHost) {
          this.onStatusChangeCB?.('p2p_connected');
        } else if (payload.type === 'auth_failed' && !this.isHost) {
          this.authFailed = true;
          this.onStatusChangeCB?.('auth_failed');
          this.closePeerConnection();
        } else if (payload.type === 'file_list') {
          this.onFileListReceivedCB?.(payload.files);
        } else if (payload.type === 'request_file_download' && this.isHost) {
          const file = this.sharedFiles.get(payload.fileId);
          if (file) {
            await this.streamFileToPeer(file);
          }
        } else if (payload.type === 'header') {
          this.rxMetadata = payload.metadata;
          this.rxChunks = [];
          this.rxBytesReceived = 0;
          this.rxLastSpeedCalcTime = Date.now();
          this.rxLastBytesCalculated = 0;

          this.onTransferProgressCB?.({
            id: this.rxMetadata!.id,
            fileName: this.rxMetadata!.name,
            fileSize: this.rxMetadata!.size,
            bytesTransferred: 0,
            percentage: 0,
            speedBps: 0,
            direction: 'download',
            status: 'transferring',
          });
        } else if (payload.type === 'done') {
          if (this.rxMetadata) {
            const blob = new Blob(this.rxChunks, { type: this.rxMetadata.type || 'application/octet-stream' });
            const fileUrl = URL.createObjectURL(blob);

            this.onTransferProgressCB?.({
              id: this.rxMetadata.id,
              fileName: this.rxMetadata.name,
              fileSize: this.rxMetadata.size,
              bytesTransferred: this.rxMetadata.size,
              percentage: 100,
              speedBps: 0,
              direction: 'download',
              status: 'completed',
              fileUrl: fileUrl,
            });

            const a = document.createElement('a');
            a.href = fileUrl;
            a.download = this.rxMetadata.name;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);

            this.rxMetadata = null;
            this.rxChunks = [];
          }
        }
      } catch (err) {
        console.error('DataChannel JSON parsing error:', err);
      }
    } else if (data instanceof ArrayBuffer) {
      if (!this.rxMetadata) return;

      this.rxChunks.push(data);
      this.rxBytesReceived += data.byteLength;

      const now = Date.now();
      const elapsed = (now - this.rxLastSpeedCalcTime) / 1000;
      let speedBps = 0;
      if (elapsed >= 0.5) {
        speedBps = (this.rxBytesReceived - this.rxLastBytesCalculated) / elapsed;
        this.rxLastSpeedCalcTime = now;
        this.rxLastBytesCalculated = this.rxBytesReceived;
      }

      const percentage = Math.min(100, Math.round((this.rxBytesReceived / this.rxMetadata.size) * 100));

      this.onTransferProgressCB?.({
        id: this.rxMetadata.id,
        fileName: this.rxMetadata.name,
        fileSize: this.rxMetadata.size,
        bytesTransferred: this.rxBytesReceived,
        percentage,
        speedBps,
        direction: 'download',
        status: 'transferring',
      });
    }
  }

  public addFileToShare(file: File): FileMetadata {
    const fileId = `file-${Date.now()}-${Math.random().toString(36).substr(2, 5)}`;
    this.sharedFiles.set(fileId, file);

    const meta: FileMetadata = {
      id: fileId,
      name: file.name,
      size: file.size,
      type: file.type,
    };

    if (this.dataChannel && this.dataChannel.readyState === 'open') {
      this.sendSharedFileList();
    }

    return meta;
  }

  public removeFileFromShare(fileId: string) {
    this.sharedFiles.delete(fileId);
    if (this.dataChannel && this.dataChannel.readyState === 'open') {
      this.sendSharedFileList();
    }
  }

  public sendSharedFileList() {
    const fileList: FileMetadata[] = [];
    this.sharedFiles.forEach((file, id) => {
      fileList.push({
        id,
        name: file.name,
        size: file.size,
        type: file.type,
      });
    });

    if (this.dataChannel && this.dataChannel.readyState === 'open') {
      this.dataChannel.send(JSON.stringify({ type: 'file_list', files: fileList }));
    }
  }

  public requestFileDownload(fileId: string) {
    if (this.dataChannel && this.dataChannel.readyState === 'open') {
      this.dataChannel.send(JSON.stringify({ type: 'request_file_download', fileId }));
    }
  }

  private async streamFileToPeer(file: File) {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') return;

    const fileId = `file-${Date.now()}`;
    const metadata: FileMetadata = {
      id: fileId,
      name: file.name,
      size: file.size,
      type: file.type,
    };

    this.dataChannel.send(JSON.stringify({ type: 'header', metadata }));

    this.onTransferProgressCB?.({
      id: fileId,
      fileName: file.name,
      fileSize: file.size,
      bytesTransferred: 0,
      percentage: 0,
      speedBps: 0,
      direction: 'upload',
      status: 'transferring',
    });

    const buffer = await file.arrayBuffer();
    let offset = 0;
    let lastCalcTime = Date.now();
    let lastBytes = 0;

    while (offset < buffer.byteLength) {
      const chunk = buffer.slice(offset, offset + CHUNK_SIZE);

      while (this.dataChannel.bufferedAmount > 1024 * 1024) {
        await new Promise((resolve) => setTimeout(resolve, 50));
      }

      this.dataChannel.send(chunk);
      offset += chunk.byteLength;

      const now = Date.now();
      const elapsed = (now - lastCalcTime) / 1000;
      let speedBps = 0;
      if (elapsed >= 0.5) {
        speedBps = (offset - lastBytes) / elapsed;
        lastCalcTime = now;
        lastBytes = offset;
      }

      const percentage = Math.min(100, Math.round((offset / file.size) * 100));

      this.onTransferProgressCB?.({
        id: fileId,
        fileName: file.name,
        fileSize: file.size,
        bytesTransferred: offset,
        percentage,
        speedBps,
        direction: 'upload',
        status: offset >= file.size ? 'completed' : 'transferring',
      });
    }

    this.dataChannel.send(JSON.stringify({ type: 'done', id: fileId }));
  }

  public closePeerConnection() {
    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }
    if (this.pc) {
      this.pc.close();
      this.pc = null;
    }
  }

  public disconnectAll() {
    this.isDisposed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.closePeerConnection();

    if (this.ws) {
      const socket = this.ws;
      this.ws = null;

      // Detach event listeners so React StrictMode double-unmount in DEV mode doesn't log errors
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;

      if (socket.readyState === WebSocket.CONNECTING) {
        socket.onopen = () => {
          socket.close();
        };
        // Also catch errors so they don't bubble up unnecessarily
        socket.onerror = () => {};
      } else {
        try {
          socket.close();
        } catch (_) {}
      }
    }
  }
}
