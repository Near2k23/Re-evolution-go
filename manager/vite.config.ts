import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// El backend sirve /assets desde manager/dist/assets y /manager/* con index.html.
// En desarrollo, VITE_API_URL (o el campo Servidor del login) apunta al backend.
export default defineConfig({
  plugins: [react()],
  base: '/',
  build: { outDir: 'dist', assetsDir: 'assets', emptyOutDir: true },
  server: { port: 5173 },
});
