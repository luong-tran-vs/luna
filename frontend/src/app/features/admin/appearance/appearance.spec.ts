import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PALETTE_STORAGE_KEY } from '../../../core/services/palette.service';
import { ThemePreference, ThemeService } from '../../../core/services/theme.service';
import { Appearance } from './appearance';

describe('Appearance', () => {
  let fixture: ComponentFixture<Appearance>;
  let el: HTMLElement;
  const html = document.documentElement;
  const preference = signal<ThemePreference>('system');
  const themeSet = vi.fn((p: ThemePreference) => preference.set(p));

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const radio = (id: string) => el.querySelector<HTMLInputElement>(`input[name="palette"][value="${id}"]`)!;
  const modeButton = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.modes button')).find((b) => text(b) === label)!;

  beforeEach(async () => {
    localStorage.clear();
    html.removeAttribute('data-palette');
    preference.set('system');
    themeSet.mockClear();
    TestBed.configureTestingModule({
      providers: [
        { provide: ThemeService, useValue: { preference, saveState: signal('idle'), set: themeSet } },
      ],
    });
    fixture = TestBed.createComponent(Appearance);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  afterEach(() => html.removeAttribute('data-palette'));

  it('lists every palette with its own colors and checks the current one', () => {
    const options = Array.from(el.querySelectorAll('label.palette'));
    expect(options.map((o) => text(o.querySelector('.palette-name')))).toEqual([
      'Tím', 'Xanh da trời', 'Xanh dương', 'Xanh ngọc', 'Hồng', 'Xanh lá', 'Xám',
    ]);
    expect(options.map((o) => o.getAttribute('data-palette'))).toEqual(['luna', 'sky', 'ocean', 'jade', 'rose', 'forest', 'graphite']);
    expect(radio('luna').checked).toBe(true);
  });

  it('applies a palette to the whole page as soon as it is chosen', async () => {
    radio('forest').click();
    await fixture.whenStable();
    expect(html.getAttribute('data-palette')).toBe('forest');
    expect(localStorage.getItem(PALETTE_STORAGE_KEY)).toBe('forest');
    expect(radio('forest').checked).toBe(true);
  });

  it('switches between light, dark and the device setting', async () => {
    expect(modeButton('Theo thiết bị').getAttribute('aria-pressed')).toBe('true');
    modeButton('Tối').click();
    await fixture.whenStable();
    expect(themeSet).toHaveBeenCalledWith('dark');
    expect(modeButton('Tối').getAttribute('aria-pressed')).toBe('true');
    expect(modeButton('Sáng').getAttribute('aria-pressed')).toBe('false');
  });
});
