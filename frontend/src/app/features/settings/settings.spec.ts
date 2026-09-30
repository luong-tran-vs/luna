import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { computed, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../core/interceptors/error-interceptor';
import { Settings as SettingsData } from '../../core/models/settings';
import { ThemePreference, ThemeSaveState, ThemeService } from '../../core/services/theme.service';
import { Settings } from './settings';

const stored: SettingsData = { theme: 'system', dailyReviewLimit: 30, timezone: 'Asia/Ho_Chi_Minh' };

describe('Settings', () => {
  let fixture: ComponentFixture<Settings>;
  let controller: HttpTestingController;
  let el: HTMLElement;
  const preference = signal<ThemePreference>('system');
  const saveState = signal<ThemeSaveState>('idle');
  const setTheme = vi.fn((p: ThemePreference) => preference.set(p));
  // Device is light in these tests, so "system" resolves to light.
  const resolved = computed(() => (preference() === 'dark' ? 'dark' : 'light'));

  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const limitInput = () => el.querySelector('#settings-limit') as HTMLInputElement;
  const zoneSelect = () => el.querySelector('#settings-zone') as HTMLSelectElement;
  const zoneSearch = () => el.querySelector('#settings-zone-search') as HTMLInputElement;
  const options = () => Array.from(zoneSelect().options).map((o) => o.value);
  const type = async (input: HTMLInputElement, value: string) => {
    input.value = value;
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const save = async () => {
    (el.querySelector('button[type="submit"]') as HTMLButtonElement).click();
    await fixture.whenStable();
  };

  const open = async (settings: SettingsData | 'error' = stored) => {
    fixture = TestBed.createComponent(Settings);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const req = controller.expectOne('/api/settings');
    if (settings === 'error') {
      req.flush('down', { status: 500, statusText: 'Error' });
    } else {
      req.flush(settings);
    }
    await fixture.whenStable();
  };

  beforeEach(async () => {
    preference.set('system');
    saveState.set('idle');
    setTheme.mockClear();
    await TestBed.configureTestingModule({
      imports: [Settings],
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ThemeService, useValue: { preference, resolved, saveState, set: setTheme } },
      ],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
  });

  afterEach(() => controller.verify());

  it('groups the settings under Giao diện, Học tập, Múi giờ and Dữ liệu', async () => {
    await open();
    expect(Array.from(el.querySelectorAll('h2')).map((h) => text(h))).toEqual(['Giao diện', 'Học tập', 'Múi giờ', 'Dữ liệu']);
  });

  // --- US1 ---

  it('shows only light and dark, with the current one checked', async () => {
    preference.set('dark');
    await open();
    const group = el.querySelector('[role="radiogroup"]')!;
    expect(group.getAttribute('aria-labelledby')).toBe('theme-heading');
    const radios = Array.from(group.querySelectorAll('input[type="radio"]')) as HTMLInputElement[];
    expect(radios.map((r) => text(r.parentElement))).toEqual(['Sáng', 'Tối']);
    expect(radios.map((r) => r.checked)).toEqual([false, true]);
  });

  it('applies and saves a theme at once', async () => {
    await open();
    const light = el.querySelector('input[value="light"]') as HTMLInputElement;
    light.click();
    await fixture.whenStable();
    expect(setTheme).toHaveBeenCalledWith('light');
    expect((el.querySelector('input[value="light"]') as HTMLInputElement).checked).toBe(true);

    saveState.set('saved');
    await fixture.whenStable();
    expect(text(el.querySelector('.group .save-status'))).toBe('Đã lưu');
    saveState.set('error');
    await fixture.whenStable();
    expect(text(el.querySelector('.group .save-status'))).toContain('Chưa lưu được vào tài khoản');
  });

  it('keeps the theme usable when the settings cannot be loaded', async () => {
    await open('error');
    expect(text(el.querySelector('[role="alert"] p'))).toBe('Không tải được cài đặt.');
    (el.querySelector('input[value="dark"]') as HTMLInputElement).click();
    expect(setTheme).toHaveBeenCalledWith('dark');

    (el.querySelector('[role="alert"] button') as HTMLButtonElement).click();
    controller.expectOne('/api/settings').flush(stored);
    await fixture.whenStable();
    expect(limitInput().value).toBe('30');
  });

  // --- US2 ---

  it('shows and saves the daily card limit with the timezone', async () => {
    await open();
    expect(limitInput().value).toBe('30');
    expect(limitInput().min).toBe('5');
    expect(limitInput().max).toBe('200');

    await type(limitInput(), '10');
    await save();
    const req = controller.expectOne({ method: 'PUT', url: '/api/settings' });
    expect(req.request.body).toEqual({ dailyReviewLimit: 10, timezone: 'Asia/Ho_Chi_Minh' });
    req.flush({ ...stored, dailyReviewLimit: 10 });
    await fixture.whenStable();
    expect(text(el.querySelector('.actions .save-status'))).toBe('Đã lưu');
  });

  it('rejects an empty or out-of-range limit without a request', async () => {
    await open();
    for (const bad of ['', '4', '201', '10.5']) {
      await type(limitInput(), bad);
      await save();
      expect(text(el.querySelector('#settings-limit-error'))).toBe('Số thẻ từ 5 đến 200');
      expect(limitInput().getAttribute('aria-invalid')).toBe('true');
      expect(limitInput().getAttribute('aria-describedby')).toContain('settings-limit-error');
    }
    controller.expectNone({ method: 'PUT', url: '/api/settings' });
  });

  it('shows the server field errors', async () => {
    await open();
    await save();
    controller.expectOne({ method: 'PUT', url: '/api/settings' }).flush(
      {
        error: 'validation_failed',
        fields: { dailyReviewLimit: 'Số thẻ từ 5 đến 200', timezone: 'Múi giờ không hợp lệ' },
      },
      { status: 400, statusText: 'Bad Request' },
    );
    await fixture.whenStable();
    expect(text(el.querySelector('#settings-limit-error'))).toBe('Số thẻ từ 5 đến 200');
    expect(text(el.querySelector('#settings-zone-error'))).toBe('Múi giờ không hợp lệ');
    expect(zoneSelect().getAttribute('aria-invalid')).toBe('true');
    expect(text(el.querySelector('.actions .save-status'))).toBe('');
  });

  it('reports a failed save', async () => {
    await open();
    await save();
    controller.expectOne({ method: 'PUT', url: '/api/settings' }).error(new ProgressEvent('error'));
    await fixture.whenStable();
    expect(text(el.querySelector('.actions .save-status'))).toBe('Không lưu được, vui lòng thử lại.');
  });

  // --- US3 ---

  it('lists the browser timezones with their offset and the stored one selected', async () => {
    await open();
    expect(zoneSelect().value).toBe('Asia/Ho_Chi_Minh');
    const hcm = Array.from(zoneSelect().options).find((o) => o.value === 'Asia/Ho_Chi_Minh')!;
    expect(hcm.textContent?.trim()).toBe('Asia/Ho_Chi_Minh (GMT+7)');
    expect(options().length).toBeGreaterThan(100);
  });

  it('filters the timezones case-insensitively and keeps the selected one', async () => {
    await open();
    await type(zoneSearch(), 'london');
    expect(options()).toContain('Europe/London');
    expect(options()).toContain('Asia/Ho_Chi_Minh');
    expect(options()).not.toContain('Asia/Tokyo');
    await type(zoneSearch(), 'ho chi');
    expect(options()).toEqual(['Asia/Ho_Chi_Minh']);
  });

  it('saves the chosen timezone', async () => {
    await open();
    await type(zoneSearch(), 'london');
    zoneSelect().value = 'Europe/London';
    zoneSelect().dispatchEvent(new Event('change'));
    await fixture.whenStable();
    await save();
    const req = controller.expectOne({ method: 'PUT', url: '/api/settings' });
    expect(req.request.body).toEqual({ dailyReviewLimit: 30, timezone: 'Europe/London' });
    req.flush({ ...stored, timezone: 'Europe/London' });
  });

  it('falls back to the current timezone when the browser lists none', async () => {
    const spy = vi.spyOn(Intl, 'supportedValuesOf').mockReturnValue([]);
    try {
      await open();
      const current = Intl.DateTimeFormat().resolvedOptions().timeZone;
      expect(options()).toContain(current);
      expect(options()).toContain('Asia/Ho_Chi_Minh');
      expect(options().length).toBeLessThanOrEqual(2);
    } finally {
      spy.mockRestore();
    }
  });

  // --- F13: data export ---

  describe('Xuất dữ liệu', () => {
    let clicked: { href: string; download: string }[];
    let revoke: ReturnType<typeof vi.fn<(url: string) => void>>;

    beforeEach(() => {
      clicked = [];
      revoke = vi.fn<(url: string) => void>();
      URL.createObjectURL = vi.fn(() => 'blob:luna-export');
      URL.revokeObjectURL = revoke;
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
        clicked.push({ href: this.href, download: this.download });
      });
    });

    afterEach(() => vi.restoreAllMocks());

    const button = () =>
      Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Xuất dữ liệu' || text(b) === 'Đang xuất…')!;

    it('shows the Dữ liệu group with the button described', async () => {
      await open();
      expect(Array.from(el.querySelectorAll('h2')).map((h) => text(h))).toContain('Dữ liệu');
      expect(button().getAttribute('aria-describedby')).toBe('export-hint');
      expect(text(el.querySelector('#export-hint'))).toContain('không có mật khẩu');
    });

    it('downloads the file with the server name', async () => {
      await open();
      button().click();
      await fixture.whenStable();
      expect(text(button())).toBe('Đang xuất…');
      expect(button().disabled).toBe(true);
      expect(button().getAttribute('aria-busy')).toBe('true');

      const req = controller.expectOne('/api/export');
      req.flush(new Blob(['{}'], { type: 'application/json' }), {
        headers: { 'Content-Disposition': 'attachment; filename="luna-export-20260930.json"' },
      });
      await new Promise((r) => setTimeout(r));
      await fixture.whenStable();
      expect(clicked).toEqual([{ href: 'blob:luna-export', download: 'luna-export-20260930.json' }]);
      expect(revoke).toHaveBeenCalledWith('blob:luna-export');
      expect(text(button())).toBe('Xuất dữ liệu');
      expect(button().disabled).toBe(false);
    });

    it('shows an error and lets the learner try again', async () => {
      await open();
      button().click();
      controller.expectOne('/api/export').error(new ProgressEvent('error'));
      await fixture.whenStable();
      expect(text(el.querySelector('.group [role="alert"]'))).toBe('Không xuất được dữ liệu, vui lòng thử lại.');
      expect(button().disabled).toBe(false);

      button().click();
      await fixture.whenStable();
      expect(el.querySelector('.group [role="alert"]')).toBeNull();
      controller.expectOne('/api/export').flush(new Blob(['{}']));
      await new Promise((r) => setTimeout(r));
      await fixture.whenStable();
      expect(clicked.length).toBe(1);
    });
  });
});
