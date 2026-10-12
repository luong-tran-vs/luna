import { DOCUMENT } from '@angular/common';
import { computed, inject, Injectable, signal } from '@angular/core';

/** Chrome/Edge/Android's install prompt event; not in the TypeScript DOM types yet. */
interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>;
  readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

/**
 * How this browser can install the PWA:
 * - `prompt`: one click opens the browser's own install dialog (Chrome, Edge, Android).
 * - `ios`: Safari on iPhone/iPad has no such dialog; the learner adds it via Share → Add to Home Screen.
 * - `installed`: already running as the installed app.
 * - `unavailable`: anything else (Firefox desktop, already installed elsewhere, `ng serve`).
 */
export type InstallMode = 'prompt' | 'ios' | 'installed' | 'unavailable';

/**
 * Keeps the browser's install prompt so the account page can offer an "Install app" button.
 * The prompt fires once, early in the page load, so `start()` runs as an app initializer.
 */
@Injectable({ providedIn: 'root' })
export class InstallService {
  private readonly window = inject(DOCUMENT).defaultView;

  private readonly deferred = signal<BeforeInstallPromptEvent | null>(null);
  private readonly installed = signal(this.runsStandalone());
  private readonly ios = this.isIosSafari();

  readonly mode = computed<InstallMode>(() => {
    if (this.installed()) return 'installed';
    if (this.deferred()) return 'prompt';
    return this.ios ? 'ios' : 'unavailable';
  });

  start(): void {
    this.window?.addEventListener('beforeinstallprompt', (event) => {
      // Keep the browser's mini-infobar quiet; the account page offers the button instead.
      event.preventDefault();
      this.deferred.set(event as BeforeInstallPromptEvent);
    });
    this.window?.addEventListener('appinstalled', () => {
      this.deferred.set(null);
      this.installed.set(true);
    });
  }

  /** Opens the browser's install dialog; true when the learner accepted. */
  async install(): Promise<boolean> {
    const event = this.deferred();
    if (!event) return false;
    // A prompt can be shown only once; the browser fires a new event if it may ask again.
    this.deferred.set(null);
    await event.prompt();
    const { outcome } = await event.userChoice;
    return outcome === 'accepted';
  }

  private runsStandalone(): boolean {
    const nav = this.window?.navigator as (Navigator & { standalone?: boolean }) | undefined;
    return !!this.window?.matchMedia?.('(display-mode: standalone)').matches || nav?.standalone === true;
  }

  private isIosSafari(): boolean {
    const nav = this.window?.navigator;
    if (!nav) return false;
    // iPadOS reports itself as a Mac; touch points tell them apart.
    const ios = /iPad|iPhone|iPod/.test(nav.userAgent) || (nav.platform === 'MacIntel' && nav.maxTouchPoints > 1);
    // Other iOS browsers (Chrome, Firefox) mark themselves in the user agent and cannot add PWAs on older iOS.
    return ios && !/CriOS|FxiOS|EdgiOS/.test(nav.userAgent);
  }
}
