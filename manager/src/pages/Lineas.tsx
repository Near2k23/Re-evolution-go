import { useMemo, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { api, type Line, type LineState } from '../api';
import { STATE_META, STATE_ORDER, groupLabel, groupOf } from '../format';
import { ErrorBox, Icon, LineCard, Loading, useOverview, useToast } from '../components/ui';

type Filter = 'todas' | LineState;

const TILE_DESC: Record<Filter, string> = {
  todas: 'En el grupo y búsqueda actuales',
  caida: STATE_META.caida.desc,
  reconectando: STATE_META.reconectando.desc,
  reconectada: 'Volvieron solas tras un corte (24 h)',
  conectada: STATE_META.conectada.desc,
};

export default function Lineas() {
  const { data, isLoading, error, dataUpdatedAt } = useOverview();
  const qc = useQueryClient();
  const toast = useToast();
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>('todas');
  const [group, setGroup] = useState('todas');
  const [q, setQ] = useState('');
  const [busy, setBusy] = useState<Record<string, boolean>>({});
  const dialog = useRef<HTMLDialogElement>(null);
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);

  const lines = data ?? [];
  const groups = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const l of lines) counts[groupOf(l.name)] = (counts[groupOf(l.name)] ?? 0) + 1;
    return ['optical', 'magma', 'qarzo', 'otras'].filter((g) => counts[g]).map((g) => ({ id: g, count: counts[g] }));
  }, [lines]);

  const query = q.toLowerCase().replace(/\s/g, '');
  const visible = lines.filter(
    (l) => (group === 'todas' || groupOf(l.name) === group) && (!query || l.name.toLowerCase().includes(query) || l.phone.includes(query.replace(/^\+/, ''))),
  );
  const countOf = (s: LineState) => visible.filter((l) => l.state === s).length;

  const sections = STATE_ORDER.filter((s) => filter === 'todas' || filter === s)
    .map((s) => ({ state: s, items: visible.filter((l) => l.state === s).sort((a, b) => a.name.localeCompare(b.name, 'es', { numeric: true })) }))
    .filter((s) => s.items.length > 0);

  const reconnectable = visible.filter((l) => l.state === 'caida' && !l.needsPairing);

  async function reconnect(list: Line[]) {
    setBusy((b) => ({ ...b, ...Object.fromEntries(list.map((l) => [l.id, true])) }));
    const results = await Promise.allSettled(list.map((l) => api.reconnect(l.id)));
    setBusy((b) => ({ ...b, ...Object.fromEntries(list.map((l) => [l.id, false])) }));
    const ok = results.filter((r) => r.status === 'fulfilled').length;
    toast(list.length === 1 ? (ok ? `${list[0].name}: reconectando…` : `${list[0].name}: no se pudo reconectar`) : `${ok} de ${list.length} líneas reconectando…`);
    qc.invalidateQueries({ queryKey: ['overview'] });
  }

  async function create(e: FormEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const inst = await api.create(newName.trim());
      dialog.current?.close();
      setNewName('');
      await qc.invalidateQueries({ queryKey: ['overview'] });
      navigate(`/lineas/${inst.id}/vincular`);
    } catch (err) {
      toast(err instanceof Error ? err.message : 'No se pudo crear la línea');
    } finally {
      setCreating(false);
    }
  }

  const tiles: { id: Filter; label: string; count: number }[] = [
    { id: 'todas', label: 'Todas', count: visible.length },
    ...STATE_ORDER.map((s) => ({ id: s as Filter, label: STATE_META[s].title, count: countOf(s) })),
  ];

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <h1>Líneas</h1>
          <p className="sub">
            {lines.length} {lines.length === 1 ? 'línea' : 'líneas'} en este servidor
            {dataUpdatedAt ? ` · actualizado ${new Date(dataUpdatedAt).toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}` : ''}
          </p>
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center', flex: '1 1 auto', justifyContent: 'flex-end' }}>
          <label className="search">
            <Icon name="buscar" size={18} />
            <span className="sr-only">Buscar línea</span>
            <input type="search" placeholder="Buscar por nombre o número" value={q} onChange={(e) => setQ(e.target.value)} />
          </label>
          <button type="button" className="btn btn-dark" onClick={() => dialog.current?.showModal()}>
            <Icon name="mas" size={18} />
            Nueva línea
          </button>
        </div>
      </header>

      <section className="tiles" aria-label="Filtrar por estado">
        {tiles.map((t) => (
          <button key={t.id} type="button" className="tile" aria-pressed={filter === t.id} onClick={() => setFilter(t.id)}>
            <span className="t">
              {t.id === 'todas' ? <span className="dot" style={{ background: filter === 'todas' ? '#fff' : 'var(--ink)' }} /> : <span className={`dot s-${t.id}`} style={{ background: 'var(--c)' }} />}
              {t.label}
            </span>
            <span className="v">{t.count}</span>
            <span className="d">{TILE_DESC[t.id]}</span>
          </button>
        ))}
      </section>

      <div className="chips scroll state-chips" role="group" aria-label="Filtrar por estado">
        {tiles.map((t) => (
          <button key={t.id} type="button" className="chip" aria-pressed={filter === t.id} onClick={() => setFilter(t.id)}>
            {t.id !== 'todas' && <span className={`dot s-${t.id}`} style={{ background: 'var(--c)', width: 8, height: 8 }} />}
            {t.label}
            <span className="count">{t.count}</span>
          </button>
        ))}
      </div>

      {groups.length > 1 && (
        <div className="chips scroll" role="group" aria-label="Filtrar por grupo">
          <span className="muted" style={{ fontSize: 13, marginRight: 4 }}>Grupo</span>
          <button type="button" className="chip" aria-pressed={group === 'todas'} onClick={() => setGroup('todas')}>
            Todos <span className="count">{lines.length}</span>
          </button>
          {groups.map((g) => (
            <button key={g.id} type="button" className="chip" aria-pressed={group === g.id} onClick={() => setGroup(g.id)}>
              {groupLabel(g.id)} <span className="count">{g.count}</span>
            </button>
          ))}
        </div>
      )}

      {isLoading && <Loading label="Cargando líneas…" />}
      {error && <ErrorBox error={error} />}
      {!isLoading && !error && sections.length === 0 && (
        <div className="empty">
          <b>{lines.length === 0 ? 'Todavía no hay líneas' : 'Ninguna línea coincide'}</b>
          <span>{lines.length === 0 ? 'Crea una con «Nueva línea» y vincúlala con el teléfono.' : 'Prueba con otro estado, grupo o búsqueda.'}</span>
        </div>
      )}

      {sections.map((sec) => (
        <section key={sec.state} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div className="section-head">
            <h2>
              <span className={`dot s-${sec.state}`} style={{ background: 'var(--c)' }} />
              {STATE_META[sec.state].title}
              <span className="n">{sec.items.length}</span>
            </h2>
            <span className="desc">{STATE_META[sec.state].desc}</span>
            {sec.state === 'caida' && reconnectable.length > 0 && (
              <button type="button" className="btn btn-light btn-sm end" onClick={() => reconnect(reconnectable)}>
                Reconectar las que tienen sesión ({reconnectable.length})
              </button>
            )}
          </div>
          <div className="grid-cards">
            {sec.items.map((l) => (
              <LineCard key={l.id} line={l} busy={!!busy[l.id]} onReconnect={(x) => reconnect([x])} />
            ))}
          </div>
        </section>
      ))}

      <dialog ref={dialog} className="sheet" aria-labelledby="nueva-titulo">
        <form onSubmit={create}>
          <h2 id="nueva-titulo">Nueva línea</h2>
          <label className="field">
            Nombre
            <input className="input on-surface" required autoFocus value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="Optical-9" />
          </label>
          <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>
            Empieza el nombre con Optical-, Magma- o Qarzo- para que el panel sepa en qué grupo trabaja. Después la vincularás con el teléfono.
          </p>
          <div className="acts">
            <button type="button" className="btn btn-soft" onClick={() => dialog.current?.close()}>
              Cancelar
            </button>
            <button type="submit" className="btn btn-dark" disabled={creating}>
              {creating ? <span className="spinner" /> : 'Crear y vincular'}
            </button>
          </div>
        </form>
      </dialog>
    </>
  );
}
