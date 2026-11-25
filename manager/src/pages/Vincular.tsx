import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, type QrInfo } from '../api';
import { useI18n } from '../i18n';
import { describe, eventsList } from '../format';
import { ErrorBox, Icon, Loading, NotFound, StatePill, useOverview, useToast } from '../components/ui';

type Method = 'qr' | 'codigo';

const isPhone = () => typeof window !== 'undefined' && window.matchMedia('(max-width: 899px)').matches;

export default function Vincular() {
  const i = useI18n();
  const { t } = i;
  const { id = '' } = useParams();
  const { data, isLoading, error } = useOverview();
  const [method, setMethod] = useState<Method>(() => (isPhone() ? 'codigo' : 'qr'));
  const line = data?.find((l) => l.id === id);
  const linked = line && (line.state === 'conectada' || line.state === 'reconectada');

  if (isLoading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  if (!line) return <NotFound />;

  return (
    <>
      <Link to="/lineas" className="back-link">
        <Icon name="atras" size={18} />
        {t('nav.lines')}
      </Link>
      <header style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10 }}>
          <h1 style={{ fontSize: 30 }}>{t('link.title', { name: line.name })}</h1>
          <StatePill state={line.state} />
        </div>
        <p className="sub" style={{ maxWidth: 640 }}>
          {linked ? t('link.already') : `${describe(i, line)} ${t('link.handy')}`}
        </p>
      </header>

      {linked ? (
        <div className="card" style={{ alignItems: 'flex-start' }}>
          <h2>{t('link.ready', { name: line.name })}</h2>
          <p className="muted" style={{ margin: 0 }}>{t('link.canSend')}</p>
          <Link className="btn btn-dark" to={`/lineas/${line.id}`}>
            {t('link.viewLine')}
          </Link>
        </div>
      ) : (
        <>
          <div className="segmented" role="tablist" aria-label={t('link.method')}>
            <button type="button" role="tab" aria-selected={method === 'qr'} onClick={() => setMethod('qr')}>
              {t('link.qr')}
            </button>
            <button type="button" role="tab" aria-selected={method === 'codigo'} onClick={() => setMethod('codigo')}>
              {t('link.code')}
            </button>
          </div>
          <div className="split">
            <section className="card main" style={{ alignItems: 'center' }}>
              {method === 'qr' ? <QrPanel token={line.token} /> : <CodePanel token={line.token} events={eventsList(line.events)} initialPhone={line.phone} />}
            </section>
            <aside className="side">
              <section className="card">
                <h2>{t('link.onPhone')}</h2>
                <ol className="steps">
                  <li><span>1</span>{t('link.step1')}</li>
                  <li><span>2</span>{method === 'qr' ? t('link.qr2') : t('link.code2')}</li>
                  <li><span>3</span>{method === 'qr' ? t('link.qr3') : t('link.code3')}</li>
                  <li><span>4</span>{t('link.step4')}</li>
                </ol>
              </section>
              {method === 'codigo' && <p className="muted" style={{ margin: '0 6px', fontSize: 13, lineHeight: 1.5 }}>{t('link.codeTip')}</p>}
            </aside>
          </div>
        </>
      )}
    </>
  );
}

function QrPanel({ token }: { token: string }) {
  const { t } = useI18n();
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
    const id = setInterval(() => setAge((a) => a + 1), 1000);
    return () => clearInterval(id);
  }, []);

  if (q.data?.passkeyStage) {
    return (
      <div className="alert info" style={{ flexDirection: 'column', alignItems: 'flex-start', width: '100%' }}>
        <b style={{ fontSize: 16 }}>{t('link.passkeyTitle')}</b>
        <span>{t('link.passkeyBody')}</span>
        {q.data.passkeyCode && <span className="pair-code" style={{ fontSize: 28 }}>{q.data.passkeyCode}</span>}
        {q.data.passkeyOpenUrl && (
          <a className="btn" style={{ background: '#14306F', color: '#fff' }} href={q.data.passkeyOpenUrl} target="_blank" rel="noreferrer">
            {t('link.openWeb')}
          </a>
        )}
      </div>
    );
  }

  if (!q.data?.qrcode) {
    return (
      <div className="empty" style={{ background: 'transparent' }} role="status">
        <span className="spinner" />
        <span>{q.error ? t('link.waiting') : t('link.preparing')}</span>
      </div>
    );
  }

  const left = Math.max(0, 20 - age);
  return (
    <>
      <div className="qr-box">
        <img src={q.data.qrcode} alt={t('link.qrAlt')} />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true">
          <circle cx="20" cy="20" r="16" fill="none" stroke="#ECEEEA" strokeWidth="4" />
          <circle cx="20" cy="20" r="16" fill="none" stroke="#15171A" strokeWidth="4" strokeLinecap="round" strokeDasharray={`${(left / 20) * 100.5} 101`} transform="rotate(-90 20 20)" />
        </svg>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={{ fontSize: 15, fontWeight: 600 }}>{left > 0 ? t('link.renewsIn', { n: left }) : t('link.renewing')}</span>
          <span className="muted" style={{ fontSize: 13 }}>{t('link.codeN', { n: seen })}</span>
        </div>
      </div>
    </>
  );
}

function CodePanel({ token, events, initialPhone }: { token: string; events: string[]; initialPhone: string }) {
  const { t } = useI18n();
  const toast = useToast();
  const [phone, setPhone] = useState(initialPhone ? '+' + initialPhone : '');
  const [code, setCode] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      setCode(await api.pair(token, phone.replace(/\D/g, ''), events.length ? events : ['MESSAGE']));
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : t('link.requestFailed'));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 16, width: '100%', maxWidth: 400 }}>
      <label className="field">
        {t('link.number')}
        <input className="input on-surface mono" type="tel" required value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+1 555 000 0000" />
      </label>
      <button type="submit" className="btn btn-dark" style={{ height: 50 }} disabled={busy}>
        {busy ? <span className="spinner" /> : code ? t('link.requestAnother') : t('link.request')}
      </button>
      {err && <div className="alert error">{err}</div>}
      {code && (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10, padding: 22, borderRadius: 18, background: 'var(--ground)' }}>
          <span className="muted" style={{ fontSize: 13 }}>{t('link.typeCode')}</span>
          <span className="pair-code">{code.length === 8 ? `${code.slice(0, 4)}-${code.slice(4)}` : code}</span>
          <button type="button" className="btn btn-light btn-sm" onClick={() => navigator.clipboard.writeText(code).then(() => toast(t('common.copied')), () => toast(t('common.copyFailed')))}>
            {t('link.copyCode')}
          </button>
        </div>
      )}
    </form>
  );
}
