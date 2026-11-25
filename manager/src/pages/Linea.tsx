import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type AdvancedSettings, type ConnectionEvent, type Line } from '../api';
import { STATE_META, ago, clock, describe, duration, eventsList, formatPhone } from '../format';
import { Avatar, ErrorBox, Icon, Loading, StatePill, useOverview, useToast } from '../components/ui';

type Tab = 'resumen' | 'ajustes' | 'webhook' | 'registro';

const EVENT_LABELS: [string, string][] = [
  ['MESSAGE', 'Mensajes recibidos'],
  ['SEND_MESSAGE', 'Mensajes enviados'],
  ['READ_RECEIPT', 'Acuses de lectura'],
  ['CONNECTION', 'Conexión'],
  ['QRCODE', 'QR y vinculación'],
  ['CALL', 'Llamadas'],
  ['GROUP', 'Grupos'],
  ['LABEL', 'Etiquetas'],
  ['CONTACT', 'Contactos'],
  ['PRESENCE', 'Presencia'],
  ['CHAT_PRESENCE', 'Escribiendo'],
  ['HISTORY_SYNC', 'Historial'],
  ['NEWSLETTER', 'Canales'],
  ['BUTTON_CLICK', 'Botones'],
];

const KIND: Record<string, { label: string; cls: string }> = {
  connected: { label: 'Conectada', cls: 'ok' },
  pair_success: { label: 'Vinculada', cls: 'ok' },
  disconnected: { label: 'Corte', cls: 'retry' },
  logged_out: { label: 'Sesión cerrada', cls: 'down' },
  temporary_ban: { label: 'Limitada', cls: 'down' },
  connect_failure: { label: 'Rechazada', cls: 'down' },
  stream_replaced: { label: 'Reemplazada', cls: 'retry' },
};

