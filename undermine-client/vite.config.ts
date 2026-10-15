import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // Read .env (VITE_HOST / VITE_PORT / VITE_SERVER_URL) explicitly —
  // Vite does not expose .env values to process.env inside the config file.
  const env = loadEnv(mode, process.cwd(), '');

  const host = env.VITE_HOST || '0.0.0.0';
  const port = parseInt(env.VITE_PORT || '5173', 10);
  const serverUrl = env.VITE_SERVER_URL || 'http://localhost:8080';

  return {
    base: './',
    plugins: [
      react(),
      tailwindcss(),
    ],
    server: {
      host,
      port,
      allowedHosts: true,
      proxy: {
        // Fallback for same-origin requests when no absolute server URL is configured
        '/api': { target: serverUrl, changeOrigin: true },
        '/ws': { target: serverUrl, ws: true, changeOrigin: true },
      },
    },
  };
});
