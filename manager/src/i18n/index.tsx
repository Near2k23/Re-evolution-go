import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import en, { type Dict } from './locales/en';
import es from './locales/es';
import pt from './locales/pt';
import fr from './locales/fr';
import it from './locales/it';
import de from './locales/de';
import ru from './locales/ru';
import zh from './locales/zh';
import ja from './locales/ja';
import ko from './locales/ko';

export type Key = keyof Dict;
export type PluralBase = { [K in Key]: K extends `${infer B}.one` ? B : never }[Key];

/** Idiomas disponibles: código, nombre en su propio idioma y etiqueta BCP 47 para Intl. */
export const LOCALES = [
  { code: 'es', name: 'Español', tag: 'es', dict: es },
  { code: 'en', name: 'English', tag: 'en', dict: en },
  { code: 'pt', name: 'Português', tag: 'pt-BR', dict: pt },
  { code: 'fr', name: 'Français', tag: 'fr', dict: fr },
  { code: 'it', name: 'Italiano', tag: 'it', dict: it },
  { code: 'de', name: 'Deutsch', tag: 'de', dict: de },
  { code: 'ru', name: 'Русский', tag: 'ru', dict: ru },
  { code: 'zh', name: '中文', tag: 'zh-CN', dict: zh },
  { code: 'ja', name: '日本語', tag: 'ja', dict: ja },
  { code: 'ko', name: '한국어', tag: 'ko', dict: ko },
] as const;

export type LocaleCode = (typeof LOCALES)[number]['code'];
const STORAGE = 'enlace.locale';

function detect(): LocaleCode {
  try {
    const saved = localStorage.getItem(STORAGE);
    if (saved && LOCALES.some((l) => l.code === saved)) return saved as LocaleCode;
  } catch {
    /* sin almacenamiento */
  }
  for (const lang of navigator.languages ?? [navigator.language]) {
    const base = lang.toLowerCase().split('-')[0];
    const hit = LOCALES.find((l) => l.code === base);
    if (hit) return hit.code;
  }
  return 'en';
}

export type Vars = Record<string, string | number>;

export interface I18n {
  locale: LocaleCode;
  tag: string;
  setLocale: (c: LocaleCode) => void;
  t: (key: Key, vars?: Vars) => string;
  tp: (base: PluralBase, n: number, vars?: Vars) => string;
  /** «hace 3 horas», «in 2 days»… */
  ago: (iso?: string) => string;
  /** 14:05, o «3 nov 14:05» si no es hoy. */
  clock: (iso?: string) => string;
  /** 41 s, 6 min, 2 h, 3 días. */
  duration: (sec: number) => string;
  num: (n: number) => string;
}

const Ctx = createContext<I18n | null>(null);

function fill(s: string, vars?: Vars) {
  if (!vars) return s;
  return s.replace(/\{(\w+)\}/g, (m, k) => (k in vars ? String(vars[k]) : m));
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<LocaleCode>(detect);
  const entry = LOCALES.find((l) => l.code === locale) ?? LOCALES[1];

  useEffect(() => {
    document.documentElement.lang = entry.tag;
  }, [entry.tag]);

  const setLocale = useCallback((c: LocaleCode) => {
    try {
      localStorage.setItem(STORAGE, c);
    } catch {
      /* sin almacenamiento */
    }
    setLocaleState(c);
  }, []);

  const value = useMemo<I18n>(() => {
    const dict = entry.dict as Dict;
    const rules = new Intl.PluralRules(entry.tag);
    const rtf = new Intl.RelativeTimeFormat(entry.tag, { numeric: 'auto' });
    const nf = new Intl.NumberFormat(entry.tag);
    const t = (key: Key, vars?: Vars) => fill(dict[key] ?? en[key] ?? key, vars);
    const unit = (u: 'second' | 'minute' | 'hour' | 'day', v: number) =>
      new Intl.NumberFormat(entry.tag, { style: 'unit', unit: u, unitDisplay: u === 'day' ? 'long' : 'short' }).format(v);
    return {
      locale,
      tag: entry.tag,
      setLocale,
      t,
      tp: (base, n, vars) => {
        const form = rules.select(n) === 'one' ? 'one' : 'other';
        return t(`${base}.${form}` as Key, { n: nf.format(n), ...vars });
      },
      ago: (iso) => {
        if (!iso) return '';
        const s = Math.round((new Date(iso).getTime() - Date.now()) / 1000);
        const a = Math.abs(s);
        if (a < 60) return rtf.format(s, 'second');
        if (a < 3600) return rtf.format(Math.round(s / 60), 'minute');
        if (a < 86400) return rtf.format(Math.round(s / 3600), 'hour');
        return rtf.format(Math.round(s / 86400), 'day');
      },
      clock: (iso) => {
        if (!iso) return '';
        const d = new Date(iso);
        const hm = d.toLocaleTimeString(entry.tag, { hour: '2-digit', minute: '2-digit' });
        if (d.toDateString() === new Date().toDateString()) return hm;
        return d.toLocaleDateString(entry.tag, { day: 'numeric', month: 'short' }) + ' ' + hm;
      },
      duration: (sec) => {
        if (sec < 60) return unit('second', sec);
        if (sec < 3600) return unit('minute', Math.round(sec / 60));
        if (sec < 86400) return unit('hour', Math.round(sec / 3600));
        return unit('day', Math.round(sec / 86400));
      },
      num: (n) => nf.format(n),
    };
  }, [entry, locale, setLocale]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useI18n(): I18n {
  const c = useContext(Ctx);
  if (!c) throw new Error('useI18n fuera de I18nProvider');
  return c;
}

/** Selector de idioma. */
export function LanguageSelect({ compact = false }: { compact?: boolean }) {
  const { locale, setLocale, t } = useI18n();
  return (
    <label className="lang-select" style={compact ? { padding: 0 } : undefined}>
      <span className="sr-only">{t('common.language')}</span>
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM3.5 9h17M3.5 15h17M12 3c2.5 2.6 3.7 5.6 3.7 9s-1.2 6.4-3.7 9c-2.5-2.6-3.7-5.6-3.7-9S9.5 5.6 12 3z" />
      </svg>
      <select value={locale} onChange={(e) => setLocale(e.target.value as LocaleCode)}>
        {LOCALES.map((l) => (
          <option key={l.code} value={l.code}>
            {l.name}
          </option>
        ))}
      </select>
    </label>
  );
}
