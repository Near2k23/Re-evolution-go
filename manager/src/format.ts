import type { Line, LineState } from './api';

export const STATE_META: Record<LineState, { title: string; label: string; desc: string }> = {
  caida: { title: 'Caídas', label: 'Caída', desc: 'Necesitan reconectarse o vincularse de nuevo' },
  reconectando: { title: 'Reconectando', label: 'Reconectando', desc: 'Intentando volver ahora mismo' },
  reconectada: { title: 'Reconectadas', label: 'Reconectada', desc: 'Volvieron solas tras un corte en las últimas 24 h' },
  conectada: { title: 'Conectadas', label: 'Conectada', desc: 'Sin cortes en las últimas 24 h' },
};

export const STATE_ORDER: LineState[] = ['caida', 'reconectando', 'reconectada', 'conectada'];

/** Grupo según el prefijo del nombre (Optical-3, Magma-1, Qarzo-2…). */
export function groupOf(name: string): string {
  const prefix = name.split('-')[0].toLowerCase();
  return ['optical', 'magma', 'qarzo'].includes(prefix) ? prefix : 'otras';
}

export function groupLabel(g: string): string {
  return g === 'otras' ? 'Otras' : g[0].toUpperCase() + g.slice(1);
}

export function initials(name: string): string {
  const [a, b] = name.split('-');
  if (b) return (a[0] + b).toUpperCase().slice(0, 3);
  return name.slice(0, 2).toUpperCase();
}

/** +57 300 123 4567 a partir de 573001234567. */
export function formatPhone(digits: string): string {
  if (!digits) return 'Sin número';
  if (digits.startsWith('57') && digits.length === 12) return `+57 ${digits.slice(2, 5)} ${digits.slice(5, 8)} ${digits.slice(8)}`;
  if (digits.startsWith('52') && digits.length >= 12) return `+52 ${digits.slice(2, digits.length - 8)} ${digits.slice(-8, -4)} ${digits.slice(-4)}`;
  return '+' + digits;
}

const rtf = new Intl.RelativeTimeFormat('es', { numeric: 'auto' });

export function ago(iso?: string, now = Date.now()): string {
  if (!iso) return '';
  const s = Math.round((new Date(iso).getTime() - now) / 1000);
  const a = Math.abs(s);
  if (a < 60) return rtf.format(s, 'second');
  if (a < 3600) return rtf.format(Math.round(s / 60), 'minute');
  if (a < 86400) return rtf.format(Math.round(s / 3600), 'hour');
  return rtf.format(Math.round(s / 86400), 'day');
}

export function clock(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  const today = new Date();
  const hm = d.toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit' });
  if (d.toDateString() === today.toDateString()) return hm;
  return d.toLocaleDateString('es', { day: 'numeric', month: 'short' }) + ' ' + hm;
}

export function duration(sec: number): string {
  if (sec < 60) return `${sec} s`;
  if (sec < 3600) return `${Math.round(sec / 60)} min`;
  if (sec < 86400) return `${Math.round(sec / 3600)} h`;
  return `${Math.round(sec / 86400)} días`;
}

const REASONS: Record<string, string> = {
  logged_out: 'Sesión cerrada desde el teléfono',
  temporary_ban: 'WhatsApp la limitó temporalmente',
  connect_failure: 'WhatsApp rechazó la conexión',
  stream_replaced: 'Se abrió la sesión en otro lugar',
  disconnected: 'Se perdió la conexión',
};

/** Frase que explica la situación de una línea. */
export function describe(l: Line): string {
  switch (l.state) {
    case 'conectada':
      return l.connectedSince ? `Conectada sin cortes ${ago(l.connectedSince)}.`.replace('sin cortes hace', 'sin cortes desde hace') : 'Conectada sin cortes en las últimas 24 h.';
    case 'reconectada': {
      const cuts = l.cuts24h === 1 ? 'un corte' : `${l.cuts24h} cortes`;
      const back = l.lastReconnectAt ? `Volvió sola a las ${clock(l.lastReconnectAt)}. ` : '';
      return `${back}${cuts[0].toUpperCase() + cuts.slice(1)} en 24 h; el más largo duró ${duration(l.longestCutSec)}.`;
    }
    case 'reconectando':
      return `Intentando volver${l.lastCutAt ? ` desde las ${clock(l.lastCutAt)}` : ''}.`;
    case 'caida': {
      if (!l.jid) return 'Todavía no se ha vinculado ningún teléfono.';
      const why = (l.lastKind && REASONS[l.lastKind]) || (l.disconnectReason ? l.disconnectReason : 'Sin conexión');
      const when = l.lastEventAt ? ` ${ago(l.lastEventAt)}` : '';
      const fix = l.needsPairing ? ' Hay que vincularla otra vez.' : ' La sesión sigue válida: basta con reconectar.';
      return `${why}${when}.${fix}`;
    }
  }
}

export function eventsList(s: string): string[] {
  return s ? s.split(',').map((x) => x.trim()).filter(Boolean) : [];
}
