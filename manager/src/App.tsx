import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { useAuth } from './auth';
import { Shell } from './components/ui';
import Login from './pages/Login';
import Panel from './pages/Panel';
import Lineas from './pages/Lineas';
import Linea from './pages/Linea';
import Vincular from './pages/Vincular';
import Eventos from './pages/Eventos';
import Grupos from './pages/Grupos';
import Chats from './pages/Chats';

export default function App() {
  const { session } = useAuth();
  return (
    <BrowserRouter basename="/manager">
      {session ? (
        <Shell>
          <Routes>
            <Route path="/" element={<Panel />} />
            <Route path="/lineas" element={<Lineas />} />
            <Route path="/lineas/:id" element={<Linea />} />
            <Route path="/lineas/:id/vincular" element={<Vincular />} />
            <Route path="/chats" element={<Chats />} />
            <Route path="/chats/:lineId" element={<Chats />} />
            <Route path="/chats/:lineId/:chat" element={<Chats />} />
            <Route path="/grupos" element={<Grupos />} />
            <Route path="/eventos" element={<Eventos />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Shell>
      ) : (
        <Routes>
          <Route path="*" element={<Login />} />
        </Routes>
      )}
    </BrowserRouter>
  );
}
