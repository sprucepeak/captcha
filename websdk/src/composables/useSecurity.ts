/**
 * Security & Anti-Bot Fingerprint utilities
 * Collects browser environment info (WebGL/Canvas/WebDriver) and performs AES-GCM encryption
 */

export interface ClientEnvInfo {
  webdriver: boolean;
  platform: string;
  userAgent: string;
  language: string;
  screenWidth: number;
  screenHeight: number;
  colorDepth: number;
  devicePixelRatio: number;
  canvasHash: string;
  timezone: number;
}

/**
 * Collect browser environment anti-bot fingerprint
 */
export function collectClientEnv(): ClientEnvInfo {
  let webdriver = false;
  try {
    webdriver = !!(
      navigator.webdriver ||
      (window as any).__nightmare ||
      (window as any)._phantom ||
      (window as any).callPhantom ||
      (document as any).__selenium_unwrapped ||
      (document as any).__webdriver_evaluate ||
      (document as any).__driver_evaluate
    );
  } catch (e) {
    webdriver = false;
  }

  // Simple Canvas hash
  let canvasHash = '';
  try {
    const canvas = document.createElement('canvas');
    canvas.width = 100;
    canvas.height = 30;
    const ctx = canvas.getContext('2d');
    if (ctx) {
      ctx.textBaseline = 'top';
      ctx.font = '14px Arial';
      ctx.fillStyle = '#f60';
      ctx.fillRect(10, 5, 60, 20);
      ctx.fillStyle = '#069';
      ctx.fillText('tac#2026', 15, 8);
      canvasHash = canvas.toDataURL().slice(-32);
    }
  } catch (e) {
    canvasHash = 'unavailable';
  }

  return {
    webdriver,
    platform: navigator.platform || '',
    userAgent: navigator.userAgent || '',
    language: navigator.language || '',
    screenWidth: window.screen ? window.screen.width : 0,
    screenHeight: window.screen ? window.screen.height : 0,
    colorDepth: window.screen ? window.screen.colorDepth : 0,
    devicePixelRatio: window.devicePixelRatio || 1,
    canvasHash,
    timezone: new Date().getTimezoneOffset()
  };
}

/**
 * Derive 256-bit AES key using Web Crypto Subtle API
 * Exactly matches Go backend: sha256("${captchaID}_salt_#2026_${timestamp}")
 */
async function deriveAesKey(captchaID: string, timestamp: number): Promise<CryptoKey> {
  const enc = new TextEncoder();
  const rawSalt = `${captchaID}_salt_#2026_${timestamp}`;
  const hashBuffer = await crypto.subtle.digest('SHA-256', enc.encode(rawSalt));
  return crypto.subtle.importKey('raw', hashBuffer, { name: 'AES-GCM' }, false, ['encrypt']);
}

/**
 * Encrypt track data and client environment fingerprint using Web Crypto AES-GCM
 */
export async function encryptPayload(
  captchaID: string,
  timestamp: number,
  payload: any
): Promise<string> {
  // If Web Crypto is unavailable, return empty string for fallback
  if (!window.crypto || !window.crypto.subtle) {
    return '';
  }

  try {
    const key = await deriveAesKey(captchaID, timestamp);
    const iv = crypto.getRandomValues(new Uint8Array(12)); // 12-byte GCM nonce
    const encodedData = new TextEncoder().encode(JSON.stringify(payload));

    const cipherBuffer = await crypto.subtle.encrypt(
      {
        name: 'AES-GCM',
        iv
      },
      key,
      encodedData
    );

    // Concat iv + ciphertext and encode as Base64 (matches Go GCM format)
    const combined = new Uint8Array(iv.length + cipherBuffer.byteLength);
    combined.set(iv, 0);
    combined.set(new Uint8Array(cipherBuffer), iv.length);

    let binary = '';
    const len = combined.byteLength;
    for (let i = 0; i < len; i++) {
      binary += String.fromCharCode(combined[i]);
    }
    return btoa(binary);
  } catch (e) {
    console.warn('[Captcha Security] Encrypt payload failed, using standard mode:', e);
    return '';
  }
}

