import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type AdvancedSettings, type ConnectionEvent, type Group, type Line } from '../api';
import { useI18n, type Key } from '../i18n';
import { describe, eventsList, formatPhone, stateLabel } from '../format';
import WebhookTab from '../components/WebhookTab';
import ProxyCard from '../components/ProxyCard';
import { Avatar, ErrorBox, Icon, Loading, NotFound, StatePill, useGroups, useOverview, useReconnect, useToast } from '../components/ui';

type Tab = 'resumen' | 'ajustes' | 'webhook' | 'registro';

const KIND_CLASS: Record<string, string> = {
  connected: 'ok',
  pair_success: 'ok',
  disconnected: 'retry',
  logged_out: 'down',
  temporary_ban: 'down',
  connect_failure: 'down',
  stream_replaced: 'retry',
};

export default function Linea() {
  const { id = '' } = useParams();
  const { t } = useI18n();
  const { data, isLoading, error } = useOverview();
  const { data: gd } = useGroups();
  const [tab, setTab] = useState<Tab>('resumen');

  if (isLoading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const line = data?.find((l) => l.id === id);
  if (!line) return <NotFound />;
  const groups = gd?.groups ?? [];
  const group = groups.find((g) => g.id === line.groupId) ?? null;

  const tabs: [Tab, string][] = [
    ['resumen', t('line.tab.summary')],
    ['ajustes', t('line.tab.settings')],
    ['webhook', t('line.tab.webhook')],
    ['registro', t('line.tab.log')],
  ];

  return (
    <>
      <Link to="/lineas" className="back-link">
        <Icon name="atras" size={18} />
        {t('nav.lines')}
      </Link>
      <Header line={line} group={group} />
      <div className="segmented" role="tablist" aria-label={t('line.tabs')}>
        {tabs.map(([k, label]) => (
          <button key={k} type="button" role="tab" aria-selected={tab === k} onClick={() => setTab(k)}>
            {label}
          </button>
        ))}
      </div>
      {tab === 'resumen' && <Resumen line={line} groups={groups} onWebhook={() => setTab('webhook')} />}
      {tab === 'ajustes' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Ajustes line={line} />
          <ProxyCard line={line} />
        </div>
      )}
      {tab === 'webhook' && <WebhookTab line={line} />}
      {tab === 'registro' && <Registro line={line} />}
    </>
  );
}

function Header({ line, group }: { line: Line; group: Group | null }) {
  const i = useI18n();
  const { t } = i;
  const qc = useQueryClient();
  const toast = useToast();
  const navigate = useNavigate();
  const { run: reconnect, busy: rbusy } = useReconnect();
  const [busy, setBusy] = useState<string | null>(null);

  async function run(kind: string, fn: () => Promise<unknown>, ok: string) {
    setBusy(kind);
    try {
      await fn();
      toast(ok);
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setBusy(null);
      qc.invalidateQueries({ queryKey: ['overview'] });
    }
  }

  const pillLabel = line.state === 'reconectada' && line.lastReconnectAt ? t('line.reconnectedAt', { time: i.clock(line.lastReconnectAt) }) : stateLabel(i, line.state);
  const pairing = line.state === 'caida' && line.needsPairing;
  const reconnectLabel = pairing ? t('card.tryReconnect') : line.state === 'reconectando' ? t('card.reconnectNow') : t('line.reconnect');

  return (
    <header className="detail-head">
      <Avatar name={line.name} group={group} line={line} big />
      <div className="titles">
        <div className="row1">
          <h1 style={{ fontSize: 30 }}>{line.name}</h1>
          <StatePill state={line.state} label={pillLabel} />
        </div>
        <span className="muted mono" style={{ fontSize: 14 }}>
          {line.phone ? formatPhone(line.phone) : t('card.noNumber')}
        </span>
      </div>
      <div className="acts">
        {pairing && (
          <Link className="btn btn-dark" to={`/lineas/${line.id}/vincular`}>
            {t('card.link')}
          </Link>
        )}
        {(line.jid || !pairing) && (
          <button
            type="button"
            className={pairing ? 'btn btn-light' : 'btn btn-dark'}
            disabled={!!rbusy[line.id]}
            title={pairing ? t('card.tryReconnectHint') : undefined}
            onClick={() => reconnect([line])}
          >
            {rbusy[line.id] ? <span className="spinner" /> : (
              <>
                <Icon name="recargar" size={18} />
                {reconnectLabel}
              </>
            )}
          </button>
        )}
        {line.jid && (
          <button
            type="button"
            className="btn btn-danger"
            disabled={!!busy}
            onClick={() => {
              if (confirm(t('line.logoutConfirm', { name: line.name }))) run('out', () => api.logout(line.token), t('line.logoutDone'));
            }}
          >
            {busy === 'out' ? <span className="spinner" /> : t('line.logout')}
          </button>
        )}
        <button
          type="button"
          className="btn btn-danger"
          disabled={!!busy}
          onClick={async () => {
            if (!confirm(t('line.deleteConfirm', { name: line.name }))) return;
            await run('del', () => api.remove(line.id), t('line.deleted', { name: line.name }));
            navigate('/lineas');
          }}
        >
          {t('common.delete')}
        </button>
      </div>
    </header>
  );
}

