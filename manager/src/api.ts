// Cliente de la API de Evolution GO (fork). La clave global se guarda solo en
// este navegador; las llamadas de una línea usan el token de esa línea.

export type LineState = 'conectada' | 'reconectada' | 'reconectando' | 'caida';

export interface Line {
  id: string;
  name: string;
  token: string;
  jid: string;
  phone: string;
  state: LineState;
  needsPairing: boolean;
  webhook: string;
  events: string;
  disconnectReason: string;
  createdAt: string;
  cuts24h: number;
  longestCutSec: number;
  lastCutAt?: string;
  lastReconnectAt?: string;
  connectedSince?: string;
  lastKind?: string;
  lastReason?: string;
  lastEventAt?: string;
  hours: ('ok' | 'corte' | 'caida')[];
  groupId?: string;
}

export interface Group {
  id: string;
  name: string;
  color: string;
  createdAt: string;
}

export interface ConnectionEvent {
  instanceId: string;
  kind: string;
  reason?: string;
  at: string;
}

export interface Cut {
  start: string;
  end?: string;
  kind: string;
  reason?: string;
}

export interface AdvancedSettings {
  alwaysOnline: boolean;
  rejectCall: boolean;
  msgRejectCall: string;
  readMessages: boolean;
  ignoreGroups: boolean;
  ignoreStatus: boolean;
}

export interface InstanceInfo {
  id: string;
  name: string;
  token: string;
  webhook: string;
  rabbitmqEnable: string;
  websocketEnable: string;
  natsEnable: string;
  events: string;
  jid: string;
}

export interface QrInfo {
  qrcode?: string;
  code?: string;
  passkeyStage?: string;
  passkeyOpenUrl?: string;
  passkeyCode?: string;
}

export interface StoredChat {
  jid: string;
  name: string;
  displayName: string;
  isGroup: boolean;
  lastMessageAt?: string;
  lastMessageId?: string;
  lastPreview: string;
  lastType: string;
  lastFromMe: boolean;
  unread: number;
  archived: boolean;
  pinned: boolean;
  mutedUntil?: string;
}

export interface StoredMessage {
  id: string;
  chat: string;
  sender: string;
  senderName?: string;
  fromMe: boolean;
  type: string;
  text: string;
  mime?: string;
  fileName?: string;
  quotedId?: string;
  status: 'received' | 'sent' | 'delivered' | 'read' | 'played';
  edited: boolean;
  deleted: boolean;
  timestamp: string;
}

const KEY = 'enlace.session';

export interface Session {
  apiUrl: string;
  apiKey: string;
}

export function loadSession(): Session | null {
  try {
    const raw = localStorage.getItem(KEY) ?? sessionStorage.getItem(KEY);
    return raw ? (JSON.parse(raw) as Session) : null;
  } catch {
    return null;
  }
}

export function saveSession(s: Session, remember: boolean) {
  try {
    clearSession();
    (remember ? localStorage : sessionStorage).setItem(KEY, JSON.stringify(s));
  } catch {
    /* almacenamiento no disponible: la sesión vive solo en memoria */
  }
}

export function clearSession() {
  try {
    localStorage.removeItem(KEY);
    sessionStorage.removeItem(KEY);
  } catch {
    /* nada */
  }
}

