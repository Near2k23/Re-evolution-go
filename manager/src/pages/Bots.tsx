import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type Bot, type BotType, type ChatwootConfig, type Line } from '../api';
import { useI18n, type Key } from '../i18n';
import { byName, formatPhone } from '../format';
import { ErrorBox, Icon, Loading, StatePill, useOverview, useToast } from '../components/ui';

type Tab = 'bots' | 'sessions' | 'chatwoot';
const TYPES: BotType[] = ['http', 'n8n', 'openai', 'typebot', 'dify', 'flowise'];
const MASK = '********';

type Field = { key: string; label: Key; kind?: 'text' | 'secret' | 'number' | 'textarea' | 'switch' | 'select'; options?: string[]; placeholder?: string; hint?: Key; required?: boolean };
const FIELDS: Record<BotType, Field[]> = {
  http: [
    { key: 'url', label: 'bots.f.url', placeholder: 'https://example.com/bot', required: true },
    { key: 'apiKey', label: 'bots.f.apiKey', kind: 'secret' },
  ],
  n8n: [
    { key: 'url', label: 'bots.f.webhookUrl', placeholder: 'https://n8n.example.com/webhook/…', required: true },
    { key: 'username', label: 'bots.f.username' },
    { key: 'password', label: 'bots.f.password', kind: 'secret' },
  ],
  openai: [
    { key: 'apiKey', label: 'bots.f.apiKey', kind: 'secret', required: true },
    { key: 'model', label: 'bots.f.model', placeholder: 'gpt-4o-mini' },
    { key: 'systemPrompt', label: 'bots.f.systemPrompt', kind: 'textarea' },
    { key: 'baseUrl', label: 'bots.f.baseUrl', placeholder: 'https://api.openai.com/v1', hint: 'bots.f.baseUrl.hint' },
    { key: 'historyLimit', label: 'bots.f.historyLimit', kind: 'number' },
    { key: 'maxTokens', label: 'bots.f.maxTokens', kind: 'number' },
    { key: 'temperature', label: 'bots.f.temperature', kind: 'number' },
    { key: 'transcribe', label: 'bots.f.transcribe', kind: 'switch', hint: 'bots.f.transcribe.hint' },
  ],
  typebot: [
    { key: 'url', label: 'bots.f.typebotUrl', placeholder: 'https://typebot.io', required: true },
    { key: 'publicId', label: 'bots.f.publicId', required: true },
  ],
  dify: [
    { key: 'apiUrl', label: 'bots.f.apiUrl', placeholder: 'https://api.dify.ai/v1' },
    { key: 'apiKey', label: 'bots.f.apiKey', kind: 'secret', required: true },
    { key: 'botType', label: 'bots.f.difyType', kind: 'select', options: ['chat', 'agent', 'workflow', 'completion'] },
  ],
  flowise: [
    { key: 'apiUrl', label: 'bots.f.predictionUrl', placeholder: 'https://flowise.example.com/api/v1/prediction/…', required: true },
    { key: 'apiKey', label: 'bots.f.apiKey', kind: 'secret' },
  ],
};

const emptyBot = (): Omit<Bot, 'id' | 'instanceId'> => ({
  type: 'http',
  name: '',
  enabled: true,
  priority: 0,
  triggerType: 'all',
  triggerOperator: 'contains',
  triggerValue: '',
  allowGroups: false,
  listenFromMe: false,
  stopOnFromMe: true,
  keywordFinish: '',
  expireMinutes: 30,
  delayMs: 1200,
  debounceMs: 0,
  splitMessages: true,
  unknownMessage: '',
  ignoreJids: [],
  config: {},
});

