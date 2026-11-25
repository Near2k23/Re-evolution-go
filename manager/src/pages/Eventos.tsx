import { useEffect, useMemo, useRef, useState } from 'react';
import { api } from '../api';
import { useOverview } from '../components/ui';

interface Evt {
  seq: number;
  at: Date;
  event: string;
  line: string;
  instanceId: string;
  summary: string;
  raw: string;
}

const TYPES: Record<string, { label: string; cls: string }> = {
  Message: { label: 'Mensaje', cls: 'ok' },
  SendMessage: { label: 'Enviado', cls: '' },
  Receipt: { label: 'Acuse', cls: 'violet' },
  Connected: { label: 'Conectada', cls: 'back' },
  Disconnected: { label: 'Desconexión', cls: 'retry' },
  LoggedOut: { label: 'Sesión cerrada', cls: 'down' },
  QRCode: { label: 'QR', cls: 'clay' },
  CallOffer: { label: 'Llamada', cls: 'down' },
};
const typeOf = (e: string) => TYPES[e] ?? { label: e, cls: '' };
const MAX = 500;

function summarize(event: string, data: Record<string, unknown> | undefined): string {
  if (!data) return '';
  const info = data.Info as Record<string, unknown> | undefined;
  const msg = data.Message as Record<string, unknown> | undefined;
  if ((event === 'Message' || event === 'SendMessage') && info) {
    const text =
      (msg?.conversation as string) ||
      ((msg?.extendedTextMessage as Record<string, unknown>)?.text as string) ||
      ((msg?.imageMessage as Record<string, unknown>)?.caption as string) ||
      (msg?.imageMessage ? '[imagen]' : msg?.audioMessage ? '[audio]' : msg?.documentMessage ? '[documento]' : msg?.videoMessage ? '[video]' : msg?.stickerMessage ? '[sticker]' : '[mensaje]');
    const who = (info.PushName as string) || String(info.Chat ?? '').split('@')[0];
    return `${event === 'SendMessage' ? '→ ' : ''}${who}: ${text}`;
  }
  if (event === 'Receipt') return `${String(data.Type ?? 'entregado')} · ${String(data.Chat ?? '').split('@')[0]}`;
  if (event === 'Disconnected') return 'Se perdió la conexión; reintento automático';
  if (event === 'LoggedOut') return 'Sesión cerrada en el teléfono';
  if (event === 'Connected') return 'Conectada';
  return '';
}

