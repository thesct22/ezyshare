export type MessageType =
  | 'join'
  | 'leave'
  | 'offer'
  | 'answer'
  | 'candidate'
  | 'create_room'
  | 'join_room'
  | 'leave_room'
  | 'peer_joined'
  | 'peer_left'
  | 'room_created'
  | 'error';

export interface SignalMessage {
  type: MessageType;
  sender_id: string;
  target_id?: string;
  room_id?: string;
  payload?: any;
}

export interface FileMetadata {
  id: string;
  name: string;
  size: number;
  type: string;
}

export interface TransferProgressState {
  id: string;
  fileName: string;
  fileSize: number;
  bytesTransferred: number;
  percentage: number;
  speedBps: number;
  direction: 'upload' | 'download';
  status: 'pending' | 'transferring' | 'completed' | 'failed' | 'cancelled';
  fileUrl?: string;
  error?: string;
}

export type ConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'signaling_ready'
  | 'in_room'
  | 'authenticating'
  | 'auth_failed'
  | 'join_error'
  | 'p2p_connected';

export interface ICEServerConfig {
  urls: string[];
  username?: string;
  credential?: string;
}
