import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type Line, type StoredChat, type StoredMessage } from '../api';
import { useI18n, type I18n, type Key } from '../i18n';
import { byName, formatPhone, initials } from '../format';
import { ErrorBox, Icon, Loading, useOverview, useToast } from '../components/ui';

const MEDIA = ['image', 'video', 'audio', 'document', 'sticker', 'location', 'contact', 'poll', 'other'];

function jidLabel(jid: string) {
  const user = jid.split('@')[0];
  return jid.endsWith('@s.whatsapp.net') ? formatPhone(user) : user;
}

function chatName(c: StoredChat) {
  return c.displayName || jidLabel(c.jid);
}

function mediaLabel(i: I18n, type: string) {
  return MEDIA.includes(type) ? i.t(`media.${type}` as Key) : '';
}

function previewOf(i: I18n, c: StoredChat) {
  const media = c.lastType && c.lastType !== 'text' ? mediaLabel(i, c.lastType) : '';
  const text = [media, c.lastPreview].filter(Boolean).join(' · ');
  return (c.lastFromMe ? i.t('chats.you') + ': ' : '') + text;
}

function dayLabel(i: I18n, d: Date) {
  const today = new Date();
  const y = new Date();
  y.setDate(today.getDate() - 1);
  if (d.toDateString() === today.toDateString()) return i.t('chats.today');
  if (d.toDateString() === y.toDateString()) return i.t('chats.yesterday');
  return d.toLocaleDateString(i.tag, { weekday: 'long', day: 'numeric', month: 'long' });
}

function listTime(i: I18n, iso?: string) {
  if (!iso) return '';
  const d = new Date(iso);
  if (d.toDateString() === new Date().toDateString()) return d.toLocaleTimeString(i.tag, { hour: '2-digit', minute: '2-digit' });
  return d.toLocaleDateString(i.tag, { day: 'numeric', month: 'short' });
}

export default function Chats() {
  const i = useI18n();
  const { t } = i;
  const { lineId, chat: chatParam } = useParams();
  const navigate = useNavigate();
  const { data: lines, isLoading } = useOverview();
  const linked = useMemo(() => (lines ?? []).filter((l) => l.jid).sort(byName), [lines]);
  const line = linked.find((l) => l.id === lineId);
  const chat = chatParam ? decodeURIComponent(chatParam) : undefined;

  useEffect(() => {
    if (!lineId && linked.length) navigate(`/chats/${linked.find((l) => l.state !== 'caida')?.id ?? linked[0].id}`, { replace: true });
  }, [lineId, linked, navigate]);

  if (isLoading) return <Loading />;
  if (!linked.length) {
    return (
      <div className="empty">
        <b>{t('chats.noLines')}</b>
        <Link className="btn btn-dark" to="/lineas">{t('nav.lines')}</Link>
      </div>
    );
  }
  if (!line) return <Loading />;

  return (
    <div className={`chats ${chat ? 'has-chat' : ''}`}>
      <ChatList line={line} lines={linked} active={chat} />
      {chat ? (
        <Conversation key={line.id + chat} line={line} chat={chat} />
      ) : (
        <section className="chat-pane chat-empty">
          <span className="muted">{t('chats.pick')}</span>
        </section>
      )}
    </div>
  );
}

