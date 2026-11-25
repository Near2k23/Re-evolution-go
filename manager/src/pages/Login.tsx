import { useState, type FormEvent } from 'react';
import { api, defaultApiUrl, loadSession } from '../api';
import { useAuth } from '../auth';
import { Icon, Logo } from '../components/ui';

export default function Login() {
  const { login } = useAuth();
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
      setError(status === 401 ? 'La clave global no es correcta.' : 'No se pudo conectar con el servidor. Revisa la dirección.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="login">
      <section className="hero">
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 20, fontWeight: 650 }}>
          <Logo size={36} dark />
          Enlace
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          <h1>Todas tus líneas de WhatsApp, en un solo lugar.</h1>
          <p>Mira qué líneas siguen conectadas, cuáles volvieron solas tras un corte y cuáles necesitan que las vincules otra vez.</p>
        </div>
        <div className="states">
          <span><span className="dot" style={{ background: 'var(--ok)', width: 8, height: 8 }} />Conectadas</span>
          <span><span className="dot" style={{ background: 'var(--back-soft)', width: 8, height: 8 }} />Reconectadas</span>
          <span><span className="dot" style={{ background: '#F0714F', width: 8, height: 8 }} />Caídas</span>
        </div>
      </section>

      <main>
        <form onSubmit={submit}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <h2 style={{ fontSize: 28 }}>Iniciar sesión</h2>
            <p className="sub">Usa la dirección del servidor y su clave global.</p>
          </div>

          <label className="field">
            Servidor
            <input className="input mono" type="url" required value={apiUrl} onChange={(e) => setApiUrl(e.target.value)} autoComplete="url" />
          </label>

          <label className="field">
            Clave global
            <span style={{ display: 'flex', alignItems: 'center', background: 'var(--surface)', borderRadius: 12, paddingRight: 4 }}>
              <input className="input" type={show ? 'text' : 'password'} required value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="GLOBAL_API_KEY" autoComplete="current-password" style={{ background: 'transparent' }} />
              <button type="button" className="btn btn-icon" style={{ background: 'transparent', color: 'var(--ink-3)', height: 40, width: 40 }} onClick={() => setShow((v) => !v)} aria-label={show ? 'Ocultar clave' : 'Mostrar clave'} aria-pressed={show}>
                <Icon name="ojo" />
              </button>
            </span>
          </label>

          <label style={{ display: 'flex', alignItems: 'center', gap: 10, minHeight: 44, fontSize: 14, color: 'var(--ink-2)' }}>
            <input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} style={{ width: 20, height: 20, margin: 0, accentColor: '#15171A' }} />
            Recordar este servidor en este navegador
          </label>

          {error && <div className="alert error" role="alert">{error}</div>}

          <button type="submit" className="btn btn-dark btn-block" style={{ height: 52, fontSize: 15 }} disabled={busy}>
            {busy ? <span className="spinner" /> : 'Entrar'}
          </button>

          <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>
            La clave global da acceso a todas las líneas del servidor. Para una sola línea, usa su token desde la integración.
          </p>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--ink-4)' }}>Basado en Evolution Go · Apache 2.0</p>
        </form>
      </main>
    </div>
  );
}