export default function Linea() {
  const { id = '' } = useParams();
  const { data, isLoading, error } = useOverview();
  const [tab, setTab] = useState<Tab>('resumen');

  if (isLoading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const line = data?.find((l) => l.id === id);
  if (!line) {
    return (
      <div className="empty">
        <b>No encontramos esta línea</b>
        <Link className="btn btn-dark" to="/lineas">Volver a Líneas</Link>
      </div>
    );
  }

  return (
    <>
      <Link to="/lineas" className="back-link">
        <Icon name="atras" size={18} />
        Líneas
      </Link>
      <Header line={line} />
      <div className="segmented" role="tablist" aria-label="Secciones de la línea">
        {(
          [
            ['resumen', 'Resumen'],
            ['ajustes', 'Ajustes'],
            ['webhook', 'Webhook y eventos'],
            ['registro', 'Registro'],
          ] as [Tab, string][]
        ).map(([t, label]) => (
          <button key={t} type="button" role="tab" aria-selected={tab === t} onClick={() => setTab(t)}>
            {label}
          </button>
        ))}
      </div>
      {tab === 'resumen' && <Resumen line={line} onWebhook={() => setTab('webhook')} />}
      {tab === 'ajustes' && <Ajustes line={line} />}
      {tab === 'webhook' && <Webhook line={line} />}
      {tab === 'registro' && <Registro line={line} />}
    </>
  );
}

function Header({ line }: { line: Line }) {
  const qc = useQueryClient();
  const toast = useToast();
  const navigate = useNavigate();
  const [busy, setBusy] = useState<string | null>(null);

  async function run(kind: string, fn: () => Promise<unknown>, ok: string) {
    setBusy(kind);
    try {
      await fn();
      toast(ok);
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Algo falló');
    } finally {
      setBusy(null);
      qc.invalidateQueries({ queryKey: ['overview'] });
    }
  }

  const pillLabel = line.state === 'reconectada' && line.lastReconnectAt ? `Reconectada a las ${clock(line.lastReconnectAt)}` : STATE_META[line.state].label;

  return (
    <header className="detail-head">
      <Avatar name={line.name} big />
      <div className="titles">
        <div className="row1">
          <h1 style={{ fontSize: 30 }}>{line.name}</h1>
          <StatePill state={line.state} label={pillLabel} />
        </div>
        <span className="muted" style={{ fontSize: 14 }}>
          <span className="mono">{formatPhone(line.phone)}</span>
        </span>
      </div>
      <div className="acts">
        {line.state === 'caida' && line.needsPairing ? (
          <Link className="btn btn-dark" to={`/lineas/${line.id}/vincular`}>Vincular</Link>
        ) : (
          <button type="button" className="btn btn-dark" disabled={!!busy} onClick={() => run('rec', () => api.reconnect(line.id), 'Reconectando…')}>
            {busy === 'rec' ? <span className="spinner" /> : 'Reconectar'}
          </button>
        )}
        {line.jid && (
          <button
            type="button"
            className="btn btn-danger"
            disabled={!!busy}
            onClick={() => {
              if (confirm(`¿Cerrar la sesión de ${line.name}? Habrá que vincularla otra vez con el teléfono.`)) {
                run('out', () => api.logout(line.token), 'Sesión cerrada');
              }
            }}
          >
            {busy === 'out' ? <span className="spinner" /> : 'Cerrar sesión'}
          </button>
        )}
        <button
          type="button"
          className="btn btn-danger"
          disabled={!!busy}
          onClick={async () => {
            if (!confirm(`¿Eliminar ${line.name}? Se borra la línea del servidor y no se puede deshacer.`)) return;
            await run('del', () => api.remove(line.id), `${line.name} eliminada`);
            navigate('/lineas');
          }}
        >
          Eliminar
        </button>
      </div>
    </header>
  );
}

function Resumen({ line, onWebhook }: { line: Line; onWebhook: () => void }) {
  const hist = useQuery({ queryKey: ['history', line.id, 7], queryFn: () => api.history(line.id, 7) });
  const days = buildWeek(hist.data?.events ?? [], line.state === 'conectada' || line.state === 'reconectada');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {!line.webhook && (
        <div className="alert warn">
          <span className="grow">Esta línea no tiene webhook: los mensajes que recibe no llegan a tu panel.</span>
          <button type="button" className="btn btn-sm" style={{ background: '#6E3605', color: '#fff' }} onClick={onWebhook}>
            Configurar
          </button>
        </div>
      )}
      <div className="grid-auto">
        <div className="stat">
          <span className="label">Situación</span>
          <span style={{ fontSize: 15, lineHeight: 1.45 }}>{describe(line)}</span>
        </div>
        <div className="stat">
          <span className="label">Cortes en 24 h</span>
          <span className="value" style={{ fontSize: 28 }}>{line.cuts24h}</span>
          <span className="hint">{line.cuts24h ? `El más largo duró ${duration(line.longestCutSec)}` : 'Sin cortes'}</span>
        </div>
        <div className="stat">
          <span className="label">Conectada desde</span>
          <span className="value" style={{ fontSize: 28 }}>{line.connectedSince ? clock(line.connectedSince) : '—'}</span>
          <span className="hint">{line.connectedSince ? ago(line.connectedSince) : 'Sin registro en 24 h'}</span>
        </div>
      </div>

      <section className="card">
        <div className="section-head">
          <h2>Conexión en los últimos 7 días</h2>
          <span className="desc">Cada bloque es una hora</span>
        </div>
        {hist.isLoading ? (
          <span className="spinner" />
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <div style={{ minWidth: 320, display: 'flex', flexDirection: 'column', gap: 6 }}>
              {days.map((d) => (
                <div className="hour-row" key={d.label}>
                  <span className="n" style={{ width: 64, fontSize: 13, color: 'var(--ink-2)' }}>{d.label}</span>
                  <div className="hours" role="img" aria-label={`${d.label}: ${d.cuts} cortes`}>
                    {d.cells.map((c, i) => (
                      <span key={i} className={`h-${c}`} style={{ height: 18 }} />
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </section>

      <section className="card" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '18px 28px' }}>
        <div className="kv"><span>ID de instancia</span><span className="mono" style={{ fontSize: 13 }}>{line.id}</span></div>
        <TokenKV token={line.token} />
        <div className="kv"><span>Eventos suscritos</span><span>{eventsList(line.events).length || 'Ninguno'}</span></div>
        <div className="kv"><span>Creada</span><span>{new Date(line.createdAt).toLocaleDateString('es', { day: 'numeric', month: 'long', year: 'numeric' })}</span></div>
      </section>
    </div>
  );
}

function TokenKV({ token }: { token: string }) {
  const toast = useToast();
  const [show, setShow] = useState(false);
  return (
    <div className="kv">
      <span>Token</span>
      <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span className="mono" style={{ fontSize: 13, wordBreak: 'break-all' }}>{show ? token : '••••••••-' + token.slice(-4)}</span>
        <button type="button" className="btn btn-soft btn-icon" style={{ width: 36, height: 36 }} aria-label={show ? 'Ocultar token' : 'Mostrar token'} onClick={() => setShow((v) => !v)}>
          <Icon name="ojo" size={16} />
        </button>
        <button
          type="button"
          className="btn btn-soft btn-icon"
          style={{ width: 36, height: 36 }}
          aria-label="Copiar token"
          onClick={() => navigator.clipboard.writeText(token).then(() => toast('Token copiado'), () => toast('No se pudo copiar'))}
        >
          <Icon name="copiar" size={16} />
        </button>
      </span>
    </div>
  );
}

function Ajustes({ line }: { line: Line }) {
  const toast = useToast();
  const q = useQuery({ queryKey: ['settings', line.id], queryFn: () => api.getSettings(line.id) });
  const [s, setS] = useState<AdvancedSettings | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (q.data) setS(q.data);
  }, [q.data]);

  if (q.isLoading || !s) return q.error ? <ErrorBox error={q.error} /> : <Loading />;

  const items: [keyof AdvancedSettings, string, string][] = [
    ['alwaysOnline', 'Siempre en línea', 'Muestra la línea como «en línea» para los contactos mientras esté conectada.'],
    ['rejectCall', 'Rechazar llamadas', 'Cuelga las llamadas entrantes y envía el mensaje de abajo.'],
    ['readMessages', 'Marcar mensajes como leídos', 'Envía el doble check azul en cuanto llega un mensaje.'],
    ['ignoreGroups', 'Ignorar grupos', 'No envía al webhook los mensajes de grupos.'],
    ['ignoreStatus', 'Ignorar estados', 'No envía al webhook las publicaciones de estados.'],
  ];
  const dirty = JSON.stringify(s) !== JSON.stringify(q.data);

  async function save() {
    if (!s) return;
    setSaving(true);
    try {
      await api.putSettings(line.id, s);
      await q.refetch();
      toast('Ajustes guardados');
    } catch (e) {
      toast(e instanceof Error ? e.message : 'No se pudo guardar');
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="card" style={{ gap: 0, paddingTop: 8, paddingBottom: 16 }}>
      {items.map(([key, title, desc]) => (
        <div className="setting" key={key}>
          <div className="txt">
            <b>{title}</b>
            <span>{desc}</span>
          </div>
          <button type="button" role="switch" className="switch" aria-checked={!!s[key]} aria-label={title} onClick={() => setS({ ...s, [key]: !s[key] })}>
            <span />
          </button>
        </div>
      ))}
      {s.rejectCall && (
        <label className="field" style={{ paddingBottom: 16 }}>
          Mensaje al rechazar una llamada
          <textarea className="input on-surface" rows={2} value={s.msgRejectCall} onChange={(e) => setS({ ...s, msgRejectCall: e.target.value })} placeholder="En este momento no podemos atender llamadas…" />
        </label>
      )}
      <div style={{ display: 'flex', gap: 8 }}>
        <button type="button" className="btn btn-dark" disabled={!dirty || saving} onClick={save}>
          {saving ? <span className="spinner" /> : 'Guardar ajustes'}
        </button>
        {dirty && (
          <button type="button" className="btn btn-soft" onClick={() => setS(q.data!)}>
            Descartar
          </button>
        )}
      </div>
    </section>
  );
}

function Webhook({ line }: { line: Line }) {
  const toast = useToast();
  const qc = useQueryClient();
  const info = useQuery({ queryKey: ['info', line.id], queryFn: () => api.info(line.id) });
  const [url, setUrl] = useState(line.webhook);
  const [events, setEvents] = useState<string[]>(() => eventsList(line.events));
  const [saving, setSaving] = useState(false);
  const dirty = url !== line.webhook || events.slice().sort().join() !== eventsList(line.events).sort().join();

  async function save() {
    if (!info.data) return;
    setSaving(true);
    try {
      await api.connect(line.token, {
        webhookUrl: url.trim(),
        subscribe: events.length ? events : ['MESSAGE'],
        rabbitmqEnable: info.data.rabbitmqEnable ?? '',
        websocketEnable: info.data.websocketEnable ?? '',
        natsEnable: info.data.natsEnable ?? '',
      });
      await qc.invalidateQueries({ queryKey: ['overview'] });
      await qc.invalidateQueries({ queryKey: ['info', line.id] });
      toast('Webhook guardado');
    } catch (e) {
      toast(e instanceof Error ? e.message : 'No se pudo guardar');
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="card">
      {!line.webhook && <div className="alert warn">Esta línea no tiene webhook. Los mensajes que recibe no llegan a tu panel hasta que pongas una URL.</div>}
      <label className="field">
        URL del webhook
        <input className="input on-surface mono" type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://tu-panel.com/api/webhook" style={{ fontSize: 14 }} />
      </label>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <span style={{ fontSize: 14, fontWeight: 600 }}>Eventos que se envían</span>
        <div className="chips">
          {EVENT_LABELS.map(([k, label]) => {
            const on = events.includes(k) || events.includes('ALL');
            return (
              <button
                key={k}
                type="button"
                className="evt-toggle"
                aria-pressed={on}
                onClick={() => setEvents((ev) => (ev.includes('ALL') ? EVENT_LABELS.map(([x]) => x).filter((x) => x !== k) : on ? ev.filter((x) => x !== k) : [...ev, k]))}
              >
                {label}
              </button>
            );
          })}
        </div>
      </div>
      <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>
        Guardar aquí reemplaza el webhook y los eventos de esta línea. Si otro sistema (por ejemplo, tu panel) los configura, lo que guardes aquí lo sobrescribe.
      </p>
      <div>
        <button type="button" className="btn btn-dark" disabled={!dirty || saving || !info.data} onClick={save}>
          {saving ? <span className="spinner" /> : 'Guardar webhook'}
        </button>
      </div>
    </section>
  );
}

function Registro({ line }: { line: Line }) {
  const hist = useQuery({ queryKey: ['history', line.id, 30], queryFn: () => api.history(line.id, 30) });
  if (hist.isLoading) return <Loading />;
  if (hist.error) return <ErrorBox error={hist.error} />;
  const evts = [...(hist.data?.events ?? [])].reverse();
  if (!evts.length) {
    return (
      <div className="empty">
        <b>Sin eventos registrados</b>
        <span>El registro empieza a llenarse desde la versión 0.7.3-re.2.</span>
      </div>
    );
  }
  return (
    <section className="card" style={{ paddingTop: 10, paddingBottom: 10, gap: 0 }}>
      {evts.map((e: ConnectionEvent, i) => {
        const k = KIND[e.kind] ?? { label: e.kind, cls: '' };
        return (
          <div className="row" key={i}>
            <span className="mono muted" style={{ width: 130, flexShrink: 0, fontSize: 12.5 }}>{new Date(e.at).toLocaleString('es', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
            <span className={`ev-chip ${k.cls}`}>{k.label}</span>
            {e.reason && <span style={{ flex: '1 1 220px', fontSize: 14, color: '#2A2F35' }}>{e.reason}</span>}
          </div>
        );
      })}
    </section>
  );
}

/** Siete filas de 24 horas (hoy al final) a partir de los eventos. */
function buildWeek(events: ConnectionEvent[], upNow: boolean) {
  const now = new Date();
  const down: [number, number][] = [];
  let from: number | null = null;
  // Estado antes del primer evento: el contrario de ese evento, o el actual.
  const first = events[0];
  const startUp = first ? !(first.kind === 'connected' || first.kind === 'pair_success') : upNow;
  const weekStart = new Date(now);
  weekStart.setHours(0, 0, 0, 0);
  weekStart.setDate(weekStart.getDate() - 6);
  if (!startUp) from = weekStart.getTime();
  for (const e of events) {
    const t = new Date(e.at).getTime();
    if (e.kind === 'connected' || e.kind === 'pair_success') {
      if (from !== null) down.push([from, t]);
      from = null;
    } else if (from === null) from = t;
  }
  if (from !== null) down.push([from, now.getTime()]);

  const out: { label: string; cells: ('ok' | 'corte' | 'caida' | 'sin')[]; cuts: number }[] = [];
  for (let d = 0; d < 7; d++) {
    const day = new Date(weekStart);
    day.setDate(weekStart.getDate() + d);
    const cells: ('ok' | 'corte' | 'caida' | 'sin')[] = [];
    let cuts = 0;
    for (let h = 0; h < 24; h++) {
      const a = day.getTime() + h * 3600e3;
      const b = a + 3600e3;
      if (a > now.getTime()) {
        cells.push('sin');
        continue;
      }
      const end = Math.min(b, now.getTime());
      let covered = 0;
      for (const [x, y] of down) covered += Math.max(0, Math.min(y, end) - Math.max(x, a));
      const span = end - a;
      if (covered <= 0) cells.push('ok');
      else if (covered >= span - 1000) cells.push('caida');
      else {
        cells.push('corte');
        cuts++;
      }
    }
    const label = d === 6 ? 'Hoy' : day.toLocaleDateString('es', { weekday: 'short', day: 'numeric' });
    out.push({ label, cells, cuts });
  }
  return out;
}
