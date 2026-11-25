import { useMemo, useRef, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { api, type Group, type Line } from '../api';
import { useI18n, type Key } from '../i18n';
import { GROUP_COLORS, byName, groupColors, suggestPrefixes } from '../format';
import { Avatar, ErrorBox, Icon, Loading, StatePill, useGroups, useOverview, useToast } from '../components/ui';

const COLOR_KEYS = Object.keys(GROUP_COLORS);

export default function Grupos() {
  const i = useI18n();
  const { t, tp } = i;
  const qc = useQueryClient();
  const toast = useToast();
  const lines = useOverview();
  const groupsQ = useGroups();
  const dialog = useRef<HTMLDialogElement>(null);
  const [editing, setEditing] = useState<Group | null>(null);
  const [name, setName] = useState('');
  const [color, setColor] = useState(COLOR_KEYS[0]);
  const [saving, setSaving] = useState(false);

  const groups = useMemo(() => [...(groupsQ.data?.groups ?? [])].sort(byName), [groupsQ.data]);
  const all = lines.data ?? [];
  const known = new Set(groups.map((g) => g.id));
  const ungrouped = all.filter((l) => !l.groupId || !known.has(l.groupId)).sort(byName);
  const suggestions = suggestPrefixes(ungrouped).filter((s) => !groups.some((g) => g.name.toLowerCase() === s.prefix.toLowerCase()));

  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: ['groups'] }), qc.invalidateQueries({ queryKey: ['overview'] })]);

  function openNew() {
    setEditing(null);
    setName('');
    setColor(COLOR_KEYS[groups.length % COLOR_KEYS.length]);
    dialog.current?.showModal();
  }
  function openEdit(g: Group) {
    setEditing(g);
    setName(g.name);
    setColor(g.color);
    dialog.current?.showModal();
  }

  async function save(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (editing) await api.updateGroup(editing.id, name.trim(), color);
      else await api.createGroup(name.trim(), color);
      dialog.current?.close();
      await refresh();
      toast(t('groups.saved'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.saveFailed'));
    } finally {
      setSaving(false);
    }
  }

  async function remove(g: Group) {
    if (!confirm(t('groups.deleteConfirm', { name: g.name }))) return;
    try {
      await api.deleteGroup(g.id);
      await refresh();
      toast(t('groups.deleted'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.error'));
    }
  }

  async function move(l: Line, groupId: string) {
    try {
      await api.assignGroup([l.id], groupId);
      await refresh();
      toast(t('groups.moved'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.saveFailed'));
    }
  }

  async function createFromPrefix(prefix: string, members: Line[]) {
    try {
      const g = await api.createGroup(prefix, COLOR_KEYS[groups.length % COLOR_KEYS.length]);
      await api.assignGroup(members.map((l) => l.id), g.id);
      await refresh();
      toast(t('groups.saved'));
    } catch (err) {
      toast(err instanceof Error ? err.message : t('common.saveFailed'));
    }
  }

  if (lines.isLoading || groupsQ.isLoading) return <Loading />;
  if (groupsQ.error) return <ErrorBox error={groupsQ.error} />;

  const moveSelect = (l: Line) => (
    <select className="input on-surface" style={{ height: 40, width: 'auto', minWidth: 150, fontSize: 13 }} value={l.groupId && known.has(l.groupId) ? l.groupId : ''} aria-label={t('groups.moveTo')} onChange={(e) => move(l, e.target.value)}>
      <option value="">{t('lines.noGroup')}</option>
      {groups.map((g) => (
        <option key={g.id} value={g.id}>
          {g.name}
        </option>
      ))}
    </select>
  );

  return (
    <>
      <header className="page-head">
        <div className="titles">
          <h1>{t('groups.title')}</h1>
          <p className="sub">{t('groups.sub')}</p>
        </div>
        <button type="button" className="btn btn-dark" onClick={openNew} aria-label={t('groups.new')}>
          <Icon name="mas" size={18} />
          <span className="hide-sm">{t('groups.new')}</span>
        </button>
      </header>

      {suggestions.length > 0 && (
        <section className="card">
          <div className="section-head">
            <h2>{t('groups.suggestTitle')}</h2>
            <span className="desc">{t('groups.suggestHint')}</span>
          </div>
          <div className="chips">
            {suggestions.slice(0, 8).map((s) => (
              <div key={s.prefix} className="attention" style={{ flex: '1 1 260px' }}>
                <div className="who">
                  <b>{s.prefix}</b>
                  <span>{s.lines.map((l) => l.name).join(', ')}</span>
                </div>
                <button type="button" className="btn btn-dark btn-sm" onClick={() => createFromPrefix(s.prefix, s.lines)}>
                  {t('groups.createAssign')}
                </button>
              </div>
            ))}
          </div>
        </section>
      )}

      {groups.length === 0 && (
        <div className="empty">
          <b>{t('groups.emptyTitle')}</b>
          <span>{t('groups.emptyHint')}</span>
        </div>
      )}

      <div className="grid-cards" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))' }}>
        {groups.map((g) => {
          const c = groupColors(g);
          const members = all.filter((l) => l.groupId === g.id).sort(byName);
          return (
            <section key={g.id} className="card" style={{ gap: 12 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <span aria-hidden="true" style={{ width: 14, height: 14, borderRadius: 7, background: c.dot, flexShrink: 0 }} />
                <div style={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <h2 style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{g.name}</h2>
                  <span className="muted" style={{ fontSize: 13 }}>{tp('groups.lines', members.length)}</span>
                </div>
                <button type="button" className="btn btn-soft btn-icon" aria-label={t('groups.edit')} onClick={() => openEdit(g)}>
                  <Icon name="lapiz" size={18} />
                </button>
                <button type="button" className="btn btn-soft btn-icon" aria-label={t('common.delete')} onClick={() => remove(g)}>
                  <Icon name="papelera" size={18} />
                </button>
              </div>
              {members.length === 0 ? (
                <p className="muted" style={{ margin: 0, fontSize: 14 }}>{t('groups.noMembers')}</p>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {members.map((l) => (
                    <MemberRow key={l.id} line={l} group={g} action={moveSelect(l)} />
                  ))}
                </div>
              )}
            </section>
          );
        })}
      </div>

      {all.length > 0 && (
        <section className="card">
          <div className="section-head">
            <h2>{t('groups.ungrouped')}</h2>
            <span className="n" style={{ color: 'var(--ink-3)' }}>{i.num(ungrouped.length)}</span>
          </div>
          {ungrouped.length === 0 ? (
            <p className="muted" style={{ margin: 0 }}>{t('groups.allGrouped')}</p>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 6 }}>
              {ungrouped.map((l) => (
                <MemberRow key={l.id} line={l} group={null} action={groups.length ? moveSelect(l) : null} />
              ))}
            </div>
          )}
        </section>
      )}

      <dialog ref={dialog} className="sheet" aria-labelledby="grupo-titulo">
        <form onSubmit={save}>
          <h2 id="grupo-titulo">{editing ? t('groups.edit') : t('groups.new')}</h2>
          <label className="field">
            {t('groups.name')}
            <input className="input on-surface" required maxLength={80} autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder={t('groups.namePlaceholder')} />
          </label>
          <fieldset style={{ border: 0, padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 10 }}>
            <legend style={{ fontSize: 14, fontWeight: 600, padding: 0, marginBottom: 10 }}>{t('groups.color')}</legend>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {COLOR_KEYS.map((k) => (
                <label key={k} title={t(`color.${k}` as Key)} style={{ width: 44, height: 44, borderRadius: 22, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', background: color === k ? 'var(--ground-2)' : 'transparent', cursor: 'pointer' }}>
                  <input type="radio" name="color" value={k} checked={color === k} onChange={() => setColor(k)} className="sr-only" aria-label={t(`color.${k}` as Key)} />
                  <span aria-hidden="true" style={{ width: 26, height: 26, borderRadius: 13, background: GROUP_COLORS[k].dot, boxShadow: color === k ? '0 0 0 3px #fff, 0 0 0 5px ' + GROUP_COLORS[k].dot : 'none' }} />
                </label>
              ))}
            </div>
          </fieldset>
          <div className="acts">
            <button type="button" className="btn btn-soft" onClick={() => dialog.current?.close()}>
              {t('common.cancel')}
            </button>
            <button type="submit" className="btn btn-dark" disabled={saving}>
              {saving ? <span className="spinner" /> : editing ? t('common.save') : t('groups.create')}
            </button>
          </div>
        </form>
      </dialog>
    </>
  );
}

function MemberRow({ line, group, action }: { line: Line; group: Group | null; action: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: 8, borderRadius: 14, background: 'var(--tint)', flexWrap: 'wrap' }}>
      <Avatar name={line.name} group={group} />
      <Link to={`/lineas/${line.id}`} style={{ flex: '1 1 120px', minWidth: 0, fontSize: 14.5, fontWeight: 600, textDecoration: 'none', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {line.name}
      </Link>
      <StatePill state={line.state} />
      {action}
    </div>
  );
}
