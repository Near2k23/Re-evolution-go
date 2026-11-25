import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type Line } from '../api';
import { useI18n, type Key } from '../i18n';
import { eventsList } from '../format';
import { ErrorBox, Icon, Loading, useToast } from './ui';

const EVENT_KEYS = ['MESSAGE', 'SEND_MESSAGE', 'READ_RECEIPT', 'CONNECTION', 'QRCODE', 'CALL', 'GROUP', 'LABEL', 'CONTACT', 'PRESENCE', 'CHAT_PRESENCE', 'HISTORY_SYNC', 'NEWSLETTER', 'BUTTON_CLICK'];

type Row = { k: string; v: string };

export default function WebhookTab({ line }: { line: Line }) {
  const i = useI18n();
  const { t } = i;
  const toast = useToast();
  const qc = useQueryClient();
  const info = useQuery({ queryKey: ['info', line.id], queryFn: () => api.info(line.id) });
  const hook = useQuery({ queryKey: ['webhook', line.id], queryFn: () => api.webhook(line.id) });
  const deliveries = useQuery({ queryKey: ['deliveries', line.id], queryFn: () => api.deliveries(line.id), refetchInterval: 10000 });

  const [url, setUrl] = useState(line.webhook);
  const [events, setEvents] = useState<string[]>(() => eventsList(line.events));
  const [rows, setRows] = useState<Row[]>([]);
  const [byEvents, setByEvents] = useState(false);
  const [secretAction, setSecretAction] = useState<'keep' | 'new' | 'remove'>('keep');
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    if (!hook.data) return;
    setRows(Object.entries(hook.data.headers ?? {}).map(([k, v]) => ({ k, v })));
    setByEvents(hook.data.byEvents);
  }, [hook.data]);

  if (hook.isLoading || info.isLoading) return <Loading />;
  if (hook.error) return <ErrorBox error={hook.error} />;
  const h = hook.data!;

  const headersObj = Object.fromEntries(rows.filter((r) => r.k.trim()).map((r) => [r.k.trim(), r.v]));
  const basicDirty = url !== line.webhook || events.slice().sort().join() !== eventsList(line.events).sort().join();
  const advDirty = JSON.stringify(headersObj) !== JSON.stringify(h.headers ?? {}) || byEvents !== h.byEvents || secretAction !== 'keep';

  async function save() {
    setSaving(true);
    try {
      if (basicDirty && info.data) {
        await api.connect(line.token, {
          webhookUrl: url.trim(),
          subscribe: events.length ? events : ['MESSAGE'],
          rabbitmqEnable: info.data.rabbitmqEnable ?? '',
          websocketEnable: info.data.websocketEnable ?? '',
          natsEnable: info.data.natsEnable ?? '',
        });
      }
      if (advDirty) {
        const res = await api.saveWebhook(line.id, { headers: headersObj, byEvents, secret: secretAction });
        if (res.secret) setNewSecret(res.secret);
      }
      setSecretAction('keep');
      await Promise.all([
        qc.invalidateQueries({ queryKey: ['overview'] }),
        qc.invalidateQueries({ queryKey: ['info', line.id] }),
        qc.invalidateQueries({ queryKey: ['webhook', line.id] }),
      ]);
      toast(t('webhook.saved'));
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.saveFailed'));
    } finally {
      setSaving(false);
    }
  }

  async function test() {
    setTesting(true);
    try {
      const r = await api.testWebhook(line.id);
      toast(r.ok ? t('webhook.testOk', { code: r.statusCode, ms: i.num(r.durationMs) }) : t('webhook.testFail', { error: r.error || String(r.statusCode) }));
      deliveries.refetch();
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setTesting(false);
    }
  }

  const signatureOn = secretAction === 'new' || (h.hasSecret && secretAction !== 'remove');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <section className="card">
        {!line.webhook && <div className="alert warn">{t('webhook.missing')}</div>}
        <label className="field">
          {t('webhook.url')}
          <input className="input on-surface mono" type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com/webhook" style={{ fontSize: 14 }} />
        </label>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <span style={{ fontSize: 14, fontWeight: 600 }}>{t('webhook.events')}</span>
          <div className="chips">
            {EVENT_KEYS.map((k) => {
              const on = events.includes(k) || events.includes('ALL');
              return (
                <button
                  key={k}
                  type="button"
                  className="evt-toggle"
                  aria-pressed={on}
                  onClick={() => setEvents((ev) => (ev.includes('ALL') ? EVENT_KEYS.filter((x) => x !== k) : on ? ev.filter((x) => x !== k) : [...ev, k]))}
                >
                  {t(`evt.${k}` as Key)}
                </button>
              );
            })}
          </div>
        </div>
        <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>{t('webhook.overwrite')}</p>
      </section>

      <section className="card" style={{ gap: 0, paddingTop: 8 }}>
        <h2 style={{ padding: '12px 0 4px' }}>{t('webhook.advanced')}</h2>
        <div className="setting">
          <div className="txt">
            <b>{t('webhook.byEvents')}</b>
            <span>{t('webhook.byEvents.desc')}</span>
          </div>
          <button type="button" role="switch" className="switch" aria-checked={byEvents} aria-label={t('webhook.byEvents')} onClick={() => setByEvents((v) => !v)}>
            <span />
          </button>
        </div>

        <div className="setting" style={{ alignItems: 'flex-start' }}>
          <div className="txt">
            <b>{t('webhook.signature')}</b>
            <span>{t('webhook.signature.desc')}</span>
            <span className={signatureOn ? '' : 'muted'} style={{ fontWeight: 600, color: signatureOn ? 'var(--ok-fg)' : undefined }}>
              {signatureOn ? t('webhook.signatureOn') : t('webhook.signatureOff')}
            </span>
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <button type="button" className="btn btn-soft btn-sm" onClick={() => setSecretAction('new')}>
              {h.hasSecret ? t('webhook.regenerate') : t('webhook.generate')}
            </button>
            {h.hasSecret && (
              <button type="button" className="btn btn-soft btn-sm" style={{ color: 'var(--down-fg)' }} onClick={() => setSecretAction('remove')}>
                {t('webhook.removeSecret')}
              </button>
            )}
          </div>
        </div>
        {newSecret && (
          <div className="alert info" style={{ marginBottom: 12 }}>
            <span className="grow">
              {t('webhook.secretOnce')}
              <br />
              <span className="mono" style={{ wordBreak: 'break-all' }}>{newSecret}</span>
            </span>
            <button type="button" className="btn btn-light btn-sm" onClick={() => navigator.clipboard.writeText(newSecret).then(() => toast(t('common.copied')))}>
              <Icon name="copiar" size={16} />
            </button>
          </div>
        )}

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, padding: '12px 0 16px' }}>
          <b style={{ fontSize: 15 }}>{t('webhook.headers')}</b>
          {rows.map((r, k) => (
            <div key={k} style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              <input className="input on-surface mono" style={{ flex: '1 1 160px', height: 44, fontSize: 13 }} aria-label={t('webhook.headerName')} placeholder="Authorization" value={r.k} onChange={(e) => setRows((rs) => rs.map((x, j) => (j === k ? { ...x, k: e.target.value } : x)))} />
              <input className="input on-surface mono" style={{ flex: '2 1 220px', height: 44, fontSize: 13 }} aria-label={t('webhook.headerValue')} placeholder="Bearer …" value={r.v} onChange={(e) => setRows((rs) => rs.map((x, j) => (j === k ? { ...x, v: e.target.value } : x)))} />
              <button type="button" className="btn btn-soft btn-icon" aria-label={t('webhook.removeHeader')} onClick={() => setRows((rs) => rs.filter((_, j) => j !== k))}>
                <Icon name="papelera" size={18} />
              </button>
            </div>
          ))}
          <button type="button" className="btn btn-soft btn-sm" style={{ alignSelf: 'flex-start' }} onClick={() => setRows((rs) => [...rs, { k: '', v: '' }])}>
            <Icon name="mas" size={16} />
            {t('webhook.addHeader')}
          </button>
        </div>
        <p className="muted" style={{ margin: '0 0 16px', fontSize: 13, lineHeight: 1.5 }}>
          {t('webhook.retryInfo', { n: h.retry.maxAttempts, s: i.num(Math.round(h.retry.timeoutMs / 1000)) })}
        </p>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button type="button" className="btn btn-dark" disabled={(!basicDirty && !advDirty) || saving} onClick={save}>
            {saving ? <span className="spinner" /> : t('webhook.save')}
          </button>
          <button type="button" className="btn btn-soft" disabled={!line.webhook || testing} onClick={test}>
            {testing ? <span className="spinner" /> : t('webhook.test')}
          </button>
        </div>
      </section>

      <section className="card" style={{ gap: 6 }}>
        <div className="section-head">
          <h2>{t('webhook.deliveries')}</h2>
          <span className="desc">{t('webhook.stats', { ok: i.num(h.stats24h.ok), failed: i.num(h.stats24h.failed) })}</span>
        </div>
        {deliveries.data?.length === 0 && <p className="muted" style={{ margin: 0 }}>{t('webhook.noDeliveries')}</p>}
        <div className="rows">
          {deliveries.data?.map((d) => (
            <div className="row" key={d.id} style={{ padding: '8px 0' }}>
              <span className="mono muted" style={{ width: 120, flexShrink: 0, fontSize: 12.5 }}>
                {new Date(d.at).toLocaleString(i.tag, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
              </span>
              <span className={`ev-chip ${d.ok ? 'ok' : 'down'}`}>{d.statusCode || '—'}</span>
              <span style={{ fontSize: 13.5, fontWeight: 600, width: 130, flexShrink: 0 }}>{d.test ? 'Test' : d.event}</span>
              <span className="muted" style={{ fontSize: 13, flex: '1 1 200px', minWidth: 0 }}>
                {t('webhook.attempts', { n: d.attempts })} · {i.num(d.durationMs)} ms{d.error ? ` · ${d.error}` : ''}
              </span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
