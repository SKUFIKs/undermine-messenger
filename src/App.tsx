import React, { useState, useEffect, useRef, useCallback } from 'react';
import type { Chat, Message, KeyPair } from './types';
import { AuthView } from './components/AuthView';
import { ChatView } from './components/ChatView';
import {
  generateKeyPair,
  computeSharedKey,
  encryptMessage,
  decryptMessage,
  encodeMessagePacket,
  decodeMessagePacket,
  decodeAckPacket,
  bytesToHex,
  hexToBytes,
  generateRandomId,
} from './crypto/e2ee';
import { Columns, Smartphone, X } from 'lucide-react';

interface AuthSession {
  token: string;
  userId: string;
  username: string;
  keyPair: KeyPair;
}

interface MessengerClientProps {
  initialUsername?: string;
  initialPassword?: string;
  storageKeyPrefix?: string;
  isDualView?: boolean;
  onToggleDualView?: () => void;
  serverUrl?: string;
  onSaveServerUrl?: (url: string) => void;
}

export const MessengerClient: React.FC<MessengerClientProps> = ({
  initialUsername,
  initialPassword,
  storageKeyPrefix = 'undermine_session_1',
  isDualView = false,
  onToggleDualView,
  serverUrl = '',
  onSaveServerUrl = () => {},
}) => {
  const [session, setSession] = useState<AuthSession | null>(() => {
    const saved = sessionStorage.getItem(storageKeyPrefix);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        const privateKey = hexToBytes(parsed.keyPair.privateKeyHex);
        const publicKey = hexToBytes(parsed.keyPair.publicKeyHex);
        return {
          token: parsed.token,
          userId: parsed.userId,
          username: parsed.username,
          keyPair: {
            privateKey,
            publicKey,
            privateKeyHex: parsed.keyPair.publicKeyHex,
            publicKeyHex: parsed.keyPair.publicKeyHex,
          },
        };
      } catch {
        return null;
      }
    }
    return null;
  });

  const [isLoading, setIsLoading] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);

  // Messenger state
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChatId, setActiveChatId] = useState<string | null>(null);
  const [chatMessages, setChatMessages] = useState<Record<string, Message[]>>({});
  const [isConnected, setIsConnected] = useState(false);
  const [availableUsers, setAvailableUsers] = useState<
    { user_id: string; username: string; public_key: string }[]
  >([]);

  // Cryptographic Shared Keys cache: contactUserId -> sharedKey (Uint8Array)
  const sharedKeysRef = useRef<Map<string, Uint8Array>>(new Map());
  // Contact public keys: contactUserId -> publicKeyHex
  const contactPublicKeysRef = useRef<Map<string, string>>(new Map());

  // WebSocket reference
  const wsRef = useRef<WebSocket | null>(null);

  // Auto-login with initial credentials if provided
  useEffect(() => {
    if (!session && initialUsername && initialPassword) {
      handleLogin(initialUsername, initialPassword);
    }
  }, [initialUsername, initialPassword]);

  // Fetch available directory of users
  const refreshDirectory = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/users/directory');
      if (res.ok) {
        const list = await res.json();
        setAvailableUsers(list);
      }
    } catch (e) {
      console.warn('Failed to fetch directory', e);
    }
  }, []);

  // Connect to WebSocket relay
  useEffect(() => {
    if (!session) {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setIsConnected(false);
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/ws?token=${session.token}`;

    const ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      setIsConnected(true);
    };

    ws.onclose = () => {
      setIsConnected(false);
    };

    ws.onerror = (err) => {
      console.error('[WS] Error:', err);
    };

    ws.onmessage = (event) => {
      if (!(event.data instanceof ArrayBuffer)) {
        return;
      }

      const data = new Uint8Array(event.data);
      if (data.length === 0) return;

      const packetType = data[0];

      if (packetType === 0x01) {
        // MessagePacket
        const packet = decodeMessagePacket(data);
        if (!packet) return;

        const msgIdHex = bytesToHex(packet.msgId);
        let decryptedText: string | null = null;
        let matchedSenderId: string | null = null;

        for (const [contactId, sharedKey] of sharedKeysRef.current.entries()) {
          try {
            decryptedText = decryptMessage(packet.nonce, packet.payload, sharedKey);
            matchedSenderId = contactId;
            break;
          } catch {
            // mismatch, continue
          }
        }

        if (!decryptedText) {
          fetch('/api/v1/users/directory')
            .then((r) => r.json())
            .then((users: { user_id: string; username: string; public_key: string }[]) => {
              for (const u of users) {
                if (u.user_id === session.userId) continue;
                try {
                  const peerPubKey = hexToBytes(u.public_key);
                  const sk = computeSharedKey(session.keyPair.privateKey, peerPubKey);
                  const text = decryptMessage(packet.nonce, packet.payload, sk);

                  sharedKeysRef.current.set(u.user_id, sk);
                  contactPublicKeysRef.current.set(u.user_id, u.public_key);

                  handleReceivedMessage(msgIdHex, u.user_id, u.username, text);
                  break;
                } catch {
                  // mismatch
                }
              }
            });
        } else if (matchedSenderId && decryptedText) {
          const contactTitle =
            chats.find((c) => c.id === matchedSenderId)?.title ||
            availableUsers.find((u) => u.user_id === matchedSenderId)?.username ||
            'Собеседник';
          handleReceivedMessage(msgIdHex, matchedSenderId, contactTitle, decryptedText);
        }
      } else if (packetType === 0x02) {
        // AckPacket
        const ack = decodeAckPacket(data);
        if (ack) {
          const ackMsgId = bytesToHex(ack.msgId);
          setChatMessages((prev) => {
            const next = { ...prev };
            for (const cId of Object.keys(next)) {
              next[cId] = next[cId].map((m) =>
                m.id === ackMsgId
                  ? { ...m, status: ack.status === 2 ? 'delivered' : 'sent' }
                  : m
              );
            }
            return next;
          });
        }
      }
    };

    wsRef.current = ws;

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [session]);

  const handleReceivedMessage = (
    msgId: string,
    senderId: string,
    senderName: string,
    text: string
  ) => {
    const newMsg: Message = {
      id: msgId,
      chatId: senderId,
      senderId: senderId,
      content: text,
      status: 'delivered',
      isOutgoing: false,
      createdAt: new Date().toISOString(),
    };

    setChatMessages((prev) => ({
      ...prev,
      [senderId]: [...(prev[senderId] || []), newMsg],
    }));

    setChats((prev) => {
      const existing = prev.find((c) => c.id === senderId);
      if (existing) {
        return prev.map((c) =>
          c.id === senderId
            ? {
                ...c,
                lastMessage: newMsg,
                unreadCount: activeChatId === senderId ? 0 : c.unreadCount + 1,
                updatedAt: newMsg.createdAt,
              }
            : c
        );
      } else {
        const newChat: Chat = {
          id: senderId,
          title: senderName,
          peerUser: {
            id: senderId,
            username: senderName,
            isOnline: true,
          },
          lastMessage: newMsg,
          unreadCount: 1,
          updatedAt: newMsg.createdAt,
        };
        return [newChat, ...prev];
      }
    });
  };

  // Login handler
  const handleLogin = async (username: string, password: string) => {
    setIsLoading(true);
    setAuthError(null);
    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Ошибка входа');
      }

      const keyPair = generateKeyPair();

      await fetch('/api/v1/auth/key', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${data.token}`,
        },
        body: JSON.stringify({ public_key: keyPair.publicKeyHex }),
      });

      const newSession: AuthSession = {
        token: data.token,
        userId: data.user_id,
        username: data.username,
        keyPair,
      };

      setSession(newSession);
      sessionStorage.setItem(
        storageKeyPrefix,
        JSON.stringify({
          token: data.token,
          userId: data.user_id,
          username: data.username,
          keyPair: {
            publicKeyHex: keyPair.publicKeyHex,
            privateKeyHex: keyPair.privateKeyHex,
          },
        })
      );

      refreshDirectory();
    } catch (err: any) {
      setAuthError(err.message || 'Ошибка сети');
    } finally {
      setIsLoading(false);
    }
  };

  // Register handler
  const handleRegister = async (username: string, email: string, password: string) => {
    setIsLoading(true);
    setAuthError(null);
    try {
      const keyPair = generateKeyPair();

      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username,
          email,
          password,
          public_key: keyPair.publicKeyHex,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Ошибка регистрации');
      }

      const newSession: AuthSession = {
        token: data.token,
        userId: data.user_id,
        username: data.username,
        keyPair,
      };

      setSession(newSession);
      sessionStorage.setItem(
        storageKeyPrefix,
        JSON.stringify({
          token: data.token,
          userId: data.user_id,
          username: data.username,
          keyPair: {
            publicKeyHex: keyPair.publicKeyHex,
            privateKeyHex: keyPair.privateKeyHex,
          },
        })
      );

      refreshDirectory();
    } catch (err: any) {
      setAuthError(err.message || 'Ошибка сети');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    sessionStorage.removeItem(storageKeyPrefix);
    setSession(null);
    setChats([]);
    setActiveChatId(null);
    setChatMessages({});
    sharedKeysRef.current.clear();
    contactPublicKeysRef.current.clear();
  };

  const startChatWithUser = (user: { user_id: string; username: string; public_key: string }) => {
    if (!session) return;
    if (user.user_id === session.userId) return;

    try {
      const peerPubKeyBytes = hexToBytes(user.public_key);
      const sharedKey = computeSharedKey(session.keyPair.privateKey, peerPubKeyBytes);

      sharedKeysRef.current.set(user.user_id, sharedKey);
      contactPublicKeysRef.current.set(user.user_id, user.public_key);

      const existingChat = chats.find((c) => c.id === user.user_id);
      if (!existingChat) {
        const newChat: Chat = {
          id: user.user_id,
          title: user.username,
          peerUser: {
            id: user.user_id,
            username: user.username,
            isOnline: true,
            publicKey: user.public_key,
          },
          unreadCount: 0,
          updatedAt: new Date().toISOString(),
        };
        setChats((prev) => [newChat, ...prev]);
      }
      setActiveChatId(user.user_id);
    } catch (err) {
      console.error('Failed to start chat:', err);
    }
  };

  const handleSearchUser = async (username: string): Promise<{ found: boolean; message?: string }> => {
    if (!session) return { found: false, message: 'Сессия не найдена' };

    try {
      const res = await fetch(`/api/v1/users/search?username=${encodeURIComponent(username)}`, {
        headers: {
          Authorization: `Bearer ${session.token}`,
        },
      });

      if (!res.ok) {
        const data = await res.json();
        return { found: false, message: data.error || 'Пользователь не найден' };
      }

      const user = await res.json();
      startChatWithUser(user);
      return { found: true };
    } catch {
      return { found: false, message: 'Ошибка связи с сервером' };
    }
  };

  const handleSendMessage = (text: string) => {
    if (!session || !activeChatId || !wsRef.current) return;

    let sharedKey = sharedKeysRef.current.get(activeChatId);
    if (!sharedKey) {
      const pubHex = contactPublicKeysRef.current.get(activeChatId);
      if (pubHex) {
        sharedKey = computeSharedKey(session.keyPair.privateKey, hexToBytes(pubHex));
        sharedKeysRef.current.set(activeChatId, sharedKey);
      }
    }

    if (!sharedKey) {
      console.error('No shared key for active chat:', activeChatId);
      return;
    }

    try {
      const msgId = generateRandomId();
      const msgIdHex = bytesToHex(msgId);
      const recipientId = hexToBytes(activeChatId);

      const { nonce, ciphertext } = encryptMessage(text, sharedKey);
      const packetBytes = encodeMessagePacket(msgId, recipientId, nonce, ciphertext);

      wsRef.current.send(packetBytes.buffer);

      const outgoingMsg: Message = {
        id: msgIdHex,
        chatId: activeChatId,
        senderId: session.userId,
        content: text,
        status: 'sending',
        isOutgoing: true,
        createdAt: new Date().toISOString(),
      };

      setChatMessages((prev) => ({
        ...prev,
        [activeChatId]: [...(prev[activeChatId] || []), outgoingMsg],
      }));

      setChats((prev) =>
        prev.map((c) =>
          c.id === activeChatId
            ? { ...c, lastMessage: outgoingMsg, updatedAt: outgoingMsg.createdAt }
            : c
        )
      );
    } catch (err) {
      console.error('Failed to send message:', err);
    }
  };

  useEffect(() => {
    refreshDirectory();
  }, [refreshDirectory]);

  const activeChat = chats.find((c) => c.id === activeChatId) || null;
  const activeMessages = activeChatId ? chatMessages[activeChatId] || [] : [];

  if (!session) {
    return (
      <AuthView
        onLogin={handleLogin}
        onRegister={handleRegister}
        isLoading={isLoading}
        error={authError}
        clearError={() => setAuthError(null)}
      />
    );
  }

  return (
    <ChatView
      currentUser={{
        id: session.userId,
        username: session.username,
        token: session.token,
        publicKeyHex: session.keyPair.publicKeyHex,
      }}
      chats={chats}
      activeChat={activeChat}
      messages={activeMessages}
      onSelectChat={(chat) => {
        setActiveChatId(chat.id);
        setChats((prev) =>
          prev.map((c) => (c.id === chat.id ? { ...c, unreadCount: 0 } : c))
        );
      }}
      onSendMessage={handleSendMessage}
      onSearchUser={handleSearchUser}
      onLogout={handleLogout}
      isConnected={isConnected}
      availableUsers={availableUsers}
      onStartChatWithUser={startChatWithUser}
      onRefreshDirectory={refreshDirectory}
      serverUrl={serverUrl}
      onSaveServerUrl={onSaveServerUrl}
      isDualView={isDualView}
      onToggleDualView={onToggleDualView}
    />
  );
};

