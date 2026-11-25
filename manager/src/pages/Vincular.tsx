import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, type QrInfo } from '../api';
import { describe, eventsList } from '../format';
import { ErrorBox, Icon, Loading, StatePill, useOverview } from '../components/ui';

type Method = 'qr' | 'codigo';

const isPhone = () => typeof window !== 'undefined' && window.matchMedia('(max-width: 899px)').matches;

export default function Vincular() {
  const { id = '' } = useParams();
  const { data, isLoading, error } = useOverview();
  const [method, setMethod] = useState<Method>(() => (isPhone() ? 'codigo' : 'qr'));
  const line = data?.find((l) => l.id === id);
  const linked = line && (line.state === 'conectada' || line.state === 'reconectada');

  if (isLoading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
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
      <header style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
          <h1 style={{ fontSize: 30 }}>Vincular {line.name}</h1>
          <StatePill state={line.state} />
        </div>
        <p className="sub" style={{ maxWidth: 640 }}>
          {linked ? 'La línea ya está vinculada.' : `${describe(line)} Ten a mano el teléfono de esta línea.`}
        </p>
      </header>

      {linked ? (
        <div className="card" style={{ alignItems: 'flex-start' }}>
          <h2>Listo: {line.name} está conectada</h2>
          <p className="muted" style={{ margin: 0 }}>Ya puede enviar y recibir mensajes.</p>
          <Link className="btn btn-dark" to={`/lineas/${line.id}`}>Ver la línea</Link>
        </div>
      ) : (
        <>
          <div className="segmented" role="tablist" aria-label="Método de vinculación">
            <button type="button" role="tab" aria-selected={method === 'qr'} onClick={() => setMethod('qr')}>
              Código QR
            </button>
            <button type="button" role="tab" aria-selected={method === 'codigo'} onClick={() => setMethod('codigo')}>
              Código de 8 caracteres
            </button>
          </div>
          <div className="split">
            <section className="card main" style={{ alignItems: 'center' }}>
              {method === 'qr' ? <QrPanel token={line.token} /> : <CodePanel token={line.token} events={eventsList(line.events)} initialPhone={line.phone} />}
            </section>
            <aside className="side">
              <section className="card">
                <h2>En el teléfono de la línea</h2>
                <ol className="steps">
                  <li><span>1</span>Abre WhatsApp y entra en Ajustes › Dispositivos vinculados.</li>
                  {method === 'qr' ? (
                    <>
                      <li><span>2</span>Toca «Vincular un dispositivo» y desbloquea el teléfono.</li>
                      <li><span>3</span>Apunta la cámara al código QR de esta pantalla.</li>
                    </>
                  ) : (
                    <>
                      <li><span>2</span>Toca «Vincular un dispositivo» y luego «Vincular con el número de teléfono».</li>
                      <li><span>3</span>Escribe el código de 8 caracteres que aparece aquí.</li>
                    </>
                  )}
                  <li><span>4</span>Espera aquí: la pantalla cambia sola cuando la línea quede conectada.</li>
                </ol>
              </section>
              {method === 'codigo' && (
                <p className="muted" style={{ margin: '0 6px', fontSize: 13, lineHeight: 1.5 }}>
                  Usa el código si abriste esta pantalla en el mismo teléfono de la línea: ahí no puedes escanear el QR.
                </p>
              )}
            </aside>
          </div>
        </>
      )}
    </>
  );
}

function QrPanel({ token }: { token: string }) {
  const q = useQuery<QrInfo>({ queryKey: ['qr', token], queryFn: () => api.qr(token), refetchInterval: 4000, retry: false });
  const [seen, setSeen] = useState(0);
  const last = useRef<string | undefined>(undefined);
  const [age, setAge] = useState(0);

  useEffect(() => {
    if (q.data?.code && q.data.code !== last.current) {
      last.current = q.data.code;
      setSeen((n) => n + 1);
      setAge(0);
    }
  }, [q.data?.code]);
  useEffect(() => {
    const t = setInterval(() => setAge((a) => a + 1), 1000);
    return () => clearInterval(t);
  }, []);

  if (q.data?.passkeyStage) {
    return (
      <div className="alert info" style={{ flexDirection: 'column', alignItems: 'flex-start', width: '100%' }}>
        <b style={{ fontSize: 16 }}>WhatsApp pide una llave de acceso</b>
        <span>Abre WhatsApp Web en un computador con la extensión instalada y confirma la passkey. Esta pantalla sigue sola.</span>
        {q.data.passkeyCode && <span className="pair-code" style={{ fontSize: 28 }}>{q.data.passkeyCode}</span>}
        {q.data.passkeyOpenUrl && (
          <a className="btn" style={{ background: '#14306F', color: '#fff' }} href={q.data.passkeyOpenUrl} target="_blank" rel="noreferrer">
            Abrir WhatsApp Web
          </a>
        )}
      </div>
    );
  }

  if (!q.data?.qrcode) {
    return (
      <div className="empty" style={{ background: 'transparent' }} role="status">
        <span className="spinner" />
        <span>{q.error ? 'Esperando el código QR…' : 'Preparando el código QR…'}</span>
      </div>
    );
  }

  const left = Math.max(0, 20 - age);
  return (
    <>
      <div className="qr-box">
        <img src={q.data.qrcode} alt="Código QR para vincular la línea" />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true">
          <circle cx="20" cy="20" r="16" fill="none" stroke="#ECEEEA" strokeWidth="4" />
          <circle cx="20" cy="20" r="16" fill="none" stroke="#15171A" strokeWidth="4" strokeLinecap="round" strokeDasharray={`${(left / 20) * 100.5} 101`} transform="rotate(-90 20 20)" />
        </svg>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={{ fontSize: 15, fontWeight: 600 }}>{left > 0 ? `Se renueva en ${left} s` : 'Renovando…'}</span>
          <span className="muted" style={{ fontSize: 13 }}>Código {seen} · los siguientes llegan solos</span>
        </div>
      </div>
    </>
  );
}

function CodePanel({ token, events, initialPhone }: { token: string; events: string[]; initialPhone: string }) {
  const [phone, setPhone] = useState(initialPhone ? '+' + initialPhone : '');
  const [code, setCode] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      const digits = phone.replace(/\D/g, '');
      const c = await api.pair(token, digits, events.length ? events : ['MESSAGE']);
      setCode(c);
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : 'No se pudo pedir el código');
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 16, width: '100%', maxWidth: 400 }}>
      <label className="field">
        Número de la línea
        <input className="input on-surface mono" type="tel" required value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+57 300 000 0000" />
      </label>
      <button type="submit" className="btn btn-dark" style={{ height: 50 }} disabled={busy}>
        {busy ? <span className="spinner" /> : code ? 'Pedir otro código' : 'Pedir código'}
      </button>
      {err && <div className="alert error">{err}</div>}
      {code && (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10, padding: 22, borderRadius: 18, background: 'var(--ground)' }}>
          <span className="muted" style={{ fontSize: 13 }}>Escribe este código en el teléfono</span>
          <span className="pair-code">{code.length === 8 ? `${code.slice(0, 4)}-${code.slice(4)}` : code}</span>
          <button type="button" className="btn btn-light btn-sm" onClick={() => navigator.clipboard.writeText(code)}>
            Copiar código
          </button>
        </div>
      )}
    </form>
  );
}
