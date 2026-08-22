import React from 'react';
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TransferProgress } from './TransferProgress';
import type { TransferProgressState } from '../types';

describe('TransferProgress Component', () => {
  it('returns null when transfers list is empty', () => {
    const { container } = render(<TransferProgress transfers={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders transferring progress state', () => {
    const mockTransfers: TransferProgressState[] = [
      {
        id: 'transfer-1',
        fileName: 'report.pdf',
        fileSize: 1048576, // 1 MB
        bytesTransferred: 524288, // 512 KB
        percentage: 50,
        speedBps: 262144, // 256 KB/s
        direction: 'upload',
        status: 'transferring',
      },
    ];

    render(<TransferProgress transfers={mockTransfers} />);

    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByText('Active & Recent Transfers')).toBeInTheDocument();
  });

  it('renders completed state with Save File download button', () => {
    const mockTransfers: TransferProgressState[] = [
      {
        id: 'transfer-2',
        fileName: 'photo.jpg',
        fileSize: 204800,
        bytesTransferred: 204800,
        percentage: 100,
        speedBps: 0,
        direction: 'download',
        status: 'completed',
        fileUrl: 'blob:mock-file-url',
      },
    ];

    render(<TransferProgress transfers={mockTransfers} />);

    expect(screen.getByText('photo.jpg')).toBeInTheDocument();
    expect(screen.getByText('Completed')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /save file/i })).toHaveAttribute(
      'href',
      'blob:mock-file-url'
    );
  });
});
