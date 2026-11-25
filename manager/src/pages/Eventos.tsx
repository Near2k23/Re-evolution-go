import { useEffect, useMemo, useRef, useState } from 'react';
import { api } from '../api';
import { useI18n, type I18n, type Key } from '../i18n';
import { byName } from '../format';
import { useOverview } from '../components/ui';

interface Evt {
  seq: number;
  at: Date;
  event: string;
  line: string;
  instanceId: string;
  data?: Record<string, unknown>;
  raw: string;
}

const CLASS: Record<string, string> = {
  Message: 'ok',
  SendMessage: '',
  Receipt: 'violet',
  Connected: 'back',
  Disconnected: 'retry',
  LoggedOut: 'down',
  QRCode: 'clay',
  CallOffer: 'down',
};
const KNOWN = Object.keys(CLASS);
const MAX = 500;

const label = (i: I18n, e: string) => (KNOWN.includes(e) ? i.t(`ev.${e}` as Key) : e);

function summarize(i: I18n, event: string, data: Record<string, unknown> | undefined): string {
  if (!data) return '';
  const info = data.Info as Record<string, unknown> | undefined;
  const msg = data.Message as Record<string, unknown> | undefined;
  if ((event === 'Message' || event === 'SendMessage') && info) {
    const text =
      (msg?.conversation as string) ||
      ((msg?.extendedTextMessage as Record<string, unknown>)?.text as string) ||
      ((msg?.imageMessage as Record<string, unknown>)?.caption as string) ||
      (msg?.imageMessage ? i.t('sum.image') : msg?.audioMessage ? i.t('sum.audio') : msg?.documentMessage ? i.t('sum.document') : msg?.videoMessage ? i.t('sum.video') : msg?.stickerMessage ? i.t('sum.sticker') : i.t('sum.message'));
    const who = (info.PushName as string) || String(info.Chat ?? '').split('@')[0];
    return `${event === 'SendMessage' ? '→ ' : ''}${who}: ${text}`;
  }
  if (event === 'Receipt') return `${String(data.Type ?? '')} · ${String(data.Chat ?? '').split('@')[0]}`;
  if (event === 'Disconnected') return i.t('sum.lost');
  if (event === 'LoggedOut') return i.t('sum.loggedOut');
  if (event === 'Connected') return i.t('sum.connected');
  return '';
}

export default function Eventos() {
  const i = useI18n();
  const { t } = i;
  const { data: lines } = useOverview();
  const [events, setEvents] = useState<Evt[]>([]);
  const [paused, setPaused] = useState(false);
  const [status, setStatus] = useState<'connecting' | 'live' | 'closed'>('connecting');
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
      setStatus('connecting');
      ws = new WebSocket(api.wsUrl());
      ws.onopen = () => setStatus('live');
      ws.onclose = () => {
        setStatus('closed');
        if (!closed) retry = setTimeout(open, 3000);
      };
      ws.onmessage = (m) => {
        if (pausedRef.current) return;
        try {
          const outer = JSON.parse(m.data as string) as { queue: string; payload: string };
          const p = JSON.parse(outer.payload) as { event?: string; instanceName?: string; instanceId?: string; data?: Record<string, unknown> };
          const event = p.event ?? outer.queue.split('.').pop() ?? '';
          setEvents((prev) =>
            [
              {
                seq: ++seq.current,
                at: new Date(),
                event,
                line: p.instanceName ?? '',
                instanceId: p.instanceId ?? outer.queue.split('.')[0],
                data: p.data,
                raw: JSON.stringify(p, null, 2),
              },
              ...prev,
            ].slice(0, MAX),
          );
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
  const kinds = useMemo(() => Array.from(new Set([...KNOWN, ...events.map((e) => e.event)])), [events]);
  const visible = events.filter((e) => !off[e.event] && (!lineFilter || e.instanceId === lineFilter));
  const current = visible.find((e) => e.seq === sel) ?? visible[0];
  const live = !paused && status === 'live';
  const statusText = paused ? t('events.paused') : status === 'live' ? t('events.live') : status === 'connecting' ? t('events.connecting') : t('events.reconnecting');
  const lineName = (e: Evt) => e.line || nameById[e.instanceId] || e.instanceId.slice(0, 8);

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <h1>{t('events.title')}</h1>
            <span className={`pill ${live ? 's-conectada' : ''}`} style={live ? undefined : { background: '#ECEEEA', color: 'var(--ink-2)' }}>
              <span className={`dot ${live ? 'vivo' : ''}`} style={{ background: live ? 'var(--ok)' : '#8A9099' }} />
              {statusText}
            </span>
          </div>
          <p className="sub">{t('events.sub', { n: i.num(MAX) })}</p>
        </div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <label className="field" style={{ flexDirection: 'row', alignItems: 'center', fontSize: 13, color: 'var(--ink-3)' }}>
            {t('events.line')}
            <select className="input" style={{ height: 44, width: 'auto', fontSize: 14, fontWeight: 600 }} value={lineFilter} onChange={(e) => setLineFilter(e.target.value)}>
              <option value="">{t('events.allLines')}</option>
              {(lines ?? [])
                .slice()
                .sort(byName)
                .map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
            </select>
          </label>
          <button type="button" className="btn btn-dark" onClick={() => setPaused((p) => !p)}>
            {paused ? t('events.resume') : t('events.pause')}
          </button>
          <button type="button" className="btn btn-light" onClick={() => setEvents([])} disabled={!events.length}>
            {t('events.clear')}
          </button>
        </div>
      </header>

      <div className="chips scroll" role="group" aria-label={t('events.types')}>
        {kinds.map((k) => (
          <button
            key={k}
            type="button"
            className="chip"
            aria-pressed={!off[k]}
            style={off[k] ? { background: 'transparent', color: 'var(--ink-4)', textDecoration: 'line-through' } : { background: 'var(--surface)', color: 'var(--ink)' }}
            onClick={() => setOff((o) => ({ ...o, [k]: !o[k] }))}
          >
            {label(i, k)}
          </button>
        ))}
      </div>

      <div className="split">
        <section className="card main" style={{ padding: 8, gap: 2 }} aria-label={t('events.list')}>
          {visible.length === 0 && <p className="muted" style={{ margin: 0, padding: 32, textAlign: 'center' }}>{events.length ? t('events.noneFiltered') : t('events.waiting')}</p>}
          {visible.map((e) => (
            <button key={e.seq} type="button" className="ev-row" aria-pressed={current?.seq === e.seq} onClick={() => setSel(e.seq)}>
              <span className="time">{e.at.toLocaleTimeString(i.tag, { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
              <span className={`ev-chip ${CLASS[e.event] ?? ''}`}>{label(i, e.event)}</span>
              <span className="line">{lineName(e)}</span>
              <span className="sum">{summarize(i, e.event, e.data)}</span>
            </button>
          ))}
        </section>
        <aside className="side">
          {current ? (
            <div className="json-panel">
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontSize: 15, fontWeight: 650, color: '#fff', flexGrow: 1 }}>
                  {label(i, current.event)} · {lineName(current)}
                </span>
                <button type="button" className="btn btn-sm" style={{ background: '#2A2F35', color: '#fff' }} onClick={() => navigator.clipboard.writeText(current.raw)}>
                  {t('events.copyJson')}
                </button>
              </div>
              <span style={{ fontSize: 13, color: '#B9BEC5' }}>{current.at.toLocaleString(i.tag)}</span>
              <pre>{current.raw}</pre>
            </div>
          ) : (
            <div className="card">
              <span className="muted">{t('events.pick')}</span>
            </div>
          )}
        </aside>
      </div>
    </>
  );
}