function ChatList({ line, lines, active }: { line: Line; lines: Line[]; active?: string }) {
  const i = useI18n();
  const { t } = i;
  const navigate = useNavigate();
  const [q, setQ] = useState('');
  const [archived, setArchived] = useState(false);
  const status = useQuery({ queryKey: ['storeStatus', line.id], queryFn: () => api.storeStatus(line.id), staleTime: 60000 });
  const chats = useQuery({ queryKey: ['chats', line.id, q, archived], queryFn: () => api.chats(line.id, q, archived), refetchInterval: 5000, enabled: status.data?.enabled !== false });

  return (
    <section className="chat-list" aria-label={t('chats.title')}>
      <label className="field" style={{ fontSize: 12.5, color: 'var(--ink-3)' }}>
        {t('chats.line')}
        <select className="input on-surface" style={{ height: 44, fontSize: 14, fontWeight: 600 }} value={line.id} onChange={(e) => navigate(`/chats/${e.target.value}`)}>
          {lines.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name} · {t(`state.${l.state}.label` as Key)}
            </option>
          ))}
        </select>
      </label>
      <label className="search" style={{ minWidth: 0, background: 'var(--ground)' }}>
        <Icon name="buscar" size={18} />
        <span className="sr-only">{t('chats.search')}</span>
        <input type="search" placeholder={t('chats.search')} value={q} onChange={(e) => setQ(e.target.value)} />
      </label>
      <div className="segmented" role="group" style={{ alignSelf: 'flex-start' }}>
        <button type="button" aria-pressed={!archived} onClick={() => setArchived(false)}>{t('chats.inbox')}</button>
        <button type="button" aria-pressed={archived} onClick={() => setArchived(true)}>{t('chats.archived')}</button>
      </div>

      <div className="chat-items">
        {status.data?.enabled === false && <div className="alert warn">{t('chats.disabled')}</div>}
        {chats.isLoading && <span className="spinner" style={{ margin: 16 }} />}
        {chats.error && <ErrorBox error={chats.error} />}
        {chats.data?.length === 0 && (
          <div style={{ padding: '24px 8px', display: 'flex', flexDirection: 'column', gap: 6 }}>
            <b>{t('chats.empty')}</b>
            <span className="muted" style={{ fontSize: 13.5, lineHeight: 1.5 }}>{t('chats.emptyHint')}</span>
          </div>
        )}
        {chats.data?.map((c) => (
          <Link key={c.jid} to={`/chats/${line.id}/${encodeURIComponent(c.jid)}`} className="chat-item" aria-current={active === c.jid ? 'page' : undefined}>
            <span className={`avatar ${c.isGroup ? 'g-group' : ''}`} aria-hidden="true">
              {c.isGroup ? <Icon name="grupos" size={18} /> : initials(chatName(c))}
            </span>
            <span className="ci-body">
              <span className="ci-row">
                <span className="ci-name">{chatName(c)}</span>
                <span className="ci-time">{listTime(i, c.lastMessageAt)}</span>
              </span>
              <span className="ci-row">
                <span className="ci-prev">{previewOf(i, c)}</span>
                {c.unread > 0 && <span className="ci-badge" aria-label={t('chats.unread', { n: c.unread })}>{c.unread}</span>}
              </span>
            </span>
          </Link>
        ))}
      </div>
    </section>
  );
}

function Ticks({ status }: { status: StoredMessage['status'] }) {
  const { t } = useI18n();
  if (status === 'received') return null;
  const double = status !== 'sent';
  const read = status === 'read' || status === 'played';
  return (
    <svg width="16" height="12" viewBox="0 0 16 12" role="img" aria-label={t(`status.${status}` as Key)} style={{ color: read ? '#7BC4FF' : 'currentColor' }}>
      <path d="M1 6.5l3 3 6-7" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
      {double && <path d="M6.5 9.5l1 0.5 6-8" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />}
    </svg>
  );
}

