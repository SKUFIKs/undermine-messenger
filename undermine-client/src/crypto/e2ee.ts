import { x25519 } from '@noble/curves/ed25519';
import { chacha20poly1305 } from '@noble/ciphers/chacha';
import type { KeyPair } from '../types';

// Convert Uint8Array to Hex string
export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

// Convert Hex string to Uint8Array
export function hexToBytes(hex: string): Uint8Array {
  const clean = hex.trim();
  if (clean.length % 2 !== 0) {
    throw new Error('Invalid hex length');
  }
  const bytes = new Uint8Array(clean.length / 2);
  for (let i = 0; i < clean.length; i += 2) {
    bytes[i / 2] = parseInt(clean.substring(i, i + 2), 16);
  }
  return bytes;
}

// Generate random 16-byte ID
export function generateRandomId(): Uint8Array {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return bytes;
}

// Generate random 12-byte nonce for ChaCha20-Poly1305
export function generateNonce(): Uint8Array {
  const nonce = new Uint8Array(12);
  crypto.getRandomValues(nonce);
  return nonce;
}

// Generate Curve25519 keypair
export function generateKeyPair(): KeyPair {
  const privateKey = x25519.utils.randomPrivateKey();
  const publicKey = x25519.getPublicKey(privateKey);
  return {
    privateKey,
    publicKey,
    privateKeyHex: bytesToHex(privateKey),
    publicKeyHex: bytesToHex(publicKey),
  };
}

// Compute Diffie-Hellman Shared Secret (Curve25519)
export function computeSharedKey(myPrivateKey: Uint8Array, theirPublicKey: Uint8Array): Uint8Array {
  return x25519.getSharedSecret(myPrivateKey, theirPublicKey);
}

// Encrypt plaintext using ChaCha20-Poly1305 AEAD
export function encryptMessage(
  plaintext: string,
  sharedKey: Uint8Array
): { nonce: Uint8Array; ciphertext: Uint8Array } {
  const nonce = generateNonce();
  const encoder = new TextEncoder();
  const plaintextBytes = encoder.encode(plaintext);
  const cipher = chacha20poly1305(sharedKey, nonce);
  const ciphertext = cipher.encrypt(plaintextBytes);
  return { nonce, ciphertext };
}

// Decrypt ciphertext using ChaCha20-Poly1305 AEAD
export function decryptMessage(
  nonce: Uint8Array,
  ciphertext: Uint8Array,
  sharedKey: Uint8Array
): string {
  const cipher = chacha20poly1305(sharedKey, nonce);
  const decryptedBytes = cipher.decrypt(ciphertext);
  const decoder = new TextDecoder();
  return decoder.decode(decryptedBytes);
}

// Encode binary Message Packet (0x01)
export function encodeMessagePacket(
  msgId: Uint8Array,
  recipientId: Uint8Array,
  nonce: Uint8Array,
  payload: Uint8Array
): Uint8Array {
  const buffer = new Uint8Array(45 + payload.length);
  buffer[0] = 0x01; // PacketTypeMessage
  buffer.set(msgId.subarray(0, 16), 1);
  buffer.set(recipientId.subarray(0, 16), 17);
  buffer.set(nonce.subarray(0, 12), 33);
  buffer.set(payload, 45);
  return buffer;
}

// Decode binary Message Packet (0x01)
export function decodeMessagePacket(data: Uint8Array): {
  msgId: Uint8Array;
  recipientId: Uint8Array;
  nonce: Uint8Array;
  payload: Uint8Array;
} | null {
  if (data.length < 45 || data[0] !== 0x01) return null;
  return {
    msgId: data.subarray(1, 17),
    recipientId: data.subarray(17, 33),
    nonce: data.subarray(33, 45),
    payload: data.subarray(45),
  };
}

// Decode Ack Packet (0x02)
export function decodeAckPacket(data: Uint8Array): {
  msgId: Uint8Array;
  status: number;
} | null {
  if (data.length < 18 || data[0] !== 0x02) return null;
  return {
    msgId: data.subarray(1, 17),
    status: data[17],
  };
}

// Generate a random 32-byte group key for ChaCha20-Poly1305
export function generateGroupKey(): Uint8Array {
  const key = new Uint8Array(32);
  crypto.getRandomValues(key);
  return key;
}

// Encode binary Group Message Packet (0x05)
// Layout: type(1) + senderId(16) + groupId(16) + msgId(16) + nonce(12) + payload
export function encodeGroupMessagePacket(
  senderId: Uint8Array,
  groupId: Uint8Array,
  msgId: Uint8Array,
  nonce: Uint8Array,
  payload: Uint8Array
): Uint8Array {
  const buffer = new Uint8Array(61 + payload.length);
  buffer[0] = 0x05; // PacketTypeGroupMessage
  buffer.set(senderId.subarray(0, 16), 1);
  buffer.set(groupId.subarray(0, 16), 17);
  buffer.set(msgId.subarray(0, 16), 33);
  buffer.set(nonce.subarray(0, 12), 49);
  buffer.set(payload, 61);
  return buffer;
}

// Decode binary Group Message Packet (0x05)
export function decodeGroupMessagePacket(data: Uint8Array): {
  senderId: Uint8Array;
  groupId: Uint8Array;
  msgId: Uint8Array;
  nonce: Uint8Array;
  payload: Uint8Array;
} | null {
  if (data.length < 61 || data[0] !== 0x05) return null;
  return {
    senderId: data.subarray(1, 17),
    groupId: data.subarray(17, 33),
    msgId: data.subarray(33, 49),
    nonce: data.subarray(49, 61),
    payload: data.subarray(61),
  };
}
