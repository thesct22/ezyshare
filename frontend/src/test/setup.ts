import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

// Mock window.URL.createObjectURL and revokeObjectURL
if (typeof window !== 'undefined') {
  window.URL.createObjectURL = vi.fn(() => 'blob:mock-url');
  window.URL.revokeObjectURL = vi.fn();
}

// Mock WebRTC RTCPeerConnection if missing in jsdom environment
if (typeof window !== 'undefined' && !window.RTCPeerConnection) {
  class MockRTCPeerConnection {
    createOffer = vi.fn().mockResolvedValue({ type: 'offer', sdp: 'mock-sdp' });
    createAnswer = vi.fn().mockResolvedValue({ type: 'answer', sdp: 'mock-sdp' });
    setLocalDescription = vi.fn().mockResolvedValue(undefined);
    setRemoteDescription = vi.fn().mockResolvedValue(undefined);
    addIceCandidate = vi.fn().mockResolvedValue(undefined);
    addTrack = vi.fn().mockReturnValue({ id: 'mock-sender' });
    createDataChannel = vi.fn().mockReturnValue({
      onopen: null,
      onmessage: null,
      onclose: null,
      onerror: null,
      send: vi.fn(),
      close: vi.fn(),
    });
    close = vi.fn();
    onicecandidate = null;
    ontrack = null;
    ondatachannel = null;
    onconnectionstatechange = null;
    connectionState = 'new';
  }

  // @ts-expect-error Mocking RTCPeerConnection for testing
  window.RTCPeerConnection = MockRTCPeerConnection;
}

// Mock WebSocket if missing in jsdom environment
if (typeof window !== 'undefined' && !window.WebSocket) {
  class MockWebSocket {
    static OPEN = 1;
    static CLOSED = 3;
    readyState = 1;
    onopen = null;
    onmessage = null;
    onclose = null;
    onerror = null;
    send = vi.fn();
    close = vi.fn();
  }

  // @ts-expect-error Mocking WebSocket for testing
  window.WebSocket = MockWebSocket;
}
