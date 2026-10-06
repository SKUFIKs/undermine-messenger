import http from 'http';
import express, { Request, Response } from 'express';
import cors from 'cors';
import { WebSocketServer, WebSocket } from 'ws';
import crypto from 'crypto';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PORT = parseInt(process.env.PORT || '3000', 10);
const HOST = '0.0.0.0';

// In-Memory Database for Zero-Knowledge Relay backend
interface UserRecord {
  id: string; // 32 hex chars (16 bytes)
  username: string;
  email: string;
  passwordHash: string;
  publicKey: string; // 64 hex chars (32 bytes)
  createdAt: Date;
}

interface SessionRecord {
  tokenHash: string; // SHA-256 of raw token
  rawToken: string;
  userId: string;
  createdAt: Date;
}

const users = new Map<string, UserRecord>(); // key: username.toLowerCase()
const usersById = new Map<string, UserRecord>(); // key: userId
const sessions = new Map<string, SessionRecord>(); // key: tokenHash
const pendingMessages = new Map<string, Buffer[]>(); // key: recipientId -> Buffer[]

// Helper functions
function hashToken(token: string): string {
  return crypto.createHash('sha256').update(token).digest('hex');
}

function hashPassword(password: string): string {
  return crypto.createHash('sha256').update(password + '_undermine_salt').digest('hex');
}

function generateId(): string {
  return crypto.randomBytes(16).toString('hex');
}

function generateToken(): string {
  return crypto.randomBytes(32).toString('hex');
}

// Seed demo users so the user can immediately test without needing two machines
function seedDemoUsers() {
  if (users.size > 0) return;

  const demoUsers = [
    {
      username: 'alice',
      email: 'alice@undermine.network',
      password: 'password123',
      // Pre-set public key for demo
      publicKey: '3b6b27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29',
    },
    {
      username: 'bob',
      email: 'bob@undermine.network',
      password: 'password123',
      publicKey: '2fe57da347cd62431528daac5fbb290730fff684afc4cfc2ed90995f58cb3b74',
    },
    {
      username: 'charlie',
      email: 'charlie@undermine.network',
      password: 'password123',
      publicKey: '68925700874e47cf782803b7ccf6186847bc2a6886e3981ac5550a6e3557e05e',
    },
  ];

  for (const u of demoUsers) {
    const userId = generateId();
    const userRec: UserRecord = {
      id: userId,
      username: u.username,
      email: u.email,
      passwordHash: hashPassword(u.password),
      publicKey: u.publicKey,
      createdAt: new Date(),
    };
    users.set(u.username.toLowerCase(), userRec);
    usersById.set(userId, userRec);
  }
}

seedDemoUsers();

const app = express();
app.use(cors());
app.use(express.json());

// REST API endpoints matching undermine-server
app.post('/api/v1/auth/register', (req: Request, res: Response) => {
  const { username, email, password, public_key } = req.body;

  if (!username || !email || !password || !public_key) {
    return res.status(400).json({ error: 'Заполните обязательные поля' });
  }

  const normalizedUsername = String(username).trim();
  if (normalizedUsername.length < 3) {
    return res.status(400).json({ error: 'Имя пользователя должно содержать не менее 3 символов' });
  }

  if (users.has(normalizedUsername.toLowerCase())) {
    return res.status(400).json({ error: 'Пользователь с таким именем уже существует' });
  }

  const cleanPubKey = String(public_key).trim();
  if (cleanPubKey.length !== 64 || !/^[0-9a-fA-F]+$/.test(cleanPubKey)) {
    return res.status(400).json({ error: 'Неверный формат публичного ключа (ожидается 64 hex-символа)' });
  }

  const userId = generateId();
  const token = generateToken();
  const tHash = hashToken(token);

  const userRec: UserRecord = {
    id: userId,
    username: normalizedUsername,
    email: String(email).trim(),
    passwordHash: hashPassword(String(password)),
    publicKey: cleanPubKey.toLowerCase(),
    createdAt: new Date(),
  };

  users.set(normalizedUsername.toLowerCase(), userRec);
  usersById.set(userId, userRec);

  sessions.set(tHash, {
    tokenHash: tHash,
    rawToken: token,
    userId: userId,
    createdAt: new Date(),
  });

  return res.status(201).json({
    token: token,
    user_id: userId,
    username: userRec.username,
  });
});

