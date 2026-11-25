import { Link } from 'react-router-dom';
import type { Line } from '../api';
import { useI18n } from '../i18n';
import { byName, describe, stateDesc } from '../format';
import { ErrorBox, Loading, useOverview } from '../components/ui';

function uptime(hours: Line['hours'], fmt: (n: number) => string): string {
  if (!hours.length) return '—';
  const score = hours.reduce((acc, h) => acc + (h === 'ok' ? 1 : h === 'corte' ? 0.5 : 0), 0);
  return fmt(Math.round((score / hours.length) * 1000) / 10) + ' %';
}

export default function Panel() {
  const i = useI18n();
  const { t, tp } = i;
  const { data, isLoading, error } = useOverview();
  if (isLoading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const lines = data ?? [];

  const by = (s: Line['state']) => lines.filter((l) => l.state === s);
  const up = by('conectada').length + by('reconectada').length;
  const cuts = lines.reduce((a, l) => a + l.cuts24h, 0);
  const recovered = by('reconectada').length;
  const noHook = lines.filter((l) => !l.webhook);
  const active = lines.filter((l) => l.state !== 'caida' || l.hours.some((h) => h !== 'caida')).sort(byName);
  const attention = [...by('reconectando'), ...by('caida').filter((l) => !l.needsPairing), ...by('caida').filter((l) => l.needsPairing && l.jid)].slice(0, 5);
  const now = new Date();
  const hour = now.getHours();
  const greeting = hour < 12 ? t('panel.morning') : hour < 19 ? t('panel.afternoon') : t('panel.evening');
  const labels = [0, 6, 12, 18, 23].map((k) => new Date(now.getTime() - (23 - k) * 3600e3).toLocaleTimeString(i.tag, { hour: '2-digit', minute: '2-digit' }));
  const recent = lines
    .filter((l) => l.lastEventAt)
    .sort((a, b) => (b.lastEventAt ?? '').localeCompare(a.lastEventAt ?? ''))
    .slice(0, 6);
  const total = Math.max(lines.length, 1);
  const pct = (n: number) => new Intl.NumberFormat(i.tag, { maximumFractionDigits: 1 }).format(n);

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <h1>{greeting}</h1>
          <p className="sub">{now.toLocaleDateString(i.tag, { weekday: 'long', day: 'numeric', month: 'long' })}</p>
        </div>
      </header>

      <section className="grid-auto" aria-label={t('nav.panel')}>
        <div className="stat dark">
          <span className="label">{t('panel.operational')}</span>
          <span className="value">
            {i.num(up)} <small>{t('panel.of', { n: i.num(lines.length) })}</small>
          </span>
          <div style={{ display: 'flex', height: 8, borderRadius: 4, overflow: 'hidden', gap: 2 }} aria-hidden="true">
            <span style={{ flex: by('conectada').length / total, background: 'var(--ok)' }} />
            <span style={{ flex: by('reconectada').length / total, background: 'var(--back-soft)' }} />
            <span style={{ flex: by('reconectando').length / total, background: 'var(--retry)' }} />
            <span style={{ flex: by('caida').length / total, background: '#4A3530' }} />
          </div>
          <span className="hint">
            {t('panel.breakdown', { c: i.num(by('conectada').length), r: i.num(recovered), g: i.num(by('reconectando').length), d: i.num(by('caida').length) })}
          </span>
        </div>
        <div className="stat">
          <span className="label">{t('panel.cuts')}</span>
          <span className="value">{i.num(cuts)}</span>
          <span className="hint">{cuts === 0 ? t('panel.noCuts') : tp('panel.recovered', recovered)}</span>
        </div>
        <div className={noHook.length ? 'stat warn' : 'stat'}>
          <span className="label">{t('panel.noHook')}</span>
          <span className="value">{i.num(noHook.length)}</span>
          <span className="hint">{noHook.length ? t('panel.noHookHint') : t('panel.allHook')}</span>
        </div>
      </section>

      <div className="split">
        <section className="card main">
          <div className="section-head">
            <h2>{t('panel.availability')}</h2>
            <span className="desc">{t('panel.activeLines')}</span>
          </div>
          <div className="legend">
            <span><i className="h-ok" />{t('panel.legendOk')}</span>
            <span><i className="h-corte" />{t('panel.legendCut')}</span>
            <span><i className="h-caida" />{t('panel.legendDown')}</span>
          </div>
          {active.length === 0 ? (
            <p className="muted" style={{ margin: 0 }}>{t('panel.noneActive')}</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {active.map((l) => (
                <div className="hour-row" key={l.id}>
                  <Link className="n" to={`/lineas/${l.id}`} style={{ textDecoration: 'none' }}>
                    {l.name}
                  </Link>
                  <div className="hours" role="img" aria-label={t('panel.hoursAria', { name: l.name, n: i.num(l.cuts24h) })}>
                    {l.hours.map((h, k) => (
                      <span key={k} className={`h-${h}`} />
                    ))}
                  </div>
                  <span className="p">{uptime(l.hours, pct)}</span>
                </div>
              ))}
              <div className="hour-row" style={{ fontSize: 12, color: 'var(--ink-4)' }}>
                <span className="n" />
                <div style={{ flexGrow: 1, display: 'flex', justifyContent: 'space-between' }}>
                  {labels.map((x, k) => (
                    <span key={k}>{x}</span>
                  ))}
                </div>
                <span className="p" />
              </div>
            </div>
          )}
        </section>

        <div className="side">
          <section className="card">
            <div className="section-head">
              <h2>{t('panel.attention')}</h2>
              <Link className="end" to="/lineas" style={{ fontSize: 14, fontWeight: 600, textDecoration: 'none' }}>
                {t('panel.seeLines')}
              </Link>
            </div>
            {attention.length === 0 && <p className="muted" style={{ margin: 0 }}>{t('panel.nothing')}</p>}
            {attention.map((l) => {
              const pair = l.needsPairing && l.state === 'caida';
              return (
                <div className="attention" key={l.id}>
                  <span className={`dot s-${l.state}`} style={{ background: 'var(--c)' }} />
                  <div className="who">
                    <b>{l.name}</b>
                    <span>{l.state === 'reconectando' ? stateDesc(i, 'reconectando') : describe(i, l)}</span>
                  </div>
                  <Link className="btn btn-dark btn-sm" to={pair ? `/lineas/${l.id}/vincular` : `/lineas/${l.id}`}>
                    {pair ? t('card.link') : t('common.view')}
                  </Link>
                </div>
              );
            })}
          </section>

          <section className="card">
            <div className="section-head">
              <h2>{t('panel.recent')}</h2>
              <Link className="end" to="/eventos" style={{ fontSize: 14, fontWeight: 600, textDecoration: 'none' }}>
                {t('panel.monitor')}
              </Link>
            </div>
            <div className="rows">
              {recent.map((l) => (
                <div className="row" key={l.id} style={{ alignItems: 'flex-start', flexWrap: 'nowrap' }}>
                  <span className="mono muted" style={{ width: 96, flexShrink: 0, fontSize: 12.5 }}>
                    {i.clock(l.lastEventAt)}
                  </span>
                  <span style={{ fontSize: 14, lineHeight: 1.4 }}>
                    <b style={{ fontWeight: 600 }}>{l.name}</b>{' '}
                    {l.lastKind === 'connected' || l.lastKind === 'pair_success'
                      ? l.cuts24h
                        ? t('panel.act.back', { d: i.duration(l.longestCutSec) })
                        : t('panel.act.connected')
                      : l.lastKind === 'logged_out'
                        ? t('panel.act.loggedOut')
                        : t('panel.act.lost')}
                  </span>
                </div>
              ))}
              {recent.length === 0 && <p className="muted" style={{ margin: 0 }}>{t('panel.noActivity')}</p>}
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
