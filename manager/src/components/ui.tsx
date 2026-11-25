import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';
import { Link, NavLink } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type Group, type Line, type LineState } from '../api';
import { useAuth } from '../auth';
import { LanguageSelect, useI18n } from '../i18n';
import { describe, formatPhone, groupColors, initials, stateLabel } from '../format';

// ---------- Iconos (trazo, 24×24) ----------
const PATHS: Record<string, string> = {
  panel: 'M4 4h6v9H4zM14 4h6v5h-6zM14 13h6v7h-6zM4 17h6v3H4z',
  lineas: 'M8 3h8a1.5 1.5 0 0 1 1.5 1.5v15A1.5 1.5 0 0 1 16 21H8a1.5 1.5 0 0 1-1.5-1.5v-15A1.5 1.5 0 0 1 8 3zM11 17.5h2',
  grupos: 'M4 6h7v5H4zM13 6h7v5h-7zM4 13h7v5H4zM13 13h7v5h-7z',
  eventos: 'M4 12h3l2.5-6 5 12 2.5-6h3',
  salir: 'M15 4h4v16h-4M10 8l-4 4 4 4M6 12h11',
  buscar: 'M15.8 15.8L20 20M17.5 11a6.5 6.5 0 1 1-13 0 6.5 6.5 0 0 1 13 0z',
  mas: 'M12 5v14M5 12h14',
  atras: 'M15 6l-6 6 6 6',
  ir: 'M9 6l6 6-6 6',
  copiar: 'M8 8h12v12H8zM16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3',
  ojo: 'M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12zM15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
  recargar: 'M20 11a8 8 0 1 0-2.3 5.7M20 5v6h-6',
  lapiz: 'M4 20h4l10.5-10.5a2.1 2.1 0 0 0-3-3L5 17v3z',
  papelera: 'M5 7h14M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3',
  enviar: 'M5 12h13M13 6l6 6-6 6',
  bot: 'M8 8h8a3 3 0 0 1 3 3v5a3 3 0 0 1-3 3H8a3 3 0 0 1-3-3v-5a3 3 0 0 1 3-3zM12 4v4M9.5 13h.01M14.5 13h.01',
  chats: 'M5 5h14a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1H10l-5 4V6a1 1 0 0 1 1-1z',
};

