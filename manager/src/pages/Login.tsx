import { useState, type FormEvent } from 'react';
import { api, defaultApiUrl, loadSession } from '../api';
import { useAuth } from '../auth';
import { LanguageSelect, useI18n } from '../i18n';
import { Icon, Logo } from '../components/ui';

export default function Login() {
  const { login } = useAuth();
  const { t } = useI18n();
  const [apiUrl, setApiUrl] = useState(() => loadSession()?.apiUrl ?? defaultApiUrl());
  const [apiKey, setApiKey] = useState('');
  const [show, setShow] = useState(false);
  const [remember, setRemember] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    const url = apiUrl.trim().replace(/\/+$/, '');
    try {
      await api.verify(url, apiKey.trim());
      login({ apiUrl: url, apiKey: apiKey.trim() }, remember);
    } catch (err) {
      const status = (err as { status?: number }).status;
      setError(status === 401 ? t('login.wrongKey') : t('login.cantConnect'));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="login">
      <section className="hero">
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 20, fontWeight: 650 }}>
          <Logo size={36} dark />
          {t('app.name')}
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          <h1>{t('login.heroTitle')}</h1>
          <p>{t('login.heroBody')}</p>
        </div>
        <div className="states">
          <span><span className="dot" style={{ background: 'var(--ok)', width: 8, height: 8 }} />{t('state.conectada.title')}</span>
          <span><span className="dot" style={{ background: 'var(--back-soft)', width: 8, height: 8 }} />{t('state.reconectada.title')}</span>
          <span><span className="dot" style={{ background: '#F0714F', width: 8, height: 8 }} />{t('state.caida.title')}</span>
        </div>
      </section>

      <main>
        <form onSubmit={submit}>
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8, flex: 1 }}>
              <h2 style={{ fontSize: 28 }}>{t('login.title')}</h2>
              <p className="sub">{t('login.sub')}</p>
            </div>
            <LanguageSelect compact />
          </div>

          <label className="field">
            {t('login.server')}
            <input className="input mono" type="url" required value={apiUrl} onChange={(e) => setApiUrl(e.target.value)} autoComplete="url" />
          </label>

          <label className="field">
            {t('login.key')}
            <span style={{ display: 'flex', alignItems: 'center', background: 'var(--surface)', borderRadius: 12, paddingRight: 4 }}>
              <input className="input" type={show ? 'text' : 'password'} required value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="GLOBAL_API_KEY" autoComplete="current-password" style={{ background: 'transparent' }} />
              <button type="button" className="btn btn-icon" style={{ background: 'transparent', color: 'var(--ink-3)', height: 40, width: 40 }} onClick={() => setShow((v) => !v)} aria-label={show ? t('login.hideKey') : t('login.showKey')} aria-pressed={show}>
                <Icon name="ojo" />
              </button>
            </span>
          </label>

          <label style={{ display: 'flex', alignItems: 'center', gap: 10, minHeight: 44, fontSize: 14, color: 'var(--ink-2)' }}>
            <input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} style={{ width: 20, height: 20, margin: 0, accentColor: '#15171A' }} />
            {t('login.remember')}
          </label>

          {error && (
            <div className="alert error" role="alert">
              {error}
            </div>
          )}

          <button type="submit" className="btn btn-dark btn-block" style={{ height: 52, fontSize: 15 }} disabled={busy}>
            {busy ? <span className="spinner" /> : t('login.enter')}
          </button>

          <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>{t('login.keyHint')}</p>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--ink-4)' }}>{t('app.basedOn')}</p>
        </form>
      </main>
    </div>
  );
}
