import type { App } from 'vue';
import Captcha from './components/Captcha.vue';

export { Captcha };
export * from './types';
export * from './composables/useCaptchaTrack';

export default {
  install(app: App) {
    app.component('Captcha', Captcha);
  }
};

