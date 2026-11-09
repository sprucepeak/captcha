import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// Demo development config
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    open: true,
    proxy: {
      '/captcha': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
});

