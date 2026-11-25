import { useMemo, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { api, type LineState } from '../api';
import { useI18n } from '../i18n';
import { STATE_ORDER, byName, groupColors, stateDesc, stateTitle } from '../format';
import { ErrorBox, Icon, LineCard, Loading, useGroups, useOverview, useReconnect, useToast } from '../components/ui';

type Filter = 'todas' | LineState;
const NONE = '__none__';

export default function Lineas() {
  const i = useI18n();
  const { t, tp } = i;
  const { data, isLoading, error, dataUpdatedAt } = useOverview();
  const { data: groupData } = useGroups();
  const qc = useQueryClient();
  const toast = useToast();
  const navigate = useNavigate();
  const { run: reconnect, busy } = useReconnect();
  const [filter, setFilter] = useState<Filter>('todas');
  const [group, setGroup] = useState('todas');
  const [q, setQ] = useState('');
  const dialog = useRef<HTMLDialogElement>(null);
  const [newName, setNewName] = useState('');
  const [newGroup, setNewGroup] = useState('');
  const [creating, setCreating] = useState(false);

  const lines = data ?? [];
  const groups = groupData?.groups ?? [];
  const groupById = useMemo(() => Object.fromEntries(groups.map((g) => [g.id, g])), [groups]);
  const ungroupedCount = lines.filter((l) => !l.groupId || !groupById[l.groupId]).length;

  const query = q.toLowerCase().replace(/\s/g, '');
  const visible = lines.filter((l) => {
    const inGroup = group === 'todas' || (group === NONE ? !l.groupId || !groupById[l.groupId] : l.groupId === group);
    const match = !query || l.name.toLowerCase().replace(/\s/g, '').includes(query) || l.phone.includes(query.replace(/^\+/, ''));
    return inGroup && match;
  });
  const countOf = (s: LineState) => visible.filter((l) => l.state === s).length;

  const sections = STATE_ORDER.filter((s) => filter === 'todas' || filter === s)
    .map((s) => ({ state: s, items: visible.filter((l) => l.state === s).sort(byName) }))
    .filter((s) => s.items.length > 0);

  const reconnectable = visible.filter((l) => l.state === 'caida' && !l.needsPairing);

  async function create(e: FormEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const inst = await api.create(newName.trim());
      if (newGroup) await api.assignGroup([inst.id], newGroup);
      dialog.current?.close();
      setNewName('');
      await qc.invalidateQueries({ queryKey: ['overview'] });
      navigate(`/lineas/${inst.id}/vincular`);
    } catch (err) {
      toast(err instanceof Error ? err.message : t('lines.createFailed'));
    } finally {
      setCreating(false);
    }
  }

  const tiles: { id: Filter; label: string; desc: string; count: number }[] = [
    { id: 'todas', label: t('state.all'), desc: t('state.all.desc'), count: visible.length },
    ...STATE_ORDER.map((s) => ({ id: s as Filter, label: stateTitle(i, s), desc: stateDesc(i, s), count: countOf(s) })),
  ];

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <h1>{t('lines.title')}</h1>
          <p className="sub">
            {tp('lines.count', lines.length)}
            {dataUpdatedAt ? ` · ${t('lines.updated', { time: new Date(dataUpdatedAt).toLocaleTimeString(i.tag) })}` : ''}
          </p>
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center', flex: '1 1 auto', justifyContent: 'flex-end' }}>
          <label className="search">
            <Icon name="buscar" size={18} />
            <span className="sr-only">{t('lines.searchLabel')}</span>
            <input type="search" placeholder={t('lines.search')} value={q} onChange={(e) => setQ(e.target.value)} />
          </label>
          <button type="button" className="btn btn-dark" onClick={() => dialog.current?.showModal()} aria-label={t('lines.new')}>
            <Icon name="mas" size={18} />
            <span className="hide-sm">{t('lines.new')}</span>
          </button>
        </div>
      </header>

      <section className="tiles" aria-label={t('state.filter')}>
        {tiles.map((x) => (
          <button key={x.id} type="button" className="tile" aria-pressed={filter === x.id} onClick={() => setFilter(x.id)}>
            <span className="t">
              {x.id === 'todas' ? <span className="dot" style={{ background: filter === 'todas' ? '#fff' : 'var(--ink)' }} /> : <span className={`dot s-${x.id}`} style={{ background: 'var(--c)' }} />}
              {x.label}
            </span>
            <span className="v">{i.num(x.count)}</span>
            <span className="d">{x.desc}</span>
          </button>
        ))}
      </section>

      <div className="chips scroll state-chips" role="group" aria-label={t('state.filter')}>
        {tiles.map((x) => (
          <button key={x.id} type="button" className="chip" aria-pressed={filter === x.id} onClick={() => setFilter(x.id)}>
            {x.id !== 'todas' && <span className={`dot s-${x.id}`} style={{ background: 'var(--c)', width: 8, height: 8 }} />}
            {x.label}
            <span className="count">{i.num(x.count)}</span>
          </button>
        ))}
      </div>

      {groups.length > 0 && (
        <div className="chips scroll" role="group" aria-label={t('lines.groupFilter')}>
          <span className="muted" style={{ fontSize: 13, marginRight: 4 }}>{t('lines.group')}</span>
          <button type="button" className="chip" aria-pressed={group === 'todas'} onClick={() => setGroup('todas')}>
            {t('lines.allGroups')} <span className="count">{i.num(lines.length)}</span>
          </button>
          {groups.map((g) => (
            <button key={g.id} type="button" className="chip" aria-pressed={group === g.id} onClick={() => setGroup(g.id)}>
              <span className="dot" style={{ background: groupColors(g).dot, width: 8, height: 8 }} />
              {g.name} <span className="count">{i.num(lines.filter((l) => l.groupId === g.id).length)}</span>
            </button>
          ))}
          {ungroupedCount > 0 && (
            <button type="button" className="chip" aria-pressed={group === NONE} onClick={() => setGroup(NONE)}>
              {t('lines.noGroup')} <span className="count">{i.num(ungroupedCount)}</span>
            </button>
          )}
        </div>
      )}

      {isLoading && <Loading />}
      {error && <ErrorBox error={error} />}
      {!isLoading && !error && sections.length === 0 && (
        <div className="empty">
          <b>{lines.length === 0 ? t('lines.emptyTitle') : t('lines.noMatchTitle')}</b>
          <span>{lines.length === 0 ? t('lines.emptyHint') : t('lines.noMatchHint')}</span>
        </div>
      )}

      {sections.map((sec) => (
        <section key={sec.state} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div className="section-head">
            <h2>
              <span className={`dot s-${sec.state}`} style={{ background: 'var(--c)' }} />
              {stateTitle(i, sec.state)}
              <span className="n">{i.num(sec.items.length)}</span>
            </h2>
            <span className="desc">{stateDesc(i, sec.state)}</span>
            {sec.state === 'caida' && reconnectable.length > 0 && (
              <button type="button" className="btn btn-light btn-sm end" onClick={() => reconnect(reconnectable)}>
                <Icon name="recargar" size={16} />
                {t('lines.reconnectAll', { n: i.num(reconnectable.length) })}
              </button>
            )}
          </div>
          <div className="grid-cards">
            {sec.items.map((l) => (
              <LineCard key={l.id} line={l} group={l.groupId ? groupById[l.groupId] : null} busy={!!busy[l.id]} onReconnect={(x) => reconnect([x])} />
            ))}
          </div>
        </section>
      ))}

      <dialog ref={dialog} className="sheet" aria-labelledby="nueva-titulo">
        <form onSubmit={create}>
          <h2 id="nueva-titulo">{t('lines.createTitle')}</h2>
          <label className="field">
            {t('lines.name')}
            <input className="input on-surface" required autoFocus value={newName} onChange={(e) => setNewName(e.target.value)} placeholder={t('lines.namePlaceholder')} />
          </label>
          {groups.length > 0 && (
            <label className="field">
              {t('lines.group')}
              <select className="input on-surface" value={newGroup} onChange={(e) => setNewGroup(e.target.value)}>
                <option value="">{t('lines.noGroup')}</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          <p className="muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>{t('lines.createHint')}</p>
          <div className="acts">
            <button type="button" className="btn btn-soft" onClick={() => dialog.current?.close()}>
              {t('common.cancel')}
            </button>
            <button type="submit" className="btn btn-dark" disabled={creating}>
              {creating ? <span className="spinner" /> : t('lines.createAndLink')}
            </button>
          </div>
        </form>
      </dialog>
    </>
  );
}