export const App: React.FC = () => {
  const [isDualView, setIsDualView] = useState(false);
  const [serverUrl, setServerUrl] = useState(() => {
    return localStorage.getItem('undermine_server_url') || 'http://localhost:3000';
  });

  const handleSaveServerUrl = (url: string) => {
    setServerUrl(url);
    localStorage.setItem('undermine_server_url', url);
  };

  return (
    <div className="h-screen w-screen bg-[#171717] flex flex-col overflow-hidden">
      {/* Top Helper Bar if Dual View is active */}
      {isDualView && (
        <div className="h-9 bg-[#1E1E1E] border-b border-[#2D2D2D] px-4 flex items-center justify-between text-[12px] text-[#A0A0A0] shrink-0">
          <div className="flex items-center gap-2">
            <Columns className="w-3.5 h-3.5 text-[#C76029]" />
            <span className="font-semibold text-white">Режим двух собеседников:</span>
            <span>Тестирование E2EE в реальном времени между двумя окнами</span>
          </div>
          <button
            onClick={() => setIsDualView(false)}
            className="flex items-center gap-1 text-[11px] hover:text-white px-2 py-0.5 rounded bg-[#2A2A2A] hover:bg-[#333333] transition-colors cursor-pointer"
          >
            <X className="w-3.5 h-3.5" />
            <span>Закрыть сплит</span>
          </button>
        </div>
      )}

      {/* Main View Area */}
      <div className="flex-1 flex overflow-hidden">
        {isDualView ? (
          <>
            {/* Window 1 */}
            <div className="flex-1 border-r-2 border-[#2D2D2D] overflow-hidden flex flex-col">
              <div className="bg-[#1C1C1C] px-3 py-1 text-[11px] font-semibold text-[#C76029] border-b border-[#2D2D2D] flex items-center justify-between">
                <span>Клиент 1 (Alice)</span>
                <span className="text-[10px] text-[#707070]">Curve25519 E2EE</span>
              </div>
              <div className="flex-1 overflow-hidden">
                <MessengerClient
                  initialUsername="alice"
                  initialPassword="password123"
                  storageKeyPrefix="undermine_dual_1"
                  isDualView={true}
                  onToggleDualView={() => setIsDualView(!isDualView)}
                  serverUrl={serverUrl}
                  onSaveServerUrl={handleSaveServerUrl}
                />
              </div>
            </div>

            {/* Window 2 */}
            <div className="flex-1 overflow-hidden flex flex-col">
              <div className="bg-[#1C1C1C] px-3 py-1 text-[11px] font-semibold text-[#3E7EB5] border-b border-[#2D2D2D] flex items-center justify-between">
                <span>Клиент 2 (Bob)</span>
                <span className="text-[10px] text-[#707070]">Curve25519 E2EE</span>
              </div>
              <div className="flex-1 overflow-hidden">
                <MessengerClient
                  initialUsername="bob"
                  initialPassword="password123"
                  storageKeyPrefix="undermine_dual_2"
                  isDualView={true}
                  onToggleDualView={() => setIsDualView(!isDualView)}
                  serverUrl={serverUrl}
                  onSaveServerUrl={handleSaveServerUrl}
                />
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 overflow-hidden">
            <MessengerClient
              storageKeyPrefix="undermine_session_main"
              isDualView={false}
              onToggleDualView={() => setIsDualView(true)}
              serverUrl={serverUrl}
              onSaveServerUrl={handleSaveServerUrl}
            />
          </div>
        )}
      </div>
    </div>
  );
};