export default function Bots() {
  const { t } = useI18n();
  const { lineId } = useParams();
  const navigate = useNavigate();
  const { data: lines, isLoading } = useOverview();
  const sorted = useMemo(() => (lines ?? []).slice().sort(byName), [lines]);
  const line = sorted.find((l) => l.id === lineId);
  const [tab, setTab] = useState<Tab>('bots');

  useEffect(() => {
    if (!lineId && sorted.length) navigate(`/bots/${sorted[0].id}`, { replace: true });
  }, [lineId, sorted, navigate]);

  if (isLoading) return <Loading />;
  if (!sorted.length) {
    return (
      <div className="empty">
        <b>{t('lines.emptyTitle')}</b>
        <Link className="btn btn-dark" to="/lineas">{t('nav.lines')}</Link>
      </div>
    );
  }
  if (!line) return <Loading />;

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <h1>{t('bots.title')}</h1>
          <p className="sub">{t('bots.sub')}</p>
        </div>
        <label className="field" style={{ flexDirection: 'row', alignItems: 'center', fontSize: 13, color: 'var(--ink-3)' }}>
          {t('chats.line')}
          <select className="input" style={{ height: 44, width: 'auto', fontSize: 14, fontWeight: 600 }} value={line.id} onChange={(e) => navigate(`/bots/${e.target.value}`)}>
            {sorted.map((l) => (
              <option key={l.id} value={l.id}>{l.name}</option>
            ))}
          </select>
        </label>
      </header>
      <div className="segmented" role="tablist" aria-label={t('bots.title')}>
        {(['bots', 'sessions', 'chatwoot'] as Tab[]).map((k) => (
          <button key={k} type="button" role="tab" aria-selected={tab === k} onClick={() => setTab(k)}>
            {t(`bots.tab.${k}` as Key)}
          </button>
        ))}
      </div>
      {tab === 'bots' && <BotList line={line} />}
      {tab === 'sessions' && <Sessions line={line} />}
      {tab === 'chatwoot' && <Chatwoot line={line} />}
    </>
  );
}

function Toggle({ label, desc, value, onChange }: { label: string; desc?: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="setting">
      <div className="txt">
        <b>{label}</b>
        {desc && <span>{desc}</span>}
      </div>
      <button type="button" role="switch" className="switch" aria-checked={value} aria-label={label} onClick={() => onChange(!value)}>
        <span />
      </button>
    </div>
  );
}

