import { Link } from 'react-router-dom';
import type { Line } from '../api';
import { STATE_META, clock, describe, duration } from '../format';
import { ErrorBox, Loading, useOverview } from '../components/ui';

function greeting(d = new Date()): string {
  const h = d.getHours();
  if (h < 12) return 'Buenos días';
  if (h < 19) return 'Buenas tardes';
  return 'Buenas noches';
}

function uptime(hours: Line['hours']): string {
  if (!hours.length) return '—';
  const score = hours.reduce((acc, h) => acc + (h === 'ok' ? 1 : h === 'corte' ? 0.5 : 0), 0);
  return `${Math.round((score / hours.length) * 1000) / 10} %`.replace('.', ',');
}

export default function Panel() {
  const { data, isLoading, error } = useOverview();
  if (isLoading) return <Loading />;
  if (error) return <ErrorBox error={error} />;
  const lines = data ?? [];

  const by = (s: Line['state']) => lines.filter((l) => l.state === s);
  const up = by('conectada').length + by('reconectada').length;
  const cuts = lines.reduce((a, l) => a + l.cuts24h, 0);
  const recovered = by('reconectada').length;
  const noHook = lines.filter((l) => !l.webhook);
  const active = lines.filter((l) => l.state !== 'caida' || l.hours.some((h) => h !== 'caida')).sort((a, b) => a.name.localeCompare(b.name, 'es', { numeric: true }));
  const attention = [...by('reconectando'), ...by('caida').filter((l) => !l.needsPairing), ...by('caida').filter((l) => l.needsPairing && l.jid)].slice(0, 5);
  const now = new Date();
  const labels = [0, 6, 12, 18, 23].map((i) => {
    const d = new Date(now.getTime() - (23 - i) * 3600 * 1000);
    return d.toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit' });
  });

  const recent = lines
    .filter((l) => l.lastEventAt)
    .sort((a, b) => (b.lastEventAt ?? '').localeCompare(a.lastEventAt ?? ''))
    .slice(0, 6);

  const total = Math.max(lines.length, 1);

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <h1>{greeting()}</h1>
          <p className="sub">{now.toLocaleDateString('es', { weekday: 'long', day: 'numeric', month: 'long' })}</p>
        </div>
      </header>

      <section className="grid-auto" aria-label="Resumen">
        <div className="stat dark">
          <span className="label">Líneas operativas</span>
          <span className="value">
            {up} <small>de {lines.length}</small>
          </span>
          <div style={{ display: 'flex', height: 8, borderRadius: 4, overflow: 'hidden', gap: 2 }} aria-hidden="true">
            <span style={{ flex: by('conectada').length / total, background: 'var(--ok)' }} />
            <span style={{ flex: by('reconectada').length / total, background: 'var(--back-soft)' }} />
            <span style={{ flex: by('reconectando').length / total, background: 'var(--retry)' }} />
            <span style={{ flex: by('caida').length / total, background: '#4A3530' }} />
          </div>
          <span className="hint">
            {by('conectada').length} conectadas · {recovered} reconectadas · {by('reconectando').length} reconectando · {by('caida').length} caídas
          </span>
        </div>
        <div className="stat">
          <span className="label">Cortes en 24 h</span>
          <span className="value">{cuts}</span>
          <span className="hint">{cuts === 0 ? 'Ninguna línea se cortó.' : `${recovered} ${recovered === 1 ? 'línea volvió sola' : 'líneas volvieron solas'}.`}</span>
        </div>
        <div className={noHook.length ? 'stat warn' : 'stat'}>
          <span className="label">Sin webhook</span>
          <span className="value">{noHook.length}</span>
          <span className="hint">{noHook.length ? 'Sus mensajes no llegan a tu panel.' : 'Todas avisan a tu panel.'}</span>
        </div>
      </section>

      <div className="split">
        <section className="card main">
          <div className="section-head">
            <h2>Disponibilidad por hora</h2>
            <span className="desc">Líneas activas en las últimas 24 h</span>
          </div>
          <div className="legend">
            <span><i className="h-ok" />Conectada</span>
            <span><i className="h-corte" />Corte, volvió</span>
            <span><i className="h-caida" />Caída</span>
          </div>
          {active.length === 0 ? (
            <p className="muted" style={{ margin: 0 }}>Ninguna línea estuvo activa en las últimas 24 h.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {active.map((l) => (
                <div className="hour-row" key={l.id}>
                  <Link className="n" to={`/lineas/${l.id}`} style={{ textDecoration: 'none' }}>
                    {l.name}
                  </Link>
                  <div className="hours" role="img" aria-label={`${l.name}: ${l.cuts24h} cortes en 24 h`}>
                    {l.hours.map((h, i) => (
                      <span key={i} className={`h-${h}`} />
                    ))}
                  </div>
                  <span className="p">{uptime(l.hours)}</span>
                </div>
              ))}
              <div className="hour-row" style={{ fontSize: 12, color: 'var(--ink-4)' }}>
                <span className="n" />
                <div style={{ flexGrow: 1, display: 'flex', justifyContent: 'space-between' }}>
                  {labels.map((t, i) => (
                    <span key={i}>{t}</span>
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
              <h2>Necesitan atención</h2>
              <Link className="end" to="/lineas" style={{ fontSize: 14, fontWeight: 600, textDecoration: 'none' }}>
                Ver líneas
              </Link>
            </div>
            {attention.length === 0 && <p className="muted" style={{ margin: 0 }}>Nada pendiente.</p>}
            {attention.map((l) => (
              <div className="attention" key={l.id}>
                <span className={`dot s-${l.state}`} style={{ background: 'var(--c)' }} />
                <div className="who">
                  <b>{l.name}</b>
                  <span>{l.state === 'reconectando' ? STATE_META.reconectando.desc : describe(l)}</span>
                </div>
                <Link className="btn btn-dark btn-sm" to={l.needsPairing && l.state === 'caida' ? `/lineas/${l.id}/vincular` : `/lineas/${l.id}`}>
                  {l.needsPairing && l.state === 'caida' ? 'Vincular' : 'Ver'}
                </Link>
              </div>
            ))}
          </section>

          <section className="card">
            <div className="section-head">
              <h2>Actividad reciente</h2>
              <Link className="end" to="/eventos" style={{ fontSize: 14, fontWeight: 600, textDecoration: 'none' }}>
                Monitor
              </Link>
            </div>
            <div className="rows">
              {recent.map((l) => (
                <div className="row" key={l.id} style={{ alignItems: 'flex-start', flexWrap: 'nowrap' }}>
                  <span className="mono muted" style={{ width: 92, flexShrink: 0, fontSize: 12.5 }}>
                    {clock(l.lastEventAt)}
                  </span>
                  <span style={{ fontSize: 14, lineHeight: 1.4 }}>
                    <b style={{ fontWeight: 600 }}>{l.name}</b>{' '}
                    {l.lastKind === 'connected' || l.lastKind === 'pair_success'
                      ? l.cuts24h
                        ? `volvió sola (corte más largo ${duration(l.longestCutSec)}).`
                        : 'se conectó.'
                      : l.lastKind === 'logged_out'
                        ? 'cerró sesión.'
                        : 'perdió la conexión.'}
                  </span>
                </div>
              ))}
              {recent.length === 0 && <p className="muted" style={{ margin: 0 }}>Sin actividad registrada todavía.</p>}
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
