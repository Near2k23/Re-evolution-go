import { createContext, useContext, useMemo, useState, type ReactNode } from 'react';
import { clearSession, loadSession, saveSession, setCurrentSession, type Session } from './api';

interface AuthCtx {
  session: Session | null;
  login: (s: Session, remember: boolean) => void;
  logout: () => void;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(() => loadSession());
  const value = useMemo<AuthCtx>(
    () => ({
      session,
      login: (s, remember) => {
        saveSession(s, remember);
        setCurrentSession(s);
        setSession(s);
      },
      logout: () => {
        clearSession();
        setCurrentSession(null);
        setSession(null);
      },
    }),
    [session],
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAuth(): AuthCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error('useAuth fuera de AuthProvider');
  return c;
}
