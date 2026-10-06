import { DOCUMENT } from '@angular/common';
import { effect, inject, Injectable, signal } from '@angular/core';

/** Also read by the inline script in index.html; keep both in sync. */
export const PALETTE_STORAGE_KEY = 'luna.palette';

export type PaletteId = 'luna' | 'sky' | 'ocean' | 'jade' | 'rose' | 'forest' | 'graphite';

export interface Palette {
  id: PaletteId;
  name: string;
  note: string;
}

/** The color palettes; their colors are in styles/tokens.css (luna) and styles/palettes.css. */
export const PALETTES: readonly Palette[] = [
  { id: 'luna', name: 'Tím', note: 'Mặc định, kèm chút xanh ngọc' },
  { id: 'sky', name: 'Xanh da trời', note: 'Xanh dương nhạt, kèm chút xanh ngọc' },
  { id: 'ocean', name: 'Xanh dương', note: 'Xanh dương đậm, kèm chút tím' },
  { id: 'jade', name: 'Xanh ngọc', note: 'Xanh pha lục, dịu mắt' },
  { id: 'rose', name: 'Hồng', note: 'Hồng đậm, kèm chút xanh ngọc' },
  { id: 'forest', name: 'Xanh lá', note: 'Xanh lá đậm, kèm chút xanh thép' },
  { id: 'graphite', name: 'Xám', note: 'Xám, kèm chút xanh ngọc' },
];

const DEFAULT_PALETTE: PaletteId = 'luna';

function isPalette(value: unknown): value is PaletteId {
  return PALETTES.some((p) => p.id === value);
}

/**
 * Holds the color palette of the app (admin and learner pages alike), applies it to
 * <html data-palette> and remembers it in this browser. Storage failures never break the page.
 */
@Injectable({ providedIn: 'root' })
export class PaletteService {
  private readonly document = inject(DOCUMENT);
  private readonly storage = this.document.defaultView?.localStorage;

  private readonly _palette = signal<PaletteId>(this.readStored());

  readonly palette = this._palette.asReadonly();

  constructor() {
    effect(() => {
      const root = this.document.documentElement;
      const palette = this._palette();
      if (palette === DEFAULT_PALETTE) {
        root.removeAttribute('data-palette');
      } else {
        root.setAttribute('data-palette', palette);
      }
    });
  }

  /** Applies a palette at once and remembers it. */
  set(palette: PaletteId): void {
    this._palette.set(palette);
    try {
      this.storage?.setItem(PALETTE_STORAGE_KEY, palette);
    } catch {
      // Storage blocked: the palette still applies for this visit.
    }
  }

  private readStored(): PaletteId {
    try {
      const value = this.storage?.getItem(PALETTE_STORAGE_KEY);
      return isPalette(value) ? value : DEFAULT_PALETTE;
    } catch {
      return DEFAULT_PALETTE;
    }
  }
}
