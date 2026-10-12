import { DOCUMENT } from '@angular/common';
import { TestBed } from '@angular/core/testing';

import { InstallService } from './install.service';

describe('InstallService', () => {
  const IPHONE = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile Safari/604.1';
  const CHROME_IPHONE = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) CriOS/130.0 Mobile/15E148 Safari/604.1';
  const DESKTOP = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/130.0 Safari/537.36';

  function setup(opts: { userAgent?: string; standalone?: boolean } = {}) {
    const window = new EventTarget() as EventTarget & Record<string, unknown>;
    window['navigator'] = { userAgent: opts.userAgent ?? DESKTOP, platform: 'Win32', maxTouchPoints: 0 };
    window['matchMedia'] = () => ({ matches: opts.standalone ?? false });
    TestBed.configureTestingModule({
      providers: [{ provide: DOCUMENT, useValue: { defaultView: window } }],
    });
    const service = TestBed.inject(InstallService);
    service.start();
    return { service, window };
  }

  function promptEvent(outcome: 'accepted' | 'dismissed') {
    const event = new Event('beforeinstallprompt', { cancelable: true });
    return Object.assign(event, {
      prompt: vi.fn(async () => undefined),
      userChoice: Promise.resolve({ outcome }),
    });
  }

  it('is unavailable until the browser offers to install', () => {
    expect(setup().service.mode()).toBe('unavailable');
  });

  it('keeps the browser prompt and opens it on install', async () => {
    const { service, window } = setup();
    const event = promptEvent('accepted');
    window.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(true);
    expect(service.mode()).toBe('prompt');

    await expect(service.install()).resolves.toBe(true);
    expect(event.prompt).toHaveBeenCalledOnce();
    // A prompt can be used only once.
    expect(service.mode()).toBe('unavailable');
  });

  it('reports a dismissed dialog', async () => {
    const { service, window } = setup();
    window.dispatchEvent(promptEvent('dismissed'));
    await expect(service.install()).resolves.toBe(false);
  });

  it('is installed once the app is installed or when it runs standalone', () => {
    const { service, window } = setup();
    window.dispatchEvent(promptEvent('accepted'));
    window.dispatchEvent(new Event('appinstalled'));
    expect(service.mode()).toBe('installed');

    TestBed.resetTestingModule();
    expect(setup({ standalone: true }).service.mode()).toBe('installed');
  });

  it('gives the Add to Home Screen steps in Safari on iPhone only', () => {
    expect(setup({ userAgent: IPHONE }).service.mode()).toBe('ios');
    TestBed.resetTestingModule();
    expect(setup({ userAgent: CHROME_IPHONE }).service.mode()).toBe('unavailable');
  });
});
