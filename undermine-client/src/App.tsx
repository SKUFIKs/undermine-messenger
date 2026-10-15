import React, { useState, useEffect, useRef, useCallback } from 'react';
import type { Chat, Message, KeyPair, Group, GroupMember } from './types';
import { AuthView } from './components/AuthView';
import { ChatView } from './components/ChatView';
import { ToastContainer } from './components/Toast';
import type { ToastItem, ToastType } from './components/Toast';
import {
  generateKeyPair,
  computeSharedKey,
  encryptMessage,
  decryptMessage,
  encodeMessagePacket,
  decodeMessagePacket,
  decodeAckPacket,
  encodeGroupMessagePacket,
  decodeGroupMessagePacket,
  generateGroupKey,
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

/** Trim trailing slashes from a server URL */
const trimUrl = (url: string) => url.replace(/\/+$/, '');

/**
 * Parse a response body safely: JSON if possible, otherwise wrap plain text
 * as an error message (Go server returns text/plain errors).
 */
const parseBody = async (res: Response): Promise<any> => {
  const text = await res.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    return { error: text };
  }
};

/** localStorage key under which a user's E2EE keypair is persisted */
const keysStorageKey = (username: string) =>
  `undermine_keys_${username.trim().toLowerCase()}`;

const saveKeyPair = (username: string, keyPair: KeyPair) => {
  try {
    localStorage.setItem(
      keysStorageKey(username),
      JSON.stringify({
        privateKeyHex: keyPair.privateKeyHex,
        publicKeyHex: keyPair.publicKeyHex,
      })
    );
  } catch (err) {
    console.warn('Failed to persist keypair:', err);
  }
};

/**
 * Load the persistent E2EE keypair for a user, creating one on first use.
 * The keypair must survive logout/restart: regenerating it on every login
 * would invalidate the shared keys cached by all peers and break decryption.
 */
const loadOrCreateKeyPair = (username: string): KeyPair => {
  try {
    const saved = localStorage.getItem(keysStorageKey(username));
    if (saved) {
      const parsed = JSON.parse(saved);
      const privateKey = hexToBytes(parsed.privateKeyHex);
      const publicKey = hexToBytes(parsed.publicKeyHex);
      return {
        privateKey,
        publicKey,
        privateKeyHex: parsed.privateKeyHex,
        publicKeyHex: parsed.publicKeyHex,
      };
    }
  } catch (err) {
    console.warn('Failed to load stored keypair:', err);
  }
  const keyPair = generateKeyPair();
  saveKeyPair(username, keyPair);
  return keyPair;
};