function Conversation({ line, chat }: { line: Line; chat: string }) {
  const i = useI18n();
  const { t } = i;
  const qc = useQueryClient();
  const toast = useToast();
  const bottom = useRef<HTMLDivElement>(null);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const isGroup = chat.endsWith('@g.us');
  const { data: chatInfo } = useQuery({ queryKey: ['chat', line.id, chat], queryFn: () => api.chat(line.id, chat), staleTime: 30000 });
  const name = chatInfo ? chatName(chatInfo) : jidLabel(chat);
  const canSend = line.state === 'conectada' || line.state === 'reconectada';

  const q = useInfiniteQuery({
    queryKey: ['messages', line.id, chat],
    queryFn: ({ pageParam }) => api.messages(line.id, chat, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.length === 50 ? last[last.length - 1].timestamp : undefined),
    refetchInterval: 4000,
  });
  const msgs = useMemo(() => (q.data?.pages.flat() ?? []).slice().reverse(), [q.data]);
  const newest = msgs[msgs.length - 1]?.id;

  useEffect(() => {
    api.markRead(line.id, chat).then(() => qc.invalidateQueries({ queryKey: ['chats', line.id] }), () => {});
  }, [line.id, chat, newest, qc]);
  useEffect(() => {
    bottom.current?.scrollIntoView({ block: 'end' });
  }, [newest]);

  async function send(e: FormEvent) {
    e.preventDefault();
    const body = text.trim();
    if (!body) return;
    setSending(true);
    try {
      await api.sendText(line.token, isGroup ? chat : chat.split('@')[0], body);
      setText('');
      setTimeout(() => q.refetch(), 400);
    } catch (err) {
      toast(err instanceof Error ? err.message : t('chats.sendFailed'));
    } finally {
      setSending(false);
    }
  }

  let lastDay = '';
  return (
    <section className="chat-pane" aria-label={name}>
      <header className="chat-head">
        <Link to={`/chats/${line.id}`} className="btn btn-soft btn-icon chat-back" aria-label={t('chats.back')}>
          <Icon name="atras" size={20} />
        </Link>
        <span className={`avatar ${isGroup ? 'g-group' : ''}`} aria-hidden="true">
          {isGroup ? <Icon name="grupos" size={18} /> : initials(name)}
        </span>
        <div style={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <span style={{ fontSize: 16, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{name}</span>
          <span className="muted" style={{ fontSize: 12.5 }}>
            {isGroup ? t('chats.group') : jidLabel(chat)} · {line.name}
          </span>
        </div>
      </header>

      <div className="chat-scroll">
        {q.hasNextPage && (
          <button type="button" className="btn btn-light btn-sm" style={{ alignSelf: 'center' }} onClick={() => q.fetchNextPage()} disabled={q.isFetchingNextPage}>
            {q.isFetchingNextPage ? <span className="spinner" /> : t('chats.loadOlder')}
          </button>
        )}
        {q.isLoading && <span className="spinner" style={{ alignSelf: 'center' }} />}
        {msgs.map((m) => {
          const d = new Date(m.timestamp);
          const day = d.toDateString();
          const sep = day !== lastDay ? dayLabel(i, d) : null;
          lastDay = day;
          const media = m.type !== 'text' ? mediaLabel(i, m.type) : '';
          return (
            <div key={m.id} style={{ display: 'contents' }}>
              {sep && <span className="day-sep">{sep}</span>}
              <div className={`bubble-row ${m.fromMe ? 'out' : 'in'}`}>
                <div className={`bubble ${m.fromMe ? 'out' : 'in'} ${m.deleted ? 'deleted' : ''}`}>
                  {isGroup && !m.fromMe && <span className="b-sender">{m.senderName || jidLabel(m.sender)}</span>}
                  {m.deleted ? (
                    <span className="b-text">{t('chats.deleted')}</span>
                  ) : (
                    <>
                      {media && (
                        <span className="b-media">
                          {media}
                          {m.fileName ? ` · ${m.fileName}` : ''}
                        </span>
                      )}
                      {m.text && <span className="b-text">{m.text}</span>}
                    </>
                  )}
                  <span className="b-meta">
                    {m.edited && !m.deleted && <span>{t('chats.edited')}</span>}
                    <span>{d.toLocaleTimeString(i.tag, { hour: '2-digit', minute: '2-digit' })}</span>
                    {m.fromMe && <Ticks status={m.status} />}
                  </span>
                </div>
              </div>
            </div>
          );
        })}
        <div ref={bottom} />
      </div>

      {canSend ? (
        <form className="composer" onSubmit={send}>
          <label style={{ flexGrow: 1, minWidth: 0, display: 'flex' }}>
            <span className="sr-only">{t('chats.type')}</span>
            <textarea
              rows={1}
              className="input on-surface"
              placeholder={t('chats.type')}
              value={text}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  (e.currentTarget.form as HTMLFormElement).requestSubmit();
                }
              }}
              style={{ resize: 'none', minHeight: 46, maxHeight: 160 }}
            />
          </label>
          <button type="submit" className="btn btn-dark btn-icon" style={{ height: 46, width: 46 }} aria-label={t('chats.send')} disabled={sending || !text.trim()}>
            {sending ? <span className="spinner" /> : <Icon name="enviar" size={20} />}
          </button>
        </form>
      ) : (
        <div className="alert warn" style={{ margin: 12 }}>{t('chats.lineDown')}</div>
      )}
    </section>
  );
}
