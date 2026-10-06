export type MessageStatus = 'sending' | 'sent' | 'delivered' | 'read' | 'failed';

export interface User {
  id: string; // 32 hex chars
  username: string;
  isOnline: boolean;
  publicKey?: string;
  lastSeen?: string;
}

export interface Message {
  id: string; // 32 hex chars
  chatId: string;
  senderId: string;
  content: string;
  status: MessageStatus;
  isOutgoing: boolean;
  createdAt: string; // ISO format or timestamp
}

export interface Chat {
  id: string;
  title: string;
  peerUser: User;
  lastMessage?: Message;
  unreadCount: number;
  updatedAt: string;
}

export interface KeyPair {
  privateKey: Uint8Array;
  publicKey: Uint8Array;
  publicKeyHex: string;
  privateKeyHex: string;
}