export const MessengerClient: React.FC<MessengerClientProps> = ({
  initialUsername,
  initialPassword,
  storageKeyPrefix = 'undermine_session_1',
  isDualView = false,
  onToggleDualView,
  serverUrl = '',
  onSaveServerUrl = () => {},
}) => {
  // API base URL: configured server or same-origin (dev proxy)
  const apiBase = trimUrl(serverUrl);
  const apiUrl = (path: string) => `${apiBase}${path}`;
  const wsBaseUrl = () => {
    if (apiBase) {
      return `${apiBase.replace(/^http/, 'ws')}/ws`;
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/ws`;
  };

  const [session, setSession] = useState<AuthSession | null>(() => {
    const saved = localStorage.getItem(storageKeyPrefix);
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
            privateKeyHex: parsed.keyPair.privateKeyHex,
            publicKeyHex: parsed.keyPair.publicKeyHex,
          },
        };
      } catch {
        return null;
      }
    }
    return null;
  });

  // ---- Toast notifications -------------------------------------------------
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const toastIdRef = useRef(0);

  const pushToast = useCallback((type: ToastType, message: string) => {
    const id = ++toastIdRef.current;
    setToasts((prev) => [...prev.slice(-3), { id, type, message }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  }, []);

  const dismissToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // Update public key on server when session is restored from storage.
  // If the stored token is no longer valid (e.g. the server DB was reset),
  // log out automatically instead of silently staying "logged in".
  useEffect(() => {
    if (!session) return;
    fetch(apiUrl('/api/v1/auth/key'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${session.token}`,
      },
      body: JSON.stringify({ public_key: session.keyPair.publicKeyHex }),
    })
      .then((res) => {
        if (res.status === 401) {
          localStorage.removeItem(storageKeyPrefix);
          setSession(null);
          pushToast('error', 'Сессия истекла. Войдите заново.');
        }
      })
      .catch((err) => console.warn('Failed to sync public key:', err));
  }, [session, apiBase, pushToast, storageKeyPrefix]);

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

  // Group chat state
  const [groups, setGroups] = useState<Group[]>([]);
  const [groupKeys, setGroupKeys] = useState<Map<string, Uint8Array>>(new Map());
  // Persistent group keys: groupId -> key hex (survives restart)
  const groupKeysLoadedRef = useRef(false);

  // Cryptographic Shared Keys cache: contactUserId -> sharedKey (Uint8Array)
  const sharedKeysRef = useRef<Map<string, Uint8Array>>(new Map());
  // Contact public keys: contactUserId -> publicKeyHex
  const contactPublicKeysRef = useRef<Map<string, string>>(new Map());

  // WebSocket reference
  const wsRef = useRef<WebSocket | null>(null);

  // Mirrors of state for use inside long-lived WS callbacks (they capture
  // the render in which the effect ran, so direct reads would go stale)
  const chatsRef = useRef<Chat[]>([]);
  const availableUsersRef = useRef<{ user_id: string; username: string; public_key: string }[]>([]);
  const activeChatIdRef = useRef<string | null>(null);
  useEffect(() => {
    chatsRef.current = chats;
  }, [chats]);
  useEffect(() => {
    availableUsersRef.current = availableUsers;
  }, [availableUsers]);
  useEffect(() => {
    activeChatIdRef.current = activeChatId;
  }, [activeChatId]);

  // ---- Persistent chat history (localStorage, per prefix + user) ----------
  const historyStorageKey = (userId: string) => `${storageKeyPrefix}_history_${userId}`;
  // Which user's history is currently loaded; saves are blocked until then so
  // that a login does not overwrite the stored history with a stale empty state
  const historyLoadedForRef = useRef<string | null>(null);
  const skipNextHistorySaveRef = useRef(false);
  // IDs of all known messages (for dedup); populated from persisted history
  const seenMessageIdsRef = useRef<Set<string>>(new Set());

  // Load persisted history when a session appears
  useEffect(() => {
    if (!session) {
      historyLoadedForRef.current = null;
      return;
    }
    if (historyLoadedForRef.current === session.userId) return;
    historyLoadedForRef.current = session.userId;
    skipNextHistorySaveRef.current = true;
    seenMessageIdsRef.current = new Set();
    try {
      const raw = localStorage.getItem(historyStorageKey(session.userId));
      if (raw) {
        const parsed = JSON.parse(raw);
        const loadedChats = Array.isArray(parsed.chats) ? parsed.chats : [];
        const loadedMessages =
          parsed.chatMessages && typeof parsed.chatMessages === 'object'
            ? parsed.chatMessages
            : {};
        setChats(loadedChats);
        setChatMessages(loadedMessages);
        for (const list of Object.values(loadedMessages) as Message[][]) {
          for (const m of list) seenMessageIdsRef.current.add(m.id);
        }
      } else {
        setChats([]);
        setChatMessages({});
      }
    } catch (err) {
      console.warn('Failed to load chat history:', err);
      setChats([]);
      setChatMessages({});
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session, storageKeyPrefix]);

  // Load groups and their keys when a session appears
  useEffect(() => {
    if (!session || groupKeysLoadedRef.current) return;
    groupKeysLoadedRef.current = true;

    (async () => {
      try {
        const res = await fetch(apiUrl('/api/v1/groups'), {
          headers: { Authorization: `Bearer ${session.token}` },
        });
        if (!res.ok) return;
        const list = await res.json();
        if (!Array.isArray(list)) return;

        const loadedGroups: Group[] = [];
        const loadedKeys = new Map<string, Uint8Array>();

        for (const g of list as { id: string; name: string; owner_id: string }[]) {
          // Fetch members
          let members: GroupMember[] = [];
          try {
            const mRes = await fetch(apiUrl(`/api/v1/groups/${g.id}/members`), {
              headers: { Authorization: `Bearer ${session.token}` },
            });
            if (mRes.ok) {
              const mList = await mRes.json();
              if (Array.isArray(mList)) {
                members = mList.map((m: { user_id: string; username: string; public_key: string }) => ({
                  userId: m.user_id,
                  username: m.username,
                  publicKeyHex: m.public_key,
                  isOnline: false,
                  joinedAt: '',
                }));
              }
            }
          } catch {
            // members fetch failed — group still usable
          }

          // Fetch own encrypted group key
          try {
            const kRes = await fetch(apiUrl(`/api/v1/groups/${g.id}/keys`), {
              headers: { Authorization: `Bearer ${session.token}` },
            });
            if (kRes.ok) {
              const kData = await kRes.json();
              const owner = (list as { id: string; owner_id: string }[]).find((x) => x.id === g.id)?.owner_id || '';
              // Decrypt group key with pairwise key to the group owner
              const ownerMember = members.find((m) => m.userId === owner);
              if (ownerMember) {
                const pairwiseKey = computeSharedKey(
                  session.keyPair.privateKey,
                  hexToBytes(ownerMember.publicKeyHex)
                );
                const keyBytes = hexToBytes(kData.encrypted_key);
                // The encrypted key is: nonce(12) + ciphertext
                const nonce = keyBytes.subarray(0, 12);
                const ciphertext = keyBytes.subarray(12);
                try {
                  const groupKey = decryptMessage(nonce, ciphertext, pairwiseKey);
                  loadedKeys.set(g.id, hexToBytes(groupKey));
                } catch {
                  // key decryption failed — group messages won't decrypt
                }
              }
            }
          } catch {
            // key fetch failed
          }

          loadedGroups.push({
            id: g.id,
            name: g.name,
            ownerId: g.owner_id,
            members,
            createdAt: '',
          });
        }

        setGroups(loadedGroups);
        setGroupKeys(loadedKeys);
      } catch (err) {
        console.warn('Failed to load groups:', err);
      }
    })();
  }, [session, apiBase]);

  // Reset group keys when session ends
  useEffect(() => {
    if (!session && groupKeysLoadedRef.current) {
      groupKeysLoadedRef.current = false;
      setGroups([]);
      setGroupKeys(new Map());
    }
  }, [session]);

  // Persist history whenever it changes (skipped right after a load, while
  // logged out, or before the stored history for this user has been read)
  useEffect(() => {
    if (!session || historyLoadedForRef.current !== session.userId) return;
    if (skipNextHistorySaveRef.current) {
      skipNextHistorySaveRef.current = false;
      return;
    }
    try {
      localStorage.setItem(
        historyStorageKey(session.userId),
        JSON.stringify({ chats, chatMessages })
      );
    } catch (err) {
      console.warn('Failed to save chat history:', err);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chats, chatMessages, session, storageKeyPrefix]);

  // Auto-login with initial credentials if provided
  useEffect(() => {
    if (!session && initialUsername && initialPassword) {
      handleLogin(initialUsername, initialPassword);
    }
  }, [initialUsername, initialPassword]);

  // Fetch available directory of users
  const refreshDirectory = useCallback(async () => {
    try {
      const res = await fetch(apiUrl('/api/v1/users/directory'));
      if (res.ok) {
        const list = await res.json();
        if (Array.isArray(list)) {
          setAvailableUsers(list);

          // Resync cached contact keys: if a peer rotated its public key,
          // recompute the shared key so new messages stay decryptable
          const me = session;
          if (me) {
            for (const u of list as { user_id: string; public_key: string }[]) {
              if (u.user_id === me.userId || !u.public_key) continue;
              const cachedPub = contactPublicKeysRef.current.get(u.user_id);
              if (cachedPub === u.public_key && sharedKeysRef.current.has(u.user_id)) {
                continue;
              }
              try {
                const sk = computeSharedKey(me.keyPair.privateKey, hexToBytes(u.public_key));
                sharedKeysRef.current.set(u.user_id, sk);
                contactPublicKeysRef.current.set(u.user_id, u.public_key);
              } catch (err) {
                console.warn(`Failed to derive shared key for ${u.user_id}:`, err);
              }
            }
          }
        }
      }
    } catch (e) {
      console.warn('Failed to fetch directory', e);
    }
  }, [apiBase, session]);

  // Connect to WebSocket relay (with automatic reconnect)
  useEffect(() => {
    if (!session) {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setIsConnected(false);
      return;
    }

    let disposed = false;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let retryDelay = 1000;
    let ws: WebSocket | null = null;
    let hadConnectionRef = { current: false };

    const connect = () => {
      if (disposed) return;

      const wsUrl = `${wsBaseUrl()}?token=${session.token}`;

      ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';

      ws.onopen = () => {
        retryDelay = 1000;
        setIsConnected(true);
        // Notify only on reconnect — the initial connect happens silently
        // right after login where the user already sees the chat UI.
        if (hadConnectionRef.current) {
          pushToast('info', 'Соединение восстановлено');
        }
        hadConnectionRef.current = true;
      };

      ws.onclose = () => {
        setIsConnected(false);
        hadConnectionRef.current = false;
        if (disposed) return;
        // Reconnect with backoff; the server flushes offline messages on connect
        console.warn(`[WS] Connection lost, reconnecting in ${retryDelay}ms`);
        pushToast('warning', 'Соединение потеряно. Переподключение...');
        retryTimer = setTimeout(() => {
          retryTimer = null;
          connect();
        }, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 15000);
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

        // Try cached shared keys first
        for (const [contactId, sharedKey] of sharedKeysRef.current.entries()) {
          try {
            decryptedText = decryptMessage(packet.nonce, packet.payload, sharedKey);
            matchedSenderId = contactId;
            break;
          } catch {
            // mismatch, continue
          }
        }

        // If no cached key worked, try computing from directory
        if (!decryptedText) {
          fetch(apiUrl('/api/v1/users/directory'))
            .then((r) => r.json())
            .then((users: { user_id: string; username: string; public_key: string }[]) => {
              for (const u of users) {
                if (u.user_id === session.userId) continue;
                try {
                  const peerPubKey = hexToBytes(u.public_key);
                  const sk = computeSharedKey(session.keyPair.privateKey, peerPubKey);
                  const text = decryptMessage(packet.nonce, packet.payload, sk);

                  // Cache the shared key for future messages
                  sharedKeysRef.current.set(u.user_id, sk);
                  contactPublicKeysRef.current.set(u.user_id, u.public_key);

                  handleReceivedMessage(msgIdHex, u.user_id, u.username, text);
                  break;
                } catch {
                  // mismatch, continue
                }
              }
            })
            .catch((err) => console.error('Failed to fetch directory:', err));
        } else if (matchedSenderId && decryptedText) {
          const contactTitle =
            chatsRef.current.find((c) => c.id === matchedSenderId)?.title ||
            availableUsersRef.current.find((u) => u.user_id === matchedSenderId)?.username ||
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
      } else if (packetType === 0x05) {
        // GroupMessagePacket
        const packet = decodeGroupMessagePacket(data);
        if (!packet) return;

        const msgIdHex = bytesToHex(packet.msgId);
        const groupId = bytesToHex(packet.groupId);
        const senderId = bytesToHex(packet.senderId);

        // Deduplicate
        if (seenMessageIdsRef.current.has(msgIdHex)) return;
        seenMessageIdsRef.current.add(msgIdHex);

        const groupKey = groupKeys.get(groupId);
        if (!groupKey) {
          console.warn('No group key for group:', groupId);
          return;
        }

        let decryptedText: string | null = null;
        try {
          decryptedText = decryptMessage(packet.nonce, packet.payload, groupKey);
        } catch {
          console.warn('Failed to decrypt group message');
          return;
        }

        if (!decryptedText) return;

        const group = groups.find((g) => g.id === groupId);
        const senderName =
          group?.members.find((m) => m.userId === senderId)?.username ||
          availableUsersRef.current.find((u) => u.user_id === senderId)?.username ||
          'Участник';

        const newMsg: Message = {
          id: msgIdHex,
          chatId: groupId,
          senderId: senderId,
          content: decryptedText,
          status: 'delivered',
          isOutgoing: false,
          createdAt: new Date().toISOString(),
        };

        setChatMessages((prev) => ({
          ...prev,
          [groupId]: [...(prev[groupId] || []), newMsg],
        }));

        setChats((prev) => {
          const existing = prev.find((c) => c.id === groupId);
          if (existing) {
            return prev.map((c) =>
              c.id === groupId
                ? {
                    ...c,
                    lastMessage: newMsg,
                    unreadCount: activeChatIdRef.current === groupId ? 0 : c.unreadCount + 1,
                    updatedAt: newMsg.createdAt,
                  }
                : c
            );
          }
          const newChat: Chat = {
            id: groupId,
            title: group?.name || 'Группа',
            peerUser: {
              id: groupId,
              username: group?.name || 'Группа',
              isOnline: true,
            },
            lastMessage: newMsg,
            unreadCount: 1,
            updatedAt: newMsg.createdAt,
          };
          return [newChat, ...prev];
        });
      }
    };

      wsRef.current = ws;
    };

    connect();

    return () => {
      disposed = true;
      if (retryTimer) {
        clearTimeout(retryTimer);
        retryTimer = null;
      }
      if (ws) {
        ws.close();
      }
      if (wsRef.current === ws) {
        wsRef.current = null;
      }
    };
  }, [session, apiBase]);

  const handleReceivedMessage = (
    msgId: string,
    senderId: string,
    senderName: string,
    text: string
  ) => {
    // Deduplicate: offline queue may re-deliver after reconnect, and history
    // is persisted across restarts. Checked via a synchronous ref because the
    // WS callback's `chatMessages` closure is stale.
    if (seenMessageIdsRef.current.has(msgId)) return;
    seenMessageIdsRef.current.add(msgId);

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
                unreadCount: activeChatIdRef.current === senderId ? 0 : c.unreadCount + 1,
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
      const res = await fetch(apiUrl('/api/v1/auth/login'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      const data = await parseBody(res);
      if (!res.ok) {
        throw new Error(data.error || 'Ошибка входа');
      }

      // Reuse the persistent keypair for this account. Generating a fresh one
      // on every login would invalidate peers' cached shared keys and break
      // decryption of incoming/outgoing messages.
      const keyPair = loadOrCreateKeyPair(data.username);

      await fetch(apiUrl('/api/v1/auth/key'), {
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
      localStorage.setItem(
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

      pushToast('success', `С возвращением, ${newSession.username}!`);
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
      // Reuse/creates a persistent keypair for this username so that keys
      // stay stable across logout/login and restarts
      const keyPair = loadOrCreateKeyPair(username);

      const res = await fetch(apiUrl('/api/v1/auth/register'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username,
          email,
          password,
          public_key: keyPair.publicKeyHex,
        }),
      });

      const data = await parseBody(res);
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
      localStorage.setItem(
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

      pushToast('success', `Аккаунт создан. Добро пожаловать, ${newSession.username}!`);
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
    localStorage.removeItem(storageKeyPrefix);
    setSession(null);
    setChats([]);
    setActiveChatId(null);
    setChatMessages({});
    sharedKeysRef.current.clear();
    contactPublicKeysRef.current.clear();
    pushToast('info', 'Вы вышли из аккаунта');
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

  // ---- Group management ---------------------------------------------------

  const createGroup = async (name: string, memberIds: string[]) => {
    if (!session) return;

    try {
      // 1. Create group on server
      const res = await fetch(apiUrl('/api/v1/groups'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session.token}`,
        },
        body: JSON.stringify({ name, members: memberIds }),
      });

      if (!res.ok) {
        const data = await parseBody(res);
        throw new Error(data.error || 'Ошибка создания группы');
      }

      const groupData = await res.json();
      const groupId = groupData.id;

      // 2. Generate group key and encrypt it for each member
      const groupKey = generateGroupKey();
      const groupKeyHex = bytesToHex(groupKey);

      // Fetch members' public keys
      const membersRes = await fetch(apiUrl(`/api/v1/groups/${groupId}/members`), {
        headers: { Authorization: `Bearer ${session.token}` },
      });
      const membersList = membersRes.ok ? await membersRes.json() : [];

      for (const member of membersList as { user_id: string; username: string; public_key: string }[]) {
        const memberPubKey = hexToBytes(member.public_key);
        const pairwiseKey = computeSharedKey(session.keyPair.privateKey, memberPubKey);
        const { nonce, ciphertext } = encryptMessage(groupKeyHex, pairwiseKey);
        const encryptedKey = new Uint8Array(12 + ciphertext.length);
        encryptedKey.set(nonce, 0);
        encryptedKey.set(ciphertext, 12);

        // Upload encrypted key for this member
        await fetch(apiUrl(`/api/v1/groups/${groupId}/keys/${member.user_id}`), {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${session.token}`,
          },
          body: JSON.stringify({ encrypted_key: bytesToHex(encryptedKey) }),
        });
      }

      // 3. Add group to local state
      const newGroup: Group = {
        id: groupId,
        name,
        ownerId: session.userId,
        members: (membersList as { user_id: string; username: string; public_key: string }[]).map(
          (m) => ({
            userId: m.user_id,
            username: m.username,
            publicKeyHex: m.public_key,
            isOnline: false,
            joinedAt: '',
          })
        ),
        createdAt: new Date().toISOString(),
      };

      setGroups((prev) => [...prev, newGroup]);
      setGroupKeys((prev) => {
        const next = new Map(prev);
        next.set(groupId, groupKey);
        return next;
      });

      // 4. Add group chat to chat list
      const newChat: Chat = {
        id: groupId,
        title: name,
        peerUser: {
          id: groupId,
          username: name,
          isOnline: true,
        },
        unreadCount: 0,
        updatedAt: new Date().toISOString(),
      };
      setChats((prev) => [newChat, ...prev]);
      setActiveChatId(groupId);

      pushToast('success', `Группа "${name}" создана`);
    } catch (err: any) {
      console.error('Failed to create group:', err);
      pushToast('error', err.message || 'Ошибка создания группы');
    }
  };

  const addGroupMember = async (groupId: string, userId: string) => {
    if (!session) return;

    try {
      // 1. Add member on server
      const res = await fetch(apiUrl(`/api/v1/groups/${groupId}/members`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session.token}`,
        },
        body: JSON.stringify({ user_id: userId }),
      });

      if (!res.ok) {
        const data = await parseBody(res);
        throw new Error(data.error || 'Ошибка добавления участника');
      }

      // 2. Get member's public key
      const searchRes = await fetch(
        apiUrl(`/api/v1/users/search?username=${encodeURIComponent(userId)}`),
        { headers: { Authorization: `Bearer ${session.token}` } }
      );
      // userId here is actually a hex ID, not username — fetch from directory instead
      const dirRes = await fetch(apiUrl('/api/v1/users/directory'), {
        headers: { Authorization: `Bearer ${session.token}` },
      });
      const dirList = dirRes.ok ? await dirRes.json() : [];
      const memberInfo = (dirList as { user_id: string; username: string; public_key: string }[]).find(
        (u) => u.user_id === userId
      );

      if (!memberInfo) {
        throw new Error('Пользователь не найден в каталоге');
      }

      // 3. Encrypt group key for the new member
      const groupKey = groupKeys.get(groupId);
      if (!groupKey) {
        throw new Error('Ключ группы не найден');
      }

      const groupKeyHex = bytesToHex(groupKey);
      const memberPubKey = hexToBytes(memberInfo.public_key);
      const pairwiseKey = computeSharedKey(session.keyPair.privateKey, memberPubKey);
      const { nonce, ciphertext } = encryptMessage(groupKeyHex, pairwiseKey);
      const encryptedKey = new Uint8Array(12 + ciphertext.length);
      encryptedKey.set(nonce, 0);
      encryptedKey.set(ciphertext, 12);

      await fetch(apiUrl(`/api/v1/groups/${groupId}/keys/${userId}`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session.token}`,
        },
        body: JSON.stringify({ encrypted_key: bytesToHex(encryptedKey) }),
      });

      // 4. Update local group state
      setGroups((prev) =>
        prev.map((g) =>
          g.id === groupId
            ? {
                ...g,
                members: [
                  ...g.members,
                  {
                    userId,
                    username: memberInfo.username,
                    publicKeyHex: memberInfo.public_key,
                    isOnline: false,
                    joinedAt: new Date().toISOString(),
                  },
                ],
              }
            : g
        )
      );

      pushToast('success', `${memberInfo.username} добавлен в группу`);
    } catch (err: any) {
      console.error('Failed to add group member:', err);
      pushToast('error', err.message || 'Ошибка добавления участника');
    }
  };

  const removeGroupMember = async (groupId: string, userId: string) => {
    if (!session) return;

    try {
      const res = await fetch(apiUrl(`/api/v1/groups/${groupId}/members/${userId}`), {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${session.token}` },
      });

      if (!res.ok) {
        const data = await parseBody(res);
        throw new Error(data.error || 'Ошибка удаления участника');
      }

      // Update local member list
      const updatedGroups = groups.map((g) =>
        g.id === groupId
          ? { ...g, members: g.members.filter((m) => m.userId !== userId) }
          : g
      );
      setGroups(updatedGroups);

      // Key rotation: only the owner rotates the key so the removed member
      // cannot decrypt future messages.
      const group = updatedGroups.find((g) => g.id === groupId);
      if (group && group.ownerId === session.userId) {
        const newGroupKey = generateGroupKey();
        const newGroupKeyHex = bytesToHex(newGroupKey);

        for (const member of group.members) {
          try {
            const memberPubKey = hexToBytes(member.publicKeyHex);
            const pairwiseKey = computeSharedKey(session.keyPair.privateKey, memberPubKey);
            const { nonce, ciphertext } = encryptMessage(newGroupKeyHex, pairwiseKey);
            const encryptedKey = new Uint8Array(12 + ciphertext.length);
            encryptedKey.set(nonce, 0);
            encryptedKey.set(ciphertext, 12);

            await fetch(apiUrl(`/api/v1/groups/${groupId}/keys/${member.userId}`), {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${session.token}`,
              },
              body: JSON.stringify({ encrypted_key: bytesToHex(encryptedKey) }),
            });
          } catch (err) {
            console.warn(`Failed to rotate key for member ${member.userId}:`, err);
          }
        }

        // Update local key
        setGroupKeys((prev) => {
          const next = new Map(prev);
          next.set(groupId, newGroupKey);
          return next;
        });

        pushToast('info', 'Участник удалён. Ключ группы ротирован.');
      } else {
        pushToast('info', 'Участник удалён из группы');
      }
    } catch (err: any) {
      console.error('Failed to remove group member:', err);
      pushToast('error', err.message || 'Ошибка удаления участника');
    }
  };

  const leaveGroup = async (groupId: string) => {
    if (!session) return;

    try {
      const res = await fetch(apiUrl(`/api/v1/groups/${groupId}`), {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${session.token}` },
      });

      if (!res.ok) {
        const data = await parseBody(res);
        throw new Error(data.error || 'Ошибка выхода из группы');
      }

      setGroups((prev) => prev.filter((g) => g.id !== groupId));
      setGroupKeys((prev) => {
        const next = new Map(prev);
        next.delete(groupId);
        return next;
      });
      setChats((prev) => prev.filter((c) => c.id !== groupId));
      setChatMessages((prev) => {
        const next = { ...prev };
        delete next[groupId];
        return next;
      });

      if (activeChatId === groupId) {
        setActiveChatId(null);
      }

      pushToast('info', 'Вы вышли из группы');
    } catch (err: any) {
      console.error('Failed to leave group:', err);
      pushToast('error', err.message || 'Ошибка выхода из группы');
    }
  };

  const handleSearchUser = async (username: string): Promise<{ found: boolean; message?: string }> => {
    if (!session) return { found: false, message: 'Сессия не найдена' };

    try {
      const res = await fetch(apiUrl(`/api/v1/users/search?username=${encodeURIComponent(username)}`), {
        headers: {
          Authorization: `Bearer ${session.token}`,
        },
      });

      if (!res.ok) {
        const data = await parseBody(res);
        return { found: false, message: data.error || 'Пользователь не найден' };
      }

      const user = await res.json();
      startChatWithUser(user);
      return { found: true };
    } catch {
      return { found: false, message: 'Ошибка связи с сервером' };
    }
  };

  const handleSendMessage = async (text: string) => {
    if (!session || !activeChatId || !wsRef.current) return;

    // Prevent sending to self
    if (activeChatId === session.userId) {
      console.error('Cannot send message to yourself');
      return;
    }

    const recipientId = activeChatId;

    let sharedKey = sharedKeysRef.current.get(recipientId);
    if (!sharedKey) {
      const pubHex = contactPublicKeysRef.current.get(recipientId);
      if (pubHex) {
        try {
          sharedKey = computeSharedKey(session.keyPair.privateKey, hexToBytes(pubHex));
          sharedKeysRef.current.set(recipientId, sharedKey);
        } catch (err) {
          console.warn('Failed to derive shared key from cached contact key:', err);
        }
      }
    }

    // Key cache is empty (e.g. history restored after restart): resolve the
    // peer's public key from the directory instead of silently dropping
    // the message.
    if (!sharedKey) {
      try {
        const res = await fetch(apiUrl('/api/v1/users/directory'));
        if (res.ok) {
          const users = await res.json();
          if (Array.isArray(users)) {
            const peer = users.find(
              (u: { user_id: string; public_key: string }) =>
                u.user_id === recipientId && u.public_key
            );
            if (peer) {
              sharedKey = computeSharedKey(session.keyPair.privateKey, hexToBytes(peer.public_key));
              sharedKeysRef.current.set(peer.user_id, sharedKey);
              contactPublicKeysRef.current.set(peer.user_id, peer.public_key);
            }
          }
        }
      } catch (err) {
        console.warn('Failed to resolve peer key from directory:', err);
      }
    }

    if (!sharedKey) {
      console.error('No shared key for active chat:', recipientId);
      return;
    }

    if (wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket is not open, message not sent');
      return;
    }

    try {
      const msgId = generateRandomId();
      const msgIdHex = bytesToHex(msgId);
      const recipientId = hexToBytes(activeChatId);

      const outgoingMsg: Message = {
        id: msgIdHex,
        chatId: activeChatId,
        senderId: session.userId,
        content: text,
        status: 'sending',
        isOutgoing: true,
        createdAt: new Date().toISOString(),
      };

      // Check if this is a group chat
      const isGroupChat = groups.some((g) => g.id === activeChatId);

      if (isGroupChat) {
        // Send as group message using group key
        const groupKey = groupKeys.get(activeChatId);
        if (!groupKey) {
          console.error('No group key for group:', activeChatId);
          pushToast('error', 'Не удалось отправить сообщение: нет ключа группы');
          return;
        }

        const { nonce, ciphertext } = encryptMessage(text, groupKey);
        const senderIdBytes = hexToBytes(session.userId);
        const packetBytes = encodeGroupMessagePacket(
          senderIdBytes,
          recipientId,
          msgId,
          nonce,
          ciphertext
        );

        wsRef.current.send(packetBytes.buffer);
      } else {
        // Send as personal message using pairwise shared key
        const { nonce, ciphertext } = encryptMessage(text, sharedKey);
        const packetBytes = encodeMessagePacket(msgId, recipientId, nonce, ciphertext);

        wsRef.current.send(packetBytes.buffer);
      }

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
      pushToast('error', 'Не удалось отправить сообщение');
    }
  };

  useEffect(() => {
    refreshDirectory();
  }, [refreshDirectory]);

  const activeChat = chats.find((c) => c.id === activeChatId) || null;
  const activeMessages = activeChatId ? chatMessages[activeChatId] || [] : [];

  if (!session) {
    return (
      <>
        <AuthView
          onLogin={handleLogin}
          onRegister={handleRegister}
          isLoading={isLoading}
          error={authError}
          clearError={() => setAuthError(null)}
        />
        <ToastContainer toasts={toasts} onDismiss={dismissToast} />
      </>
    );
  }

  return (
    <>
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
        onNotify={pushToast}
        groups={groups}
        onCreateGroup={createGroup}
        onAddGroupMember={addGroupMember}
        onRemoveGroupMember={removeGroupMember}
        onLeaveGroup={leaveGroup}
      />
      <ToastContainer toasts={toasts} onDismiss={dismissToast} />
    </>
  );
};

export const App: React.FC = () => {
  const [isDualView, setIsDualView] = useState(false);
  const [serverUrl, setServerUrl] = useState(() => {
    return localStorage.getItem('undermine_server_url') || import.meta.env.VITE_SERVER_URL || '';
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
                <span>Клиент 1</span>
                <span className="text-[10px] text-[#707070]">Curve25519 E2EE</span>
              </div>
              <div className="flex-1 overflow-hidden">
                <MessengerClient
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
                <span>Клиент 2</span>
                <span className="text-[10px] text-[#707070]">Curve25519 E2EE</span>
              </div>
              <div className="flex-1 overflow-hidden">
                <MessengerClient
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
