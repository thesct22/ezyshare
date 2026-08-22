// Zero-Knowledge Client-Side Key Derivation & Auth Utilities (PBKDF2-HMAC-SHA256)

export async function deriveKeyFromPassword(password: string, roomId: string): Promise<CryptoKey> {
  const encoder = new TextEncoder();
  const passwordBytes = encoder.encode(password);
  const saltBytes = encoder.encode(`ezyshare-salt-${roomId}`);

  // Import raw password
  const baseKey = await window.crypto.subtle.importKey(
    'raw',
    passwordBytes,
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );

  // Derive 256-bit AES-GCM key using 100,000 iterations of PBKDF2
  return window.crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: saltBytes,
      iterations: 100000,
      hash: 'SHA-256',
    },
    baseKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

export async function generateAuthHash(password: string, roomId: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(`${password}:${roomId}:ezyshare-auth-proof`);
  const hashBuffer = await window.crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
}

export async function verifyAuthHash(
  authProof: string,
  password: string,
  roomId: string
): Promise<boolean> {
  const expectedHash = await generateAuthHash(password, roomId);
  return authProof === expectedHash;
}
