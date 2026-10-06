import { TestBed } from '@angular/core/testing';

import { PALETTE_STORAGE_KEY, PaletteService } from './palette.service';

describe('PaletteService', () => {
  const html = document.documentElement;

  const create = () => {
    const service = TestBed.inject(PaletteService);
    TestBed.tick();
    return service;
  };

  beforeEach(() => {
    localStorage.clear();
    html.removeAttribute('data-palette');
    TestBed.configureTestingModule({});
  });

  afterEach(() => html.removeAttribute('data-palette'));

  it('starts with the default palette and sets nothing on the page', () => {
    const service = create();
    expect(service.palette()).toBe('luna');
    expect(html.hasAttribute('data-palette')).toBe(false);
  });

  it('applies a chosen palette to the page at once and remembers it', () => {
    const service = create();
    service.set('ocean');
    TestBed.tick();
    expect(html.getAttribute('data-palette')).toBe('ocean');
    expect(localStorage.getItem(PALETTE_STORAGE_KEY)).toBe('ocean');
  });

  it('reads the stored palette and ignores an unknown one', () => {
    localStorage.setItem(PALETTE_STORAGE_KEY, 'jade');
    expect(create().palette()).toBe('jade');
    expect(html.getAttribute('data-palette')).toBe('jade');

    TestBed.resetTestingModule();
    localStorage.setItem(PALETTE_STORAGE_KEY, 'neon');
    expect(create().palette()).toBe('luna');
    expect(html.hasAttribute('data-palette')).toBe(false);
  });

  it('removes the attribute when going back to the default palette', () => {
    const service = create();
    service.set('rose');
    TestBed.tick();
    expect(html.getAttribute('data-palette')).toBe('rose');
    service.set('luna');
    TestBed.tick();
    expect(html.hasAttribute('data-palette')).toBe(false);
  });
});
