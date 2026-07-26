import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.ESCALITE_API_PROXY_TARGET ?? 'http://localhost:8080'
  const frameAncestors = env.ESCALITE_STATUS_PAGE_FRAME_ANCESTORS ?? '*'

  return {
    plugins: [react(), tailwindcss()],
    server: {
      host: '0.0.0.0',
      port: 5174,
      headers: {
        'Content-Security-Policy': `frame-ancestors ${frameAncestors}`,
      },
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
    preview: {
      host: '0.0.0.0',
      port: 5174,
      headers: {
        'Content-Security-Policy': `frame-ancestors ${frameAncestors}`,
      },
    },
  }
})