function Resumen({ line, groups, onWebhook }: { line: Line; groups: Group[]; onWebhook: () => void }) {
  const i = useI18n();
  const { t } = i;
  const qc = useQueryClient();
  const toast = useToast();
  const hist = useQuery({ queryKey: ['history', line.id, 7], queryFn: () => api.history(line.id, 7) });
  const days = buildWeek(hist.data?.events ?? [], line.state === 'conectada' || line.state === 'reconectada', i.tag, t('line.today'));

  async function setGroup(groupId: string) {
    try {
      await api.assignGroup([line.id], groupId);
      await qc.invalidateQueries({ queryKey: ['overview'] });
      toast(t('groups.moved'));
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.saveFailed'));
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {!line.webhook && (
        <div className="alert warn">
          <span className="grow">{t('line.noHookAlert')}</span>
          <button type="button" className="btn btn-sm" style={{ background: '#6E3605', color: '#fff' }} onClick={onWebhook}>
            {t('line.configure')}
          </button>
        </div>
      )}
      <div className="grid-auto">
        <div className="stat">
          <span className="label">{t('line.status')}</span>
          <span style={{ fontSize: 15, lineHeight: 1.45 }}>{describe(i, line)}</span>
        </div>
        <div className="stat">
          <span className="label">{t('line.cuts')}</span>
          <span className="value" style={{ fontSize: 28 }}>{i.num(line.cuts24h)}</span>
          <span className="hint">{line.cuts24h ? t('line.longest', { d: i.duration(line.longestCutSec) }) : t('line.noCuts')}</span>
        </div>
        <div className="stat">
          <span className="label">{t('line.connectedSince')}</span>
          <span className="value" style={{ fontSize: 28 }}>{line.connectedSince ? i.clock(line.connectedSince) : '—'}</span>
          <span className="hint">{line.connectedSince ? i.ago(line.connectedSince) : t('line.noRecord')}</span>
        </div>
      </div>

      <section className="card">
        <div className="section-head">
          <h2>{t('line.week')}</h2>
          <span className="desc">{t('line.eachBlock')}</span>
        </div>
        {hist.isLoading ? (
          <span className="spinner" />
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <div style={{ minWidth: 320, display: 'flex', flexDirection: 'column', gap: 6 }}>
              {days.map((d) => (
                <div className="hour-row" key={d.label}>
                  <span className="n" style={{ width: 72, fontSize: 13, color: 'var(--ink-2)' }}>{d.label}</span>
                  <div className="hours" role="img" aria-label={t('line.dayAria', { day: d.label, n: i.num(d.cuts) })}>
                    {d.cells.map((c, k) => (
                      <span key={k} className={`h-${c}`} style={{ height: 18 }} />
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </section>

      <section className="card" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '18px 28px' }}>
        <label className="kv">
          <span>{t('line.group')}</span>
          <select className="input on-surface" style={{ height: 44, fontSize: 14 }} value={line.groupId ?? ''} onChange={(e) => setGroup(e.target.value)}>
            <option value="">{t('lines.noGroup')}</option>
            {groups.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </select>
        </label>
        <div className="kv"><span>{t('line.instanceId')}</span><span className="mono" style={{ fontSize: 13, wordBreak: 'break-all' }}>{line.id}</span></div>
        <TokenKV token={line.token} />
        <div className="kv"><span>{t('line.events')}</span><span>{eventsList(line.events).length ? i.num(eventsList(line.events).length) : t('line.none')}</span></div>
        <div className="kv"><span>{t('line.created')}</span><span>{new Date(line.createdAt).toLocaleDateString(i.tag, { day: 'numeric', month: 'long', year: 'numeric' })}</span></div>
      </section>
    </div>
  );
}

function TokenKV({ token }: { token: string }) {
  const { t } = useI18n();
  const toast = useToast();
  const [show, setShow] = useState(false);
  return (
    <div className="kv">
      <span>{t('line.token')}</span>
      <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span className="mono" style={{ fontSize: 13, wordBreak: 'break-all' }}>{show ? token : '••••••••-' + token.slice(-4)}</span>
        <button type="button" className="btn btn-soft btn-icon" style={{ width: 36, height: 36 }} aria-label={show ? t('line.hideToken') : t('line.showToken')} onClick={() => setShow((v) => !v)}>
          <Icon name="ojo" size={16} />
        </button>
        <button
          type="button"
          className="btn btn-soft btn-icon"
          style={{ width: 36, height: 36 }}
          aria-label={t('line.copyToken')}
          onClick={() => navigator.clipboard.writeText(token).then(() => toast(t('common.copied')), () => toast(t('common.copyFailed')))}
        >
          <Icon name="copiar" size={16} />
        </button>
      </span>
    </div>
  );
}

function Ajustes({ line }: { line: Line }) {
  const { t } = useI18n();
  const toast = useToast();
  const q = useQuery({ queryKey: ['settings', line.id], queryFn: () => api.getSettings(line.id) });
  const [s, setS] = useState<AdvancedSettings | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (q.data) setS(q.data);
  }, [q.data]);

  if (q.isLoading || !s) return q.error ? <ErrorBox error={q.error} /> : <Loading />;

  const items: (keyof AdvancedSettings)[] = ['alwaysOnline', 'rejectCall', 'readMessages', 'readStatus', 'ignoreGroups', 'ignoreStatus'];
  const dirty = JSON.stringify(s) !== JSON.stringify(q.data);

  async function save() {
    if (!s) return;
    setSaving(true);
    try {
      await api.putSettings(line.id, s);
      await q.refetch();
      toast(t('settings.saved'));
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.saveFailed'));
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="card" style={{ gap: 0, paddingTop: 8, paddingBottom: 16 }}>
      {items.map((key) => {
        const title = t(`settings.${key}` as Key);
        return (
          <div className="setting" key={key}>
            <div className="txt">
              <b>{title}</b>
              <span>{t(`settings.${key}.desc` as Key)}</span>
            </div>
            <button type="button" role="switch" className="switch" aria-checked={!!s[key]} aria-label={title} onClick={() => setS({ ...s, [key]: !s[key] })}>
              <span />
            </button>
          </div>
        );
      })}
      {s.rejectCall && (
        <label className="field" style={{ paddingBottom: 16 }}>
          {t('settings.rejectMsg')}
          <textarea className="input on-surface" rows={2} value={s.msgRejectCall} onChange={(e) => setS({ ...s, msgRejectCall: e.target.value })} placeholder={t('settings.rejectMsg.placeholder')} />
        </label>
      )}
      <div style={{ display: 'flex', gap: 8 }}>
        <button type="button" className="btn btn-dark" disabled={!dirty || saving} onClick={save}>
          {saving ? <span className="spinner" /> : t('settings.save')}
        </button>
        {dirty && (
          <button type="button" className="btn btn-soft" onClick={() => setS(q.data!)}>
            {t('common.discard')}
          </button>
        )}
      </div>
    </section>
  );
}

function Registro({ line }: { line: Line }) {
  const i = useI18n();
  const { t } = i;
  const hist = useQuery({ queryKey: ['history', line.id, 30], queryFn: () => api.history(line.id, 30) });
  if (hist.isLoading) return <Loading />;
  if (hist.error) return <ErrorBox error={hist.error} />;
  const evts = [...(hist.data?.events ?? [])].reverse();
  if (!evts.length) {
    return (
      <div className="empty">
        <b>{t('log.emptyTitle')}</b>
        <span>{t('log.emptyHint')}</span>
      </div>
    );
  }
  return (
    <section className="card" style={{ paddingTop: 10, paddingBottom: 10, gap: 0 }}>
      {evts.map((e: ConnectionEvent, k) => {
        const known = e.kind in KIND_CLASS;
        return (
          <div className="row" key={k}>
            <span className="mono muted" style={{ width: 140, flexShrink: 0, fontSize: 12.5 }}>
              {new Date(e.at).toLocaleString(i.tag, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
            </span>
            <span className={`ev-chip ${KIND_CLASS[e.kind] ?? ''}`}>{known ? t(`kind.${e.kind}` as Key) : e.kind}</span>
            {e.reason && <span style={{ flex: '1 1 220px', fontSize: 14, color: '#2A2F35' }}>{e.reason}</span>}
          </div>
        );
      })}
    </section>
  );
}

/** Siete filas de 24 horas (hoy al final) a partir de los eventos. */
function buildWeek(events: ConnectionEvent[], upNow: boolean, tag: string, todayLabel: string) {
  const now = new Date();
  const down: [number, number][] = [];
  let from: number | null = null;
  const first = events[0];
  const startUp = first ? !(first.kind === 'connected' || first.kind === 'pair_success') : upNow;
  const weekStart = new Date(now);
  weekStart.setHours(0, 0, 0, 0);
  weekStart.setDate(weekStart.getDate() - 6);
  if (!startUp) from = weekStart.getTime();
  for (const e of events) {
    const at = new Date(e.at).getTime();
    if (e.kind === 'connected' || e.kind === 'pair_success') {
      if (from !== null) down.push([from, at]);
      from = null;
    } else if (from === null) from = at;
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
      if (a > now.getTime()) {
        cells.push('sin');
        continue;
      }
      const end = Math.min(a + 3600e3, now.getTime());
      let covered = 0;
      for (const [x, y] of down) covered += Math.max(0, Math.min(y, end) - Math.max(x, a));
      if (covered <= 0) cells.push('ok');
      else if (covered >= end - a - 1000) cells.push('caida');
      else {
        cells.push('corte');
        cuts++;
      }
    }
    out.push({ label: d === 6 ? todayLabel : day.toLocaleDateString(tag, { weekday: 'short', day: 'numeric' }), cells, cuts });
  }
  return out;
}
