import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { WebRTCManager } from './webrtc';

describe('WebRTCManager', () => {
  let service: WebRTCManager;

  beforeEach(() => {
    service = new WebRTCManager();
  });

  afterEach(() => {
    service.leaveRoom();
  });

  describe('fetchICEServers', () => {
    it('should fallback to Google STUN servers if fetch fails', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValueOnce(new Error('Network error'));

      const config = await service.fetchICEServers();

      expect(config.iceServers).toEqual([
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' },
      ]);
    });

    it('should return backend ICE servers when endpoint returns 200 OK', async () => {
      const mockServers = [{ urls: 'stun:custom.stun.server:3478' }];
      vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce({
        ok: true,
        json: async () => ({ iceServers: mockServers }),
      } as Response);

      const config = await service.fetchICEServers();

      expect(config.iceServers).toEqual(mockServers);
    });
  });

  describe('Room Lifecycle', () => {
    it('should set isHost to true when creating a room', () => {
      service.createRoom('test-room-id', 'secret123');
      expect(service.isHost).toBe(true);
      expect(service.myPeerId).toBeTruthy();
    });

    it('should reset room state cleanly when leaveRoom is called', () => {
      service.createRoom('test-room-id');
      service.leaveRoom();

      expect(service.currentRoomId).toBe('');
      expect(service.isHost).toBe(false);
      expect(service.roomPassword).toBe('');
    });
  });
});