export function Icon({ name, size = 20 }: { name: string; size?: number }) {
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

export function useGroups() {
  return useQuery({ queryKey: ['groups'], queryFn: api.groups, staleTime: 10000 });
}

/** Reconecta una o varias líneas y avisa si siguen caídas pasados unos segundos. */
export function useReconnect() {
  const qc = useQueryClient();
  const toast = useToast();
  const i = useI18n();
  const [busy, setBusy] = useState<Record<string, boolean>>({});

  const run = useCallback(
    async (list: Line[]) => {
      if (!list.length) return;
      setBusy((b) => ({ ...b, ...Object.fromEntries(list.map((l) => [l.id, true])) }));
      const results = await Promise.allSettled(list.map((l) => api.reconnect(l.id)));
      setBusy((b) => ({ ...b, ...Object.fromEntries(list.map((l) => [l.id, false])) }));
      const ok = results.filter((r) => r.status === 'fulfilled').length;
      if (list.length === 1) {
        toast(ok ? i.t('lines.reconnecting', { name: list[0].name }) : i.t('lines.reconnectFailed', { name: list[0].name }));
      } else {
        toast(i.t('lines.bulk', { ok: i.num(ok), n: i.num(list.length) }));
      }
      await qc.invalidateQueries({ queryKey: ['overview'] });
      // Comprobación posterior: si una sola línea sigue caída, se dice.
      if (list.length === 1 && ok) {
        setTimeout(async () => {
          const fresh = await qc.fetchQuery({ queryKey: ['overview'], queryFn: api.overview });
          const l = fresh.find((x) => x.id === list[0].id);
          if (l && l.state === 'caida') toast(i.t('card.stillDown', { name: l.name }));
        }, 8000);
      }
    },
    [qc, toast, i],
  );
  return { run, busy };
}

// ---------- Avisos breves ----------
const ToastCtx = createContext<(msg: string) => void>(() => {});
export function ToastProvider({ children }: { children: ReactNode }) {
  const [msg, setMsg] = useState<string | null>(null);
  useEffect(() => {
    if (!msg) return;
    const t = setTimeout(() => setMsg(null), 4000);
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
export function Shell({ children }: { children: ReactNode }) {
  const { session, logout } = useAuth();
  const { t } = useI18n();
  const { data, isError } = useOverview();
  const down = data?.filter((l) => l.state === 'caida').length ?? 0;
  const host = session ? new URL(session.apiUrl).host : '';
  const nav = [
    { to: '/', label: t('nav.panel'), icon: 'panel', end: true },
    { to: '/lineas', label: t('nav.lines'), icon: 'lineas', end: false },
    { to: '/chats', label: t('nav.chats'), icon: 'chats', end: false },
    { to: '/bots', label: t('nav.bots'), icon: 'bot', end: false },
    { to: '/grupos', label: t('nav.groups'), icon: 'grupos', end: false },
    { to: '/eventos', label: t('nav.events'), icon: 'eventos', end: false },
  ];

  return (
    <div className="shell">
      <aside className="sidebar">
        <Link to="/" className="brand">
          <Logo />
          {t('app.name')}
        </Link>
        <nav className="nav" aria-label={t('nav.main')}>
          {nav.map((n) => (
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
              {isError ? t('server.down') : t('server.online')}
            </strong>
            <span className="mono">{host}</span>
          </div>
          <LanguageSelect />
          <button type="button" className="side-link" onClick={logout}>
            <Icon name="salir" />
            {t('nav.logout')}
          </button>
          <span className="notice">{t('app.basedOn')}</span>
        </div>
      </aside>

      <main className="content">
        <div className="mobile-top">
          <Link to="/" className="brand" style={{ padding: 0 }}>
            <Logo size={28} />
            {t('app.name')}
          </Link>
          <LanguageSelect compact />
        </div>
        {children}
      </main>

      <nav className="tabbar" aria-label={t('nav.main')}>
        {nav.map((n) => (
          <NavLink key={n.to} to={n.to} end={n.end}>
            <Icon name={n.icon} size={24} />
            {n.to === '/lineas' && down > 0 && <span className="badge">{down}</span>}
            <span>{n.label}</span>
          </NavLink>
        ))}
        <button type="button" className="tab-exit" onClick={logout}>
          <Icon name="salir" size={24} />
          <span>{t('nav.logoutShort')}</span>
        </button>
      </nav>
    </div>
  );
}

// ---------- Estado y tarjetas ----------
export function StatePill({ state, label }: { state: LineState; label?: string }) {
  const i = useI18n();
  return (
    <span className={`pill s-${state}`}>
      <span className="dot" />
      {label ?? stateLabel(i, state)}
    </span>
  );
}

export function Avatar({ name, group, big = false }: { name: string; group?: Group | null; big?: boolean }) {
  const c = groupColors(group);
  return (
    <span aria-hidden="true" className="avatar" style={{ background: c.bg, color: c.fg, ...(big ? { width: 64, height: 64, borderRadius: 18, fontSize: 20 } : {}) }}>
      {initials(name)}
    </span>
  );
}

export function GroupTag({ group }: { group?: Group | null }) {
  if (!group) return null;
  const c = groupColors(group);
  return (
    <span className="tag" style={{ background: c.bg, color: c.fg }}>
      {group.name}
    </span>
  );
}

export function LineCard({ line, group, onReconnect, busy }: { line: Line; group?: Group | null; onReconnect: (l: Line) => void; busy: boolean }) {
  const i = useI18n();
  const { t } = i;
  const spin = <span className="spinner" />;
  let primary: ReactNode;
  let secondary: ReactNode = null;

  if (line.state === 'caida' && line.needsPairing) {
    primary = (
      <Link className="btn btn-dark" to={`/lineas/${line.id}/vincular`}>
        {t('card.link')}
      </Link>
    );
    // Puede seguir vinculada aunque parezca que no: se deja intentar.
    if (line.jid) {
      secondary = (
        <button type="button" className="btn btn-soft" onClick={() => onReconnect(line)} disabled={busy} title={t('card.tryReconnectHint')}>
          {busy ? spin : t('card.tryReconnect')}
        </button>
      );
    }
  } else if (line.state === 'caida') {
    primary = (
      <button type="button" className="btn btn-dark" onClick={() => onReconnect(line)} disabled={busy}>
        {busy ? spin : t('card.reconnect')}
      </button>
    );
  } else if (line.state === 'reconectando') {
    primary = (
      <button type="button" className="btn btn-dark" onClick={() => onReconnect(line)} disabled={busy}>
        {busy ? spin : t('card.reconnectNow')}
      </button>
    );
  } else {
    primary = (
      <Link className="btn btn-soft" to={`/lineas/${line.id}`}>
        {line.state === 'conectada' ? t('card.viewLine') : t('card.viewHistory')}
      </Link>
    );
  }

  return (
    <article className="line-card">
      <div className="top">
        <Avatar name={line.name} group={group} />
        <div className="who">
          <span className="name">{line.name}</span>
          <span className="phone">{line.phone ? formatPhone(line.phone) : t('card.noNumber')}</span>
        </div>
        <StatePill state={line.state} />
      </div>
      <p className="detail">{describe(i, line)}</p>
      <div className="meta">
        <GroupTag group={group} />
        {line.webhook ? <span className="tag">{t('card.webhookOn')}</span> : <span className="tag warn">{t('card.webhookOff')}</span>}
      </div>
      <div className="actions">
        {primary}
        {secondary}
        <Link className="btn btn-soft btn-icon" to={`/lineas/${line.id}`} aria-label={t('card.details', { name: line.name })}>
          <Icon name="ir" size={18} />
        </Link>
      </div>
    </article>
  );
}

export function Loading({ label }: { label?: string }) {
  const { t } = useI18n();
  return (
    <div className="empty" role="status">
      <span className="spinner" />
      <span>{label ?? t('common.loading')}</span>
    </div>
  );
}

export function ErrorBox({ error }: { error: unknown }) {
  const { t } = useI18n();
  return <div className="alert error">{error instanceof Error ? error.message : t('common.error')}</div>;
}

export function NotFound() {
  const { t } = useI18n();
  return (
    <div className="empty">
      <b>{t('common.notFound')}</b>
      <Link className="btn btn-dark" to="/lineas">
        {t('common.backToLines')}
      </Link>
    </div>
  );
}
