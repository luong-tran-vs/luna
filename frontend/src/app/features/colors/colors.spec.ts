import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { PALETTE_STORAGE_KEY } from '../../core/services/palette.service';
import { Colors } from './colors';

describe('Colors', () => {
  let fixture: ComponentFixture<Colors>;
  let el: HTMLElement;
  const html = document.documentElement;
  const radio = (id: string) => el.querySelector<HTMLInputElement>(`input[name="palette"][value="${id}"]`)!;

  beforeEach(async () => {
    localStorage.clear();
    html.removeAttribute('data-palette');
    await TestBed.configureTestingModule({ imports: [Colors], providers: [provideRouter([])] }).compileComponents();
    fixture = TestBed.createComponent(Colors);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => {
    localStorage.clear();
    html.removeAttribute('data-palette');
  });

  it('goes back to the account tab and names the color group', () => {
    expect(el.querySelector('a[aria-label="Quay lại Tài khoản"]')!.getAttribute('href')).toBe('/account');
    expect(el.querySelector('h1')!.textContent).toContain('Màu giao diện');
    expect(el.querySelector('[role="radiogroup"]')!.getAttribute('aria-labelledby')).toBe('colors-heading');
  });

  it('applies a chosen color to the whole app at once', async () => {
    expect(radio('luna').checked).toBe(true);
    radio('sky').click();
    await fixture.whenStable();
    expect(html.getAttribute('data-palette')).toBe('sky');
    expect(localStorage.getItem(PALETTE_STORAGE_KEY)).toBe('sky');

    radio('luna').click();
    await fixture.whenStable();
    expect(html.hasAttribute('data-palette')).toBe(false);
  });
});