function BotList({ line }: { line: Line }) {
  const { t } = useI18n();
  const qc = useQueryClient();
  const toast = useToast();
  const q = useQuery({ queryKey: ['bots', line.id], queryFn: () => api.bots(line.id) });
  const [editing, setEditing] = useState<Bot | 'new' | null>(null);

  async function toggle(b: Bot) {
    try {
      const { id: _id, instanceId: _i, ...rest } = b;
      await api.updateBot(line.id, b.id, { ...rest, enabled: !b.enabled });
      await q.refetch();
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.saveFailed'));
    }
  }
  async function remove(b: Bot) {
    if (!confirm(t('bots.deleteConfirm', { name: b.name }))) return;
    await api.deleteBot(line.id, b.id);
    await qc.invalidateQueries({ queryKey: ['bots', line.id] });
    toast(t('bots.deleted'));
  }

  if (editing) return <BotEditor line={line} bot={editing === 'new' ? null : editing} onDone={() => { setEditing(null); q.refetch(); }} />;
  if (q.isLoading) return <Loading />;
  if (q.error) return <ErrorBox error={q.error} />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
      <div>
        <button type="button" className="btn btn-dark" onClick={() => setEditing('new')}>
          <Icon name="mas" size={18} />
          {t('bots.new')}
        </button>
      </div>
      {q.data?.length === 0 && (
        <div className="empty">
          <b>{t('bots.emptyTitle')}</b>
          <span>{t('bots.emptyHint')}</span>
        </div>
      )}
      <div className="grid-cards">
        {q.data?.map((b) => (
          <article key={b.id} className="line-card">
            <div className="top">
              <span className="avatar" style={{ background: 'var(--ground)', color: 'var(--ink)' }} aria-hidden="true">
                <Icon name="chats" size={18} />
              </span>
              <div className="who">
                <span className="name">{b.name}</span>
                <span className="phone">{t(`bots.type.${b.type}` as Key)}</span>
              </div>
              <button type="button" role="switch" className="switch" aria-checked={b.enabled} aria-label={t('bots.enabled')} onClick={() => toggle(b)}>
                <span />
              </button>
            </div>
            <p className="detail">
              {b.triggerType === 'all' ? t('bots.trigger.allDesc') : t('bots.trigger.keywordDesc', { op: t(`bots.op.${b.triggerOperator}` as Key), value: b.triggerValue })}
            </p>
            <div className="meta">
              {b.allowGroups && <span className="tag">{t('bots.allowGroups')}</span>}
              {b.stopOnFromMe && <span className="tag">{t('bots.stopOnFromMe')}</span>}
              {b.keywordFinish && <span className="tag">{t('bots.keywordFinish')}: {b.keywordFinish}</span>}
            </div>
            <div className="actions">
              <button type="button" className="btn btn-soft" onClick={() => setEditing(b)}>{t('bots.edit')}</button>
              <button type="button" className="btn btn-soft btn-icon" aria-label={t('common.delete')} onClick={() => remove(b)}>
                <Icon name="papelera" size={18} />
              </button>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}

function BotEditor({ line, bot, onDone }: { line: Line; bot: Bot | null; onDone: () => void }) {
  const { t } = useI18n();
  const toast = useToast();
  const [b, setB] = useState<Omit<Bot, 'id' | 'instanceId'>>(() => (bot ? { ...bot } : emptyBot()));
  const [ignore, setIgnore] = useState((bot?.ignoreJids ?? []).join('\n'));
  const [saving, setSaving] = useState(false);
  const set = <K extends keyof typeof b>(k: K, v: (typeof b)[K]) => setB((x) => ({ ...x, [k]: v }));
  const setCfg = (k: string, v: unknown) => setB((x) => ({ ...x, config: { ...x.config, [k]: v } }));

  async function save(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    const body = { ...b, ignoreJids: ignore.split(/[\n,]/).map((x) => x.trim()).filter(Boolean) };
    try {
      if (bot) await api.updateBot(line.id, bot.id, body);
      else await api.createBot(line.id, body);
      toast(t('bots.saved'));
      onDone();
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.saveFailed'));
    } finally {
      setSaving(false);
    }
  }

  const num = (v: string) => (v === '' ? 0 : Number(v));

  return (
    <form onSubmit={save} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <button type="button" className="back-link" style={{ border: 0, background: 'transparent', padding: 0 }} onClick={onDone}>
        <Icon name="atras" size={18} />
        {t('bots.tab.bots')}
      </button>

      <section className="card">
        <h2>{bot ? t('bots.edit') : t('bots.new')}</h2>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12 }}>
          <label className="field">
            {t('bots.name')}
            <input className="input on-surface" required value={b.name} onChange={(e) => set('name', e.target.value)} />
          </label>
          <label className="field">
            {t('bots.typeLabel')}
            <select className="input on-surface" value={b.type} onChange={(e) => set('type', e.target.value as BotType)} disabled={!!bot}>
              {TYPES.map((x) => (
                <option key={x} value={x}>{t(`bots.type.${x}` as Key)}</option>
              ))}
            </select>
          </label>
          <label className="field">
            {t('bots.priority')}
            <input className="input on-surface" type="number" value={b.priority} onChange={(e) => set('priority', num(e.target.value))} />
          </label>
        </div>
        <p className="muted" style={{ margin: 0, fontSize: 13.5, lineHeight: 1.5 }}>{t(`bots.type.${b.type}.desc` as Key)}</p>
        <Toggle label={t('bots.enabled')} value={b.enabled} onChange={(v) => set('enabled', v)} />
      </section>

      <section className="card">
        <h2>{t('bots.section.connection')}</h2>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 12 }}>
          {FIELDS[b.type].map((f) => {
            const v = b.config[f.key];
            let control: ReactNode;
            if (f.kind === 'switch') {
              return (
                <div key={f.key} style={{ gridColumn: '1 / -1' }}>
                  <Toggle label={t(f.label)} desc={f.hint ? t(f.hint) : undefined} value={!!v} onChange={(x) => setCfg(f.key, x)} />
                </div>
              );
            }
            if (f.kind === 'textarea') {
              control = <textarea className="input on-surface" rows={5} value={(v as string) ?? ''} onChange={(e) => setCfg(f.key, e.target.value)} />;
            } else if (f.kind === 'select') {
              control = (
                <select className="input on-surface" value={(v as string) ?? f.options![0]} onChange={(e) => setCfg(f.key, e.target.value)}>
                  {f.options!.map((o) => (
                    <option key={o} value={o}>{t(`bots.dify.${o}` as Key)}</option>
                  ))}
                </select>
              );
            } else {
              control = (
                <input
                  className="input on-surface"
                  type={f.kind === 'secret' ? 'password' : f.kind === 'number' ? 'number' : 'text'}
                  step={f.key === 'temperature' ? '0.1' : undefined}
                  required={f.required && !(f.kind === 'secret' && v === MASK)}
                  placeholder={f.placeholder}
                  value={v === undefined || v === null ? '' : String(v)}
                  autoComplete="off"
                  onChange={(e) => setCfg(f.key, f.kind === 'number' ? (e.target.value === '' ? undefined : Number(e.target.value)) : e.target.value)}
                />
              );
            }
            return (
              <label key={f.key} className="field" style={f.kind === 'textarea' ? { gridColumn: '1 / -1' } : undefined}>
                {t(f.label)}
                {control}
                {f.hint && <span className="muted" style={{ fontSize: 12.5, fontWeight: 400 }}>{t(f.hint)}</span>}
              </label>
            );
          })}
        </div>
      </section>

      <section className="card">
        <h2>{t('bots.section.trigger')}</h2>
        <div className="segmented" role="group">
          <button type="button" aria-pressed={b.triggerType === 'all'} onClick={() => set('triggerType', 'all')}>{t('bots.trigger.all')}</button>
          <button type="button" aria-pressed={b.triggerType === 'keyword'} onClick={() => set('triggerType', 'keyword')}>{t('bots.trigger.keyword')}</button>
        </div>
        {b.triggerType === 'keyword' && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12 }}>
            <label className="field">
              {t('bots.operator')}
              <select className="input on-surface" value={b.triggerOperator} onChange={(e) => set('triggerOperator', e.target.value as Bot['triggerOperator'])}>
                {(['contains', 'equals', 'startsWith', 'endsWith', 'regex'] as const).map((o) => (
                  <option key={o} value={o}>{t(`bots.op.${o}` as Key)}</option>
                ))}
              </select>
            </label>
            <label className="field">
              {t('bots.triggerValue')}
              <input className="input on-surface" required value={b.triggerValue} onChange={(e) => set('triggerValue', e.target.value)} />
            </label>
          </div>
        )}
        <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>{t('bots.trigger.hint')}</p>
      </section>

      <section className="card" style={{ gap: 0 }}>
        <h2 style={{ paddingBottom: 8 }}>{t('bots.section.behavior')}</h2>
        <Toggle label={t('bots.allowGroups')} desc={t('bots.allowGroups.desc')} value={b.allowGroups} onChange={(v) => set('allowGroups', v)} />
        <Toggle label={t('bots.stopOnFromMe')} desc={t('bots.stopOnFromMe.desc')} value={b.stopOnFromMe} onChange={(v) => set('stopOnFromMe', v)} />
        <Toggle label={t('bots.listenFromMe')} desc={t('bots.listenFromMe.desc')} value={b.listenFromMe} onChange={(v) => set('listenFromMe', v)} />
        <Toggle label={t('bots.splitMessages')} desc={t('bots.splitMessages.desc')} value={b.splitMessages} onChange={(v) => set('splitMessages', v)} />
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 12, padding: '12px 0' }}>
          <label className="field">
            {t('bots.keywordFinish')}
            <input className="input on-surface" value={b.keywordFinish} onChange={(e) => set('keywordFinish', e.target.value)} placeholder="#salir" />
          </label>
          <label className="field">
            {t('bots.expireMinutes')}
            <input className="input on-surface" type="number" min={0} value={b.expireMinutes} onChange={(e) => set('expireMinutes', num(e.target.value))} />
          </label>
          <label className="field">
            {t('bots.delayMs')}
            <input className="input on-surface" type="number" min={0} step={100} value={b.delayMs} onChange={(e) => set('delayMs', num(e.target.value))} />
          </label>
          <label className="field">
            {t('bots.debounceMs')}
            <input className="input on-surface" type="number" min={0} step={500} value={b.debounceMs} onChange={(e) => set('debounceMs', num(e.target.value))} />
          </label>
        </div>
        <label className="field" style={{ paddingBottom: 12 }}>
          {t('bots.unknownMessage')}
          <input className="input on-surface" value={b.unknownMessage} onChange={(e) => set('unknownMessage', e.target.value)} placeholder={t('bots.unknownMessage.placeholder')} />
        </label>
        <label className="field" style={{ paddingBottom: 12 }}>
          {t('bots.ignoreJids')}
          <textarea className="input on-surface mono" rows={3} value={ignore} onChange={(e) => setIgnore(e.target.value)} placeholder={'+573001112233\n120363…@g.us'} style={{ fontSize: 13 }} />
        </label>
      </section>

      <div style={{ display: 'flex', gap: 8 }}>
        <button type="submit" className="btn btn-dark" disabled={saving}>{saving ? <span className="spinner" /> : t('common.save')}</button>
        <button type="button" className="btn btn-soft" onClick={onDone}>{t('common.cancel')}</button>
      </div>
    </form>
  );
}

