import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { RoomCard } from './RoomCard';

describe('RoomCard Component', () => {
  const defaultProps = {
    myPeerId: 'peer-123',
    currentRoomId: '',
    onCreateRoom: vi.fn(),
    onJoinRoom: vi.fn(),
    onLeaveRoom: vi.fn(),
    status: 'disconnected' as const,
  };

  it('renders Create Room tab by default', () => {
    render(<RoomCard {...defaultProps} />);
    expect(screen.getByText('Create a Zero-Knowledge Sharing Room')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create room/i })).toBeInTheDocument();
  });

  it('calls onCreateRoom when form is submitted', () => {
    const onCreateRoom = vi.fn();
    render(<RoomCard {...defaultProps} onCreateRoom={onCreateRoom} />);

    const input = screen.getByLabelText(/custom room id/i);
    fireEvent.change(input, { target: { value: 'my-custom-room' } });

    const submitBtn = screen.getByRole('button', { name: /create room/i });
    fireEvent.click(submitBtn);

    expect(onCreateRoom).toHaveBeenCalledWith('my-custom-room', undefined);
  });

  it('switches to Join Room tab and submits room ID', () => {
    const onJoinRoom = vi.fn();
    render(<RoomCard {...defaultProps} onJoinRoom={onJoinRoom} />);

    const joinTab = screen.getByText('Join Existing Room');
    fireEvent.click(joinTab);

    expect(screen.getByText('Join Room to Inspect & Download Shared Files')).toBeInTheDocument();

    const input = screen.getByLabelText(/room id/i);
    fireEvent.change(input, { target: { value: 'target-room-456' } });

    const joinBtn = screen.getByRole('button', { name: /join room/i });
    fireEvent.click(joinBtn);

    expect(onJoinRoom).toHaveBeenCalledWith('target-room-456', undefined);
  });

  it('displays Leave/Close room button when connected and calls onLeaveRoom', () => {
    const onLeaveRoom = vi.fn();
    render(
      <RoomCard
        {...defaultProps}
        currentRoomId="active-room"
        status="in_room"
        onLeaveRoom={onLeaveRoom}
      />
    );

    const closeRoomBtn = screen.getByRole('button', { name: /close room/i });
    expect(closeRoomBtn).toBeInTheDocument();

    fireEvent.click(closeRoomBtn);
    expect(onLeaveRoom).toHaveBeenCalledTimes(1);
  });
});
