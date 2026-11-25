import type { Group, Line, LineState } from './api';
import type { I18n, Key } from './i18n';

export const STATE_ORDER: LineState[] = ['caida', 'reconectando', 'reconectada', 'conectada'];

export function stateTitle(i: I18n, s: LineState) {
  return i.t(`state.${s}.title` as Key);
}
export function stateLabel(i: I18n, s: LineState) {
  return i.t(`state.${s}.label` as Key);
}
export function stateDesc(i: I18n, s: LineState) {
  return i.t(`state.${s}.desc` as Key);
}

/** Dos o tres letras para el avatar: «Ventas-3» → «V3», «Soporte» → «SO». */
export function initials(name: string): string {
  const parts = name.split(/[-_\s]+/).filter(Boolean);
  if (parts.length >= 2) {
    const [a, b] = parts;
    return (a[0] + (/^\d+$/.test(b) ? b : b[0])).toUpperCase().slice(0, 3);
  }
  return name.slice(0, 2).toUpperCase();
}

/** Número en formato internacional sin asumir el país: se agrupa de 3 en 3 desde el final (últimos 4 juntos). */
export function formatPhone(digits: string): string {
  if (!digits) return '';
  const d = digits.replace(/\D/g, '');
  const groups: string[] = [];
  let rest = d;
  // Últimos 4, y luego grupos de 3 hacia la izquierda.
  groups.unshift(rest.slice(-4));
  rest = rest.slice(0, -4);
  while (rest.length > 3) {
    groups.unshift(rest.slice(-3));
    rest = rest.slice(0, -3);
  }
  if (rest) groups.unshift(rest);
  return '+' + groups.join(' ');
}

/** Frase que explica la situación de una línea, en el idioma activo. */
export function describe(i: I18n, l: Line): string {
  switch (l.state) {
    case 'conectada':
      return l.connectedSince ? i.t('describe.connectedSince', { ago: i.ago(l.connectedSince) }) : i.t('describe.connected');
    case 'reconectada': {
      const back = l.lastReconnectAt ? i.t('describe.backAt', { time: i.clock(l.lastReconnectAt) }) + ' ' : '';
      return back + i.tp('describe.cuts', l.cuts24h, { d: i.duration(l.longestCutSec) });
    }
    case 'reconectando':
      return l.lastCutAt ? i.t('describe.reconnectingSince', { time: i.clock(l.lastCutAt) }) : i.t('describe.reconnecting');
    case 'caida': {
      if (!l.jid) return i.t('describe.neverLinked');
      const kinds = ['logged_out', 'temporary_ban', 'connect_failure', 'stream_replaced', 'disconnected'] as const;
      const k = kinds.find((x) => x === l.lastKind);
      const why = k ? i.t(`reason.${k}` as Key) : i.t('reason.none');
      const when = l.lastEventAt ? ` (${i.ago(l.lastEventAt)})` : '';
      return `${why}${when}. ${l.needsPairing ? i.t('describe.needsPairing') : i.t('describe.sessionValid')}`;
    }
  }
}

export function eventsList(s: string): string[] {
  return s ? s.split(',').map((x) => x.trim()).filter(Boolean) : [];
}

/** Colores de grupo: fondo y texto con contraste suficiente. */
export const GROUP_COLORS: Record<string, { bg: string; fg: string; dot: string }> = {
  green: { bg: '#E3EBE1', fg: '#24452A', dot: '#4C8A55' },
  blue: { bg: '#E2E7F4', fg: '#26386A', dot: '#4767B8' },
  orange: { bg: '#F7E6D6', fg: '#6E3A0C', dot: '#C87533' },
  violet: { bg: '#ECE8F2', fg: '#4A3D6B', dot: '#7B68A8' },
  clay: { bg: '#F2E4DA', fg: '#6B3418', dot: '#B0643A' },
  slate: { bg: '#E6E8EB', fg: '#2F3640', dot: '#69727F' },
  teal: { bg: '#DDEFEC', fg: '#164943', dot: '#2E8B80' },
  rose: { bg: '#F5E1E6', fg: '#6E2238', dot: '#B9506D' },
};
const NO_GROUP = { bg: '#ECEEEA', fg: '#2F3640', dot: '#8A9099' };

export function groupColors(group?: Group | null) {
  return (group && GROUP_COLORS[group.color]) || NO_GROUP;
}

/** Ordena por nombre con números naturales: Línea-2 antes que Línea-10. */
export function byName(a: { name: string }, b: { name: string }) {
  return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' });
}

/** Prefijos compartidos por 2 o más líneas sin grupo («Ventas-1», «Ventas-2» → «Ventas»). */
export function suggestPrefixes(lines: Line[]): { prefix: string; lines: Line[] }[] {
  const map = new Map<string, Line[]>();
  for (const l of lines) {
    const m = l.name.match(/^(.+?)[-_\s]+[^-_\s]+$/);
    if (!m) continue;
    const p = m[1].trim();
    if (!p) continue;
    map.set(p, [...(map.get(p) ?? []), l]);
  }
  return [...map.entries()]
    .filter(([, ls]) => ls.length >= 2)
    .map(([prefix, ls]) => ({ prefix, lines: ls.sort(byName) }))
    .sort((a, b) => b.lines.length - a.lines.length);
}