function Sessions({ line }: { line: Line }) {
  const i = useI18n();
  const { t } = i;
  const toast = useToast();
  const sessions = useQuery({ queryKey: ['botSessions', line.id], queryFn: () => api.botSessions(line.id), refetchInterval: 5000 });
  const bots = useQuery({ queryKey: ['bots', line.id], queryFn: () => api.bots(line.id) });
  const botName = (id: string) => bots.data?.find((b) => b.id === id)?.name ?? '—';

  async function setStatus(id: string, status: 'open' | 'paused' | 'closed') {
    try {
      await api.setSession(line.id, id, status);
      await sessions.refetch();
    } catch (e) {
      toast(e instanceof Error ? e.message : t('common.error'));
    }
  }

  if (sessions.isLoading) return <Loading />;
  if (sessions.error) return <ErrorBox error={sessions.error} />;
  if (!sessions.data?.length) {
    return (
      <div className="empty">
        <b>{t('bots.sessions.empty')}</b>
        <span>{t('bots.sessions.emptyHint')}</span>
      </div>
    );
  }
  return (
    <section className="card" style={{ gap: 0, paddingTop: 8, paddingBottom: 8 }}>
      {sessions.data.map((s) => (
        <div key={s.id} className="row" style={{ padding: '12px 0' }}>
          <div style={{ flex: '1 1 220px', minWidth: 0, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <b style={{ fontSize: 14.5 }}>{s.pushName || formatPhone(s.remoteJid.split('@')[0])}</b>
            <span className="muted" style={{ fontSize: 13 }}>{botName(s.botId)} · {i.ago(s.lastActivity)}</span>
          </div>
          <StatePill state={s.status === 'open' ? 'conectada' : 'reconectando'} label={t(`bots.session.${s.status}` as Key)} />
          <div style={{ display: 'flex', gap: 6 }}>
            {s.status === 'open' ? (
              <button type="button" className="btn btn-soft btn-sm" onClick={() => setStatus(s.id, 'paused')}>{t('bots.session.pause')}</button>
            ) : (
              <button type="button" className="btn btn-soft btn-sm" onClick={() => setStatus(s.id, 'open')}>{t('bots.session.resume')}</button>
            )}
            <button type="button" className="btn btn-soft btn-sm" onClick={() => setStatus(s.id, 'closed')}>{t('bots.session.close')}</button>
          </div>
        </div>
      ))}
    </section>
  );
}

function Chatwoot({ line }: { line: Line }) {
  const { t } = useI18n();
  const toast = useToast();
  const q = useQuery({ queryKey: ['chatwoot', line.id], queryFn: () => api.chatwoot(line.id) });
  const [c, setC] = useState<ChatwootConfig | null>(null);
  const [token, setToken] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (q.data) setC(q.data);
  }, [q.data]);
  if (q.isLoading || !c) return q.error ? <ErrorBox error={q.error} /> : <Loading />;
  const set = <K extends keyof ChatwootConfig>(k: K, v: ChatwootConfig[K]) => setC((x) => (x ? { ...x, [k]: v } : x));

  async function save(e: FormEvent) {
    e.preventDefault();
    if (!c) return;
    setBusy(true);
    try {
      const { hasToken: _h, webhookUrl: _w, ...rest } = c;
      const saved = await api.saveChatwoot(line.id, { ...rest, token });
      setC(saved);
      setToken('');
      toast(t('cw.saved'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.saveFailed'));
    } finally {
      setBusy(false);
    }
  }
  async function test() {
    if (!c) return;
    setBusy(true);
    try {
      const r = await api.testChatwoot(line.id, c.url, c.accountId, token);
      toast(r.ok ? t('cw.testOk') : t('cw.testFail', { error: r.error ?? '' }));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="card" onSubmit={save} style={{ gap: 0 }}>
      <h2 style={{ paddingBottom: 4 }}>Chatwoot</h2>
      <p className="muted" style={{ margin: '0 0 8px', fontSize: 13.5, lineHeight: 1.5 }}>{t('cw.desc')}</p>
      <Toggle label={t('cw.enabled')} value={c.enabled} onChange={(v) => set('enabled', v)} />
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12, padding: '12px 0' }}>
        <label className="field">
          {t('cw.url')}
          <input className="input on-surface" type="url" required={c.enabled} value={c.url} onChange={(e) => set('url', e.target.value)} placeholder="https://chatwoot.example.com" />
        </label>
        <label className="field">
          {t('cw.accountId')}
          <input className="input on-surface" type="number" min={1} required={c.enabled} value={c.accountId || ''} onChange={(e) => set('accountId', Number(e.target.value))} />
        </label>
        <label className="field">
          {t('cw.token')}
          <input className="input on-surface" type="password" autoComplete="off" required={c.enabled && !c.hasToken} value={token} onChange={(e) => setToken(e.target.value)} placeholder={c.hasToken ? MASK : ''} />
        </label>
        <label className="field">
          {t('cw.inboxName')}
          <input className="input on-surface" value={c.inboxName} onChange={(e) => set('inboxName', e.target.value)} placeholder="WhatsApp" />
        </label>
      </div>
      <p className="muted" style={{ margin: '0 0 8px', fontSize: 13, lineHeight: 1.5 }}>
        {c.inboxId ? t('cw.inboxLinked', { id: c.inboxId }) : t('cw.inboxAuto')}
      </p>
      <Toggle label={t('cw.signMessages')} desc={t('cw.signMessages.desc')} value={c.signMessages} onChange={(v) => set('signMessages', v)} />
      <Toggle label={t('cw.reopen')} desc={t('cw.reopen.desc')} value={c.reopenConversation} onChange={(v) => set('reopenConversation', v)} />
      <Toggle label={t('cw.pending')} desc={t('cw.pending.desc')} value={c.conversationPending} onChange={(v) => set('conversationPending', v)} />
      <Toggle label={t('cw.ignoreGroups')} value={c.ignoreGroups} onChange={(v) => set('ignoreGroups', v)} />
      {c.webhookUrl && (
        <div className="field" style={{ padding: '12px 0' }}>
          {t('cw.webhookUrl')}
          <span className="mono muted" style={{ fontSize: 12.5, fontWeight: 400, wordBreak: 'break-all' }}>{c.webhookUrl}</span>
        </div>
      )}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', paddingTop: 8 }}>
        <button type="submit" className="btn btn-dark" disabled={busy}>{busy ? <span className="spinner" /> : t('common.save')}</button>
        <button type="button" className="btn btn-soft" disabled={busy || !c.url || !c.accountId} onClick={test}>{t('cw.test')}</button>
      </div>
    </form>
  );
}
