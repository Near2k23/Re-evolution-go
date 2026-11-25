import { useEffect, useState, type FormEvent } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api, type Line } from '../api';
import { useI18n } from '../i18n';
import { useToast } from './ui';

export default function ProxyCard({ line }: { line: Line }) {
  const { t } = useI18n();
  const toast = useToast();
  const q = useQuery({ queryKey: ['proxy', line.id], queryFn: () => api.proxy(line.id) });
  const [form, setForm] = useState({ protocol: 'http', host: '', port: '', username: '', password: '' });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (q.data) setForm({ protocol: q.data.protocol || 'http', host: q.data.host, port: q.data.port, username: q.data.username, password: '' });
  }, [q.data]);

  async function save(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.setProxy(line.id, form);
      await q.refetch();
      toast(t('proxy.saved'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.saveFailed'));
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!confirm(t('proxy.removeConfirm'))) return;
    setBusy(true);
    try {
      await api.removeProxy(line.id);
      setForm({ protocol: 'http', host: '', port: '', username: '', password: '' });
      await q.refetch();
      toast(t('proxy.removed'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.error'));
    } finally {
      setBusy(false);
    }
  }

  const set = (k: keyof typeof form) => (e: { target: { value: string } }) => setForm((f) => ({ ...f, [k]: e.target.value }));

  return (
    <form className="card" onSubmit={save}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        <h2>{t('proxy.title')}</h2>
        <span className="muted" style={{ fontSize: 13.5, lineHeight: 1.5 }}>{q.data ? t('proxy.desc') : t('proxy.none')}</span>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
        <label className="field">
          {t('proxy.protocol')}
          <select className="input on-surface" value={form.protocol} onChange={set('protocol')}>
            <option value="http">HTTP</option>
            <option value="https">HTTPS</option>
            <option value="socks5">SOCKS5</option>
          </select>
        </label>
        <label className="field">
          {t('proxy.host')}
          <input className="input on-surface mono" required value={form.host} onChange={set('host')} placeholder="proxy.example.com" style={{ fontSize: 14 }} />
        </label>
        <label className="field">
          {t('proxy.port')}
          <input className="input on-surface mono" required inputMode="numeric" value={form.port} onChange={set('port')} placeholder="8080" style={{ fontSize: 14 }} />
        </label>
        <label className="field">
          {t('proxy.user')}
          <input className="input on-surface" value={form.username} onChange={set('username')} autoComplete="off" />
        </label>
        <label className="field">
          {t('proxy.password')}
          <input className="input on-surface" type="password" value={form.password} onChange={set('password')} autoComplete="new-password" placeholder={q.data?.hasPassword ? '••••••••' : ''} />
        </label>
      </div>
      {q.data?.hasPassword && <p className="muted" style={{ margin: 0, fontSize: 13 }}>{t('proxy.passwordAgain')}</p>}
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <button type="submit" className="btn btn-dark" disabled={busy}>
          {busy ? <span className="spinner" /> : t('proxy.save')}
        </button>
        {q.data && (
          <button type="button" className="btn btn-soft" style={{ color: 'var(--down-fg)' }} disabled={busy} onClick={remove}>
            {t('proxy.remove')}
          </button>
        )}
      </div>
    </form>
  );
}