export function defaultApiUrl(): string {
  const env = (import.meta as unknown as { env?: Record<string, string> }).env?.VITE_API_URL;
  return env || window.location.origin;
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

let current: Session | null = loadSession();
export function setCurrentSession(s: Session | null) {
  current = s;
}

async function request<T>(method: string, path: string, opts: { key?: string; body?: unknown; base?: string } = {}): Promise<T> {
  const base = (opts.base ?? current?.apiUrl ?? defaultApiUrl()).replace(/\/+$/, '');
  const key = opts.key ?? current?.apiKey ?? '';
  const res = await fetch(base + path, {
    method,
    headers: { apikey: key, ...(opts.body !== undefined ? { 'Content-Type': 'application/json' } : {}) },
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });
  const text = await res.text();
  let data: unknown = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  if (!res.ok) {
    const msg = (data && typeof data === 'object' && 'error' in data ? String((data as { error: unknown }).error) : '') || `Error ${res.status}`;
    throw new ApiError(res.status, msg);
  }
  return data as T;
}

type Wrapped<T> = { message: string; data: T };

export const api = {
  /** Comprueba servidor y clave global. */
  async verify(apiUrl: string, apiKey: string): Promise<void> {
    await request('GET', '/server/ok', { base: apiUrl, key: apiKey });
    await request('GET', '/instance/overview', { base: apiUrl, key: apiKey });
  },
  overview: () => request<Wrapped<Line[]>>('GET', '/instance/overview').then((r) => r.data ?? []),
  history: (id: string, days = 7) =>
    request<Wrapped<{ events: ConnectionEvent[]; cuts: Cut[] }>>('GET', `/instance/history/${id}?days=${days}`).then((r) => r.data),
  info: (id: string) => request<Wrapped<InstanceInfo>>('GET', `/instance/info/${id}`).then((r) => r.data),
  create: (name: string) =>
    request<Wrapped<InstanceInfo>>('POST', '/instance/create', { body: { name, token: crypto.randomUUID() } }).then((r) => r.data),
  remove: (id: string) => request('DELETE', `/instance/delete/${id}`),
  reconnect: (id: string) => request('POST', `/instance/reconnect/${id}`),

  groups: () => request<Wrapped<Group[]> & { colors: string[] }>('GET', '/line-groups').then((r) => ({ groups: r.data ?? [], colors: r.colors ?? [] })),
  createGroup: (name: string, color: string) => request<Wrapped<Group>>('POST', '/line-groups', { body: { name, color } }).then((r) => r.data),
  updateGroup: (id: string, name: string, color: string) => request<Wrapped<Group>>('PUT', `/line-groups/${id}`, { body: { name, color } }).then((r) => r.data),
  deleteGroup: (id: string) => request('DELETE', `/line-groups/${id}`),
  storeStatus: (id: string) => request<Wrapped<{ enabled: boolean; history: boolean; retentionDays: number }>>('GET', `/store/${id}/status`).then((r) => r.data),
  chats: (id: string, q = '', archived = false) =>
    request<Wrapped<StoredChat[]>>('GET', `/store/${id}/chats?limit=100&archived=${archived}&q=${encodeURIComponent(q)}`).then((r) => r.data ?? []),
  messages: (id: string, chat: string, before?: string) =>
    request<Wrapped<StoredMessage[]>>('GET', `/store/${id}/messages?limit=50&chat=${encodeURIComponent(chat)}${before ? `&before=${encodeURIComponent(before)}` : ''}`).then((r) => r.data ?? []),
  markRead: (id: string, chat: string) => request('POST', `/store/${id}/read`, { body: { chat } }),
  sendText: (token: string, number: string, text: string) => request('POST', '/send/text', { key: token, body: { number, text } }),
  assignGroup: (instanceIds: string[], groupId: string) => request('POST', '/line-groups/assign', { body: { instanceIds, groupId } }),

  // Con el token de la línea.
  connect: (token: string, body: { webhookUrl: string; subscribe: string[]; rabbitmqEnable: string; websocketEnable: string; natsEnable: string; immediate?: boolean }) =>
    request<Wrapped<{ jid: string; webhookUrl: string; eventString: string }>>('POST', '/instance/connect', { key: token, body }),
  qr: (token: string) => request<Wrapped<QrInfo>>('GET', '/instance/qr', { key: token }).then((r) => r.data),
  pair: (token: string, phone: string, subscribe: string[]) =>
    request<Wrapped<{ PairingCode: string }>>('POST', '/instance/pair', { key: token, body: { phone, subscribe } }).then((r) => r.data.PairingCode),
  logout: (token: string) => request('DELETE', '/instance/logout', { key: token }),
  disconnect: (token: string) => request('POST', '/instance/disconnect', { key: token }),
  getSettings: (id: string) => request<AdvancedSettings | Wrapped<AdvancedSettings>>('GET', `/instance/${id}/advanced-settings`).then((r) => ('data' in (r as object) ? (r as Wrapped<AdvancedSettings>).data : (r as AdvancedSettings))),
  putSettings: (id: string, s: AdvancedSettings) => request('PUT', `/instance/${id}/advanced-settings`, { body: s }),

  /** URL del WebSocket de eventos (todas las líneas, con la clave global). */
  wsUrl(): string {
    const base = (current?.apiUrl ?? defaultApiUrl()).replace(/\/+$/, '').replace(/^http/, 'ws');
    return `${base}/ws?token=${encodeURIComponent(current?.apiKey ?? '')}`;
  },
};