app.post('/api/v1/auth/login', (req: Request, res: Response) => {
  const { username, password } = req.body;

  if (!username || !password) {
    return res.status(400).json({ error: 'Заполните имя пользователя и пароль' });
  }

  const normalized = String(username).trim().toLowerCase();
  const user = users.get(normalized);

  if (!user || user.passwordHash !== hashPassword(String(password))) {
    return res.status(401).json({ error: 'Неверное имя пользователя или пароль' });
  }

  const token = generateToken();
  const tHash = hashToken(token);

  sessions.set(tHash, {
    tokenHash: tHash,
    rawToken: token,
    userId: user.id,
    createdAt: new Date(),
  });

  return res.status(200).json({
    token: token,
    user_id: user.id,
    username: user.username,
    public_key: user.publicKey,
  });
});

app.get('/api/v1/users/search', (req: Request, res: Response) => {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Отсутствует заголовок Authorization' });
  }

  const rawToken = authHeader.substring(7).trim();
  const tHash = hashToken(rawToken);
  const session = sessions.get(tHash);

  if (!session) {
    return res.status(401).json({ error: 'Недействительный токен авторизации' });
  }

  const query = String(req.query.username || '').trim().toLowerCase();
  if (!query) {
    return res.status(400).json({ error: 'Параметр username обязателен' });
  }

  const target = users.get(query);
  if (!target) {
    return res.status(404).json({ error: 'Пользователь не найден' });
  }

  return res.status(200).json({
    user_id: target.id,
    username: target.username,
    public_key: target.publicKey,
  });
});

app.post('/api/v1/auth/key', (req: Request, res: Response) => {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Отсутствует заголовок Authorization' });
  }

  const rawToken = authHeader.substring(7).trim();
  const session = sessions.get(hashToken(rawToken));
  if (!session) {
    return res.status(401).json({ error: 'Недействительный токен авторизации' });
  }

  const { public_key } = req.body;
  const cleanPubKey = String(public_key || '').trim().toLowerCase();
  if (cleanPubKey.length !== 64) {
    return res.status(400).json({ error: 'Неверный формат публичного ключа' });
  }

  const user = usersById.get(session.userId);
  if (user) {
    user.publicKey = cleanPubKey;
  }

  return res.status(200).json({ status: 'ok' });
});

// Directory listing of available contacts for quick start
app.get('/api/v1/users/directory', (req: Request, res: Response) => {
  const list = Array.from(usersById.values()).map(u => ({
    user_id: u.id,
    username: u.username,
    public_key: u.publicKey,
  }));
  return res.json(list);
});

// Create HTTP server
const server = http.createServer(app);

// WebSocket Hub
const wsServer = new WebSocketServer({ server, path: '/ws' });

interface WSClient {
  userId: string;
  ws: WebSocket;
}

const activeSockets = new Map<string, WSClient>(); // userId -> WSClient