export default function Eventos() {
  const { data: lines } = useOverview();
  const [events, setEvents] = useState<Evt[]>([]);
  const [paused, setPaused] = useState(false);
  const [status, setStatus] = useState<'conectando' | 'vivo' | 'cerrado'>('conectando');
  const [off, setOff] = useState<Record<string, boolean>>({});
  const [lineFilter, setLineFilter] = useState('');
  const [sel, setSel] = useState<number | null>(null);
  const pausedRef = useRef(paused);
  const seq = useRef(0);
  pausedRef.current = paused;

  useEffect(() => {
    let ws: WebSocket | null = null;
    let retry: ReturnType<typeof setTimeout>;
    let closed = false;
    const open = () => {
      setStatus('conectando');
      ws = new WebSocket(api.wsUrl());
      ws.onopen = () => setStatus('vivo');
      ws.onclose = () => {
        setStatus('cerrado');
        if (!closed) retry = setTimeout(open, 3000);
      };
      ws.onmessage = (m) => {
        if (pausedRef.current) return;
        try {
          const outer = JSON.parse(m.data as string) as { queue: string; payload: string };
          const p = JSON.parse(outer.payload) as { event?: string; instanceName?: string; instanceId?: string; data?: Record<string, unknown> };
          const event = p.event ?? outer.queue.split('.').pop() ?? '';
          const e: Evt = {
            seq: ++seq.current,
            at: new Date(),
            event,
            line: p.instanceName ?? '',
            instanceId: p.instanceId ?? outer.queue.split('.')[0],
            summary: summarize(event, p.data),
            raw: JSON.stringify(p, null, 2),
          };
          setEvents((prev) => [e, ...prev].slice(0, MAX));
        } catch {
          /* mensaje no JSON: se ignora */
        }
      };
    };
    open();
    return () => {
      closed = true;
      clearTimeout(retry);
      ws?.close();
    };
  }, []);

  const nameById = useMemo(() => Object.fromEntries((lines ?? []).map((l) => [l.id, l.name])), [lines]);
  const kinds = useMemo(() => Array.from(new Set([...Object.keys(TYPES), ...events.map((e) => e.event)])), [events]);
  const visible = events.filter((e) => !off[e.event] && (!lineFilter || e.instanceId === lineFilter));
  const current = visible.find((e) => e.seq === sel) ?? visible[0];

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <h1>Eventos</h1>
            <span className={`pill ${paused || status !== 'vivo' ? '' : 's-conectada'}`} style={paused || status !== 'vivo' ? { background: '#ECEEEA', color: 'var(--ink-2)' } : undefined}>
              <span className={`dot ${!paused && status === 'vivo' ? 'vivo' : ''}`} style={{ background: paused || status !== 'vivo' ? '#8A9099' : 'var(--ok)' }} />
              {paused ? 'En pausa' : status === 'vivo' ? 'En vivo' : status === 'conectando' ? 'Conectando…' : 'Reconectando…'}
            </span>
          </div>
          <p className="sub">Lo que el servidor envía a webhooks y WebSocket, en el momento en que pasa. Se guardan los últimos {MAX} en esta pestaña.</p>
        </div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <label className="field" style={{ flexDirection: 'row', alignItems: 'center', fontSize: 13, color: 'var(--ink-3)' }}>
            Línea
            <select className="input" style={{ height: 44, width: 'auto', fontSize: 14, fontWeight: 600 }} value={lineFilter} onChange={(e) => setLineFilter(e.target.value)}>
              <option value="">Todas las líneas</option>
              {(lines ?? [])
                .slice()
                .sort((a, b) => a.name.localeCompare(b.name, 'es', { numeric: true }))
                .map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
            </select>
          </label>
          <button type="button" className="btn btn-dark" onClick={() => setPaused((p) => !p)}>
            {paused ? 'Reanudar' : 'Pausar'}
          </button>
          <button type="button" className="btn btn-light" onClick={() => setEvents([])} disabled={!events.length}>
            Vaciar
          </button>
        </div>
      </header>

      <div className="chips scroll" role="group" aria-label="Tipos de evento">
        {kinds.map((k) => (
          <button
            key={k}
            type="button"
            className="chip"
            aria-pressed={!off[k]}
            style={off[k] ? { background: 'transparent', color: 'var(--ink-4)', textDecoration: 'line-through' } : { background: 'var(--surface)', color: 'var(--ink)' }}
            onClick={() => setOff((o) => ({ ...o, [k]: !o[k] }))}
          >
            {typeOf(k).label}
          </button>
        ))}
      </div>

      <div className="split">
        <section className="card main" style={{ padding: 8, gap: 2 }} aria-label="Lista de eventos">
          {visible.length === 0 && <p className="muted" style={{ margin: 0, padding: 32, textAlign: 'center' }}>{events.length ? 'Ningún evento con estos filtros.' : 'Esperando eventos…'}</p>}
          {visible.map((e) => (
            <button key={e.seq} type="button" className="ev-row" aria-pressed={current?.seq === e.seq} onClick={() => setSel(e.seq)}>
              <span className="time">{e.at.toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
              <span className={`ev-chip ${typeOf(e.event).cls}`}>{typeOf(e.event).label}</span>
              <span className="line">{e.line || nameById[e.instanceId] || e.instanceId.slice(0, 8)}</span>
              <span className="sum">{e.summary}</span>
            </button>
          ))}
        </section>
        <aside className="side">
          {current ? (
            <div className="json-panel">
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontSize: 15, fontWeight: 650, color: '#fff', flexGrow: 1 }}>
                  {typeOf(current.event).label} · {current.line || nameById[current.instanceId]}
                </span>
                <button type="button" className="btn btn-sm" style={{ background: '#2A2F35', color: '#fff' }} onClick={() => navigator.clipboard.writeText(current.raw)}>
                  Copiar JSON
                </button>
              </div>
              <span style={{ fontSize: 13, color: '#B9BEC5' }}>{current.at.toLocaleString('es')}</span>
              <pre>{current.raw}</pre>
            </div>
          ) : (
            <div className="card">
              <span className="muted">Elige un evento para ver su contenido.</span>
            </div>
          )}
        </aside>
      </div>
    </>
  );
}
