import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';
import { Link, NavLink } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api, type Line, type LineState } from '../api';
import { useAuth } from '../auth';
import { STATE_META, describe, formatPhone, groupOf, initials } from '../format';

// ---------- Iconos (trazo, 24×24) ----------
const PATHS: Record<string, string> = {
  panel: 'M4 4h6v9H4zM14 4h6v5h-6zM14 13h6v7h-6zM4 17h6v3H4z',
  lineas: 'M8 3h8a1.5 1.5 0 0 1 1.5 1.5v15A1.5 1.5 0 0 1 16 21H8a1.5 1.5 0 0 1-1.5-1.5v-15A1.5 1.5 0 0 1 8 3zM11 17.5h2',
  chats: 'M5 5h14a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1H10l-5 4V6a1 1 0 0 1 1-1z',
  eventos: 'M4 12h3l2.5-6 5 12 2.5-6h3',
  salir: 'M15 4h4v16h-4M10 8l-4 4 4 4M6 12h11',
  buscar: 'M15.8 15.8L20 20M17.5 11a6.5 6.5 0 1 1-13 0 6.5 6.5 0 0 1 13 0z',
  mas: 'M12 5v14M5 12h14',
  atras: 'M15 6l-6 6 6 6',
  ir: 'M9 6l6 6-6 6',
  copiar: 'M8 8h12v12H8zM16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3',
  ojo: 'M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12zM15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
};

export function Icon({ name, size = 20 }: { name: keyof typeof PATHS | string; size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={PATHS[name] ?? ''} />
    </svg>
  );
}

export function Logo({ size = 32, dark = false }: { size?: number; dark?: boolean }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
      <rect width="32" height="32" rx="9" fill={dark ? '#262A2F' : '#15171A'} />
      <circle cx="12.5" cy="16" r="5" fill="none" stroke="#7BE0B4" strokeWidth="2.4" />
      <circle cx="19.5" cy="16" r="5" fill="none" stroke="#FFFFFF" strokeWidth="2.4" />
    </svg>
  );
}

// ---------- Datos compartidos ----------
export function useOverview() {
  return useQuery({ queryKey: ['overview'], queryFn: api.overview, refetchInterval: 5000 });
}

// ---------- Avisos breves ----------
const ToastCtx = createContext<(msg: string) => void>(() => {});
export function ToastProvider({ children }: { children: ReactNode }) {
  const [msg, setMsg] = useState<string | null>(null);
  useEffect(() => {
    if (!msg) return;
    const t = setTimeout(() => setMsg(null), 3200);
    return () => clearTimeout(t);
  }, [msg]);
  const show = useCallback((m: string) => setMsg(m), []);
  return (
    <ToastCtx.Provider value={show}>
      {children}
      {msg && (
        <div className="toast" role="status">
          {msg}
        </div>
      )}
    </ToastCtx.Provider>
  );
}
export const useToast = () => useContext(ToastCtx);

// ---------- Armazón ----------
const NAV = [
  { to: '/', label: 'Panel', icon: 'panel', end: true },
  { to: '/lineas', label: 'Líneas', icon: 'lineas', end: false },
  { to: '/eventos', label: 'Eventos', icon: 'eventos', end: false },
];

export function Shell({ children }: { children: ReactNode }) {
  const { session, logout } = useAuth();
  const { data, isError } = useOverview();
  const down = data?.filter((l) => l.state === 'caida').length ?? 0;
  const host = session ? new URL(session.apiUrl).host : '';

  return (
    <div className="shell">
      <aside className="sidebar">
        <Link to="/" className="brand">
          <Logo />
          Enlace
        </Link>
        <nav className="nav" aria-label="Principal">
          {NAV.map((n) => (
            <NavLink key={n.to} to={n.to} end={n.end}>
              <Icon name={n.icon} />
              <span className="grow">{n.label}</span>
              {n.to === '/lineas' && down > 0 && <span className="badge">{down}</span>}
            </NavLink>
          ))}
        </nav>
        <div className="side-foot">
          <div className="server-card">
            <strong>
              <span className="dot" style={{ background: isError ? 'var(--down)' : 'var(--ok)' }} />
              {isError ? 'Servidor sin respuesta' : 'Servidor en línea'}
            </strong>
            <span className="mono">{host}</span>
          </div>
          <button type="button" className="side-link" onClick={logout}>
            <Icon name="salir" />
            Cerrar sesión
          </button>
          <span className="notice">Basado en Evolution Go · Apache 2.0</span>
        </div>
      </aside>

      <main className="content">{children}</main>

      <nav className="tabbar" aria-label="Principal">
        {NAV.map((n) => (
          <NavLink key={n.to} to={n.to} end={n.end}>
            <Icon name={n.icon} size={24} />
            {n.to === '/lineas' && down > 0 && <span className="badge">{down}</span>}
            <span>{n.label}</span>
          </NavLink>
        ))}
        <button type="button" onClick={logout} style={{ border: 0, background: 'transparent', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4, fontSize: 11.5, color: 'var(--ink-3)', minHeight: 52 }}>
          <Icon name="salir" size={24} />
          <span>Salir</span>
        </button>
      </nav>
    </div>
  );
}

// ---------- Estado y tarjetas ----------
export function StatePill({ state, label }: { state: LineState; label?: string }) {
  return (
    <span className={`pill s-${state}`}>
      <span className="dot" />
      {label ?? STATE_META[state].label}
    </span>
  );
}

export function Avatar({ name, big = false }: { name: string; big?: boolean }) {
  return (
    <span aria-hidden="true" className={`avatar g-${groupOf(name)}`} style={big ? { width: 64, height: 64, borderRadius: 18, fontSize: 20 } : undefined}>
      {initials(name)}
    </span>
  );
}

export function LineCard({ line, onReconnect, busy }: { line: Line; onReconnect: (l: Line) => void; busy: boolean }) {
  let action: ReactNode;
  if (line.state === 'caida' && line.needsPairing) {
    action = (
      <Link className="btn btn-dark" to={`/lineas/${line.id}/vincular`}>
        Vincular
      </Link>
    );
  } else if (line.state === 'caida') {
    action = (
      <button type="button" className="btn btn-dark" onClick={() => onReconnect(line)} disabled={busy}>
        {busy ? <span className="spinner" /> : 'Reconectar'}
      </button>
    );
  } else {
    action = (
      <Link className="btn btn-soft" to={`/lineas/${line.id}`}>
        {line.state === 'conectada' ? 'Ver línea' : 'Ver historial'}
      </Link>
    );
  }

  return (
    <article className="line-card">
      <div className="top">
        <Avatar name={line.name} />
        <div className="who">
          <span className="name">{line.name}</span>
          <span className="phone">{formatPhone(line.phone)}</span>
        </div>
        <StatePill state={line.state} />
      </div>
      <p className="detail">{describe(line)}</p>
      <div className="meta">
        {line.webhook ? <span className="tag">Webhook activo</span> : <span className="tag warn">Sin webhook</span>}
      </div>
      <div className="actions">
        {action}
        <Link className="btn btn-soft btn-icon" to={`/lineas/${line.id}`} aria-label={`Ver detalle de ${line.name}`}>
          <Icon name="ir" size={18} />
        </Link>
      </div>
    </article>
  );
}

export function Loading({ label = 'Cargando…' }: { label?: string }) {
  return (
    <div className="empty" role="status">
      <span className="spinner" />
      <span>{label}</span>
    </div>
  );
}

export function ErrorBox({ error }: { error: unknown }) {
  return <div className="alert error">{error instanceof Error ? error.message : 'Algo falló.'}</div>;
}