wsServer.on('connection', (ws: WebSocket, req) => {
  const urlObj = new URL(req.url || '', `http://${req.headers.host || 'localhost'}`);
  const token = urlObj.searchParams.get('token');
  const userIdParam = urlObj.searchParams.get('user_id');

  let authenticatedUserId: string | null = null;

  if (token) {
    const session = sessions.get(hashToken(token));
    if (session) {
      authenticatedUserId = session.userId;
    }
  } else if (userIdParam && usersById.has(userIdParam)) {
    authenticatedUserId = userIdParam;
  }

  if (!authenticatedUserId) {
    ws.close(1008, 'Unauthorized');
    return;
  }

  const userId = authenticatedUserId;
  const client: WSClient = { userId, ws };
  activeSockets.set(userId, client);

  console.log(`[WS] User connected: ${userId} (${usersById.get(userId)?.username})`);

  // Deliver offline pending messages if any
  const pending = pendingMessages.get(userId);
  if (pending && pending.length > 0) {
    console.log(`[WS] Delivering ${pending.length} pending messages to ${userId}`);
    for (const msgData of pending) {
      try {
        ws.send(msgData);
      } catch (err) {
        console.error('[WS] Error sending pending message:', err);
      }
    }
    pendingMessages.delete(userId);
  }

  // Handle incoming binary packets
  ws.on('message', (data: Buffer | ArrayBuffer | Buffer[]) => {
    let buf: Buffer;
    if (Buffer.isBuffer(data)) {
      buf = data;
    } else if (data instanceof ArrayBuffer) {
      buf = Buffer.from(data);
    } else if (Array.isArray(data)) {
      buf = Buffer.concat(data);
    } else {
      return;
    }

    if (buf.length < 1) return;

    const packetType = buf[0];

    // PacketTypeMessage = 0x01
    // min length = 45 bytes (1 type + 16 msgId + 16 recipientId + 12 nonce + payload)
    if (packetType === 0x01 && buf.length >= 45) {
      const msgId = buf.subarray(1, 17);
      const recipientIdBytes = buf.subarray(17, 33);
      const recipientId = recipientIdBytes.toString('hex');

      console.log(`[WS] Routing message: msgId=${msgId.toString('hex')} from ${userId} to ${recipientId}`);

      const recipient = activeSockets.get(recipientId);
      let ackStatus: number;

      if (recipient && recipient.ws.readyState === WebSocket.OPEN) {
        try {
          recipient.ws.send(buf);
          ackStatus = 0x02; // Delivered
          console.log(`[WS] Message delivered online to ${recipientId}`);
        } catch {
          ackStatus = 0x01; // Pending
          if (!pendingMessages.has(recipientId)) pendingMessages.set(recipientId, []);
          pendingMessages.get(recipientId)!.push(buf);
        }
      } else {
        ackStatus = 0x01; // Stored in offline queue
        console.log(`[WS] Recipient ${recipientId} offline, queued message`);
        if (!pendingMessages.has(recipientId)) pendingMessages.set(recipientId, []);
        pendingMessages.get(recipientId)!.push(buf);
      }

      // Send ACK back to sender (18 bytes: 1 type + 16 msgId + 1 status)
      const ackPacket = Buffer.alloc(18);
      ackPacket[0] = 0x02; // PacketTypeMessageAck
      msgId.copy(ackPacket, 1, 0, 16);
      ackPacket[17] = ackStatus;

      try {
        ws.send(ackPacket);
      } catch (err) {
        console.error('[WS] Error sending ACK to sender:', err);
      }
    } else if (packetType === 0x02 && buf.length >= 18) {
      // PacketTypeMessageAck forwarded by client
      const msgId = buf.subarray(1, 17);
      console.log(`[WS] Client ACK received for msgId=${msgId.toString('hex')}`);
    } else if (packetType === 0x03) {
      // Ping -> send Pong (0x04)
      const pong = Buffer.from([0x04]);
      ws.send(pong);
    }
  });

  ws.on('close', () => {
    console.log(`[WS] User disconnected: ${userId}`);
    if (activeSockets.get(userId)?.ws === ws) {
      activeSockets.delete(userId);
    }
  });

  ws.on('error', (err) => {
    console.error(`[WS] Error for user ${userId}:`, err);
  });
});

// Mount Vite or static file serving
async function startServer() {
  const isProd = process.env.NODE_ENV === 'production';

  if (!isProd) {
    const rootDir = path.resolve(__dirname).replace(/\\/g, '/');
    const { createServer: createViteServer } = await import('vite');
    const vite = await createViteServer({
      root: rootDir,
      configFile: path.resolve(rootDir, 'vite.config.ts').replace(/\\/g, '/'),
      server: {
        middlewareMode: true,
        host: '0.0.0.0',
        port: PORT,
      },
      appType: 'spa',
    });

    app.use(vite.middlewares);
  } else {
    const distPath = path.resolve(__dirname, 'dist');
    app.use(express.static(distPath));
    app.get('*', (_req: Request, res: Response) => {
      res.sendFile(path.resolve(distPath, 'index.html'));
    });
  }

  server.listen(PORT, HOST, () => {
    console.log(`==================================================`);
    console.log(`UnderMine Server running on http://${HOST}:${PORT}`);
    console.log(`WebSocket endpoint running on ws://${HOST}:${PORT}/ws`);
    console.log(`Zero-Knowledge Relay ready`);
    console.log(`==================================================`);
  });
}

startServer().catch((err) => {
  console.error('Failed to start server:', err);
  process.exit(1);
});
