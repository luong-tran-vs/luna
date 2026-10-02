import { ChangeDetectionStrategy, Component } from '@angular/core';

/** Top of the login and register pages (client sketch, screen 1): logo, name, tagline, picture. */
@Component({
  selector: 'lu-auth-hero',
  template: `
    <div class="brand">
      <svg class="logo" viewBox="0 0 48 48" aria-hidden="true">
        <circle cx="24" cy="24" r="22" class="logo-bg" />
        <path d="M30 12a12 12 0 1 0 6 21A14 14 0 0 1 30 12z" class="logo-moon" />
      </svg>
      <p class="name">Luna</p>
      <p class="tagline">Học mỗi ngày – Tiến bộ từng bước</p>
    </div>
    <svg class="scene" viewBox="0 0 320 140" aria-hidden="true">
      <circle cx="262" cy="34" r="16" class="sun" />
      <path d="M0 110 C60 70 110 80 160 100 C210 120 260 80 320 96 V140 H0Z" class="hill-back" />
      <path d="M0 124 C70 104 130 112 180 122 C230 132 280 112 320 118 V140 H0Z" class="hill-front" />
      <g class="book">
        <path d="M118 96 L160 104 L160 128 L118 120Z" class="page-left" />
        <path d="M202 96 L160 104 L160 128 L202 120Z" class="page-right" />
        <path d="M126 102 L152 107 M126 109 L152 114 M168 107 L194 102 M168 114 L194 109" class="lines" />
      </g>
      <path d="M70 46 l3 7 7 3 -7 3 -3 7 -3 -7 -7 -3 7 -3z" class="star" />
      <path d="M210 60 l2 4 4 2 -4 2 -2 4 -2 -4 -4 -2 4 -2z" class="star" />
    </svg>
  `,
  styles: `
    :host {
      display: grid;
      gap: var(--space-4);
      text-align: center;
    }
    .brand {
      display: grid;
      justify-items: center;
      gap: var(--space-1);
    }
    .logo {
      width: 48px;
      height: 48px;
    }
    .logo-bg {
      fill: var(--color-primary-soft);
    }
    .logo-moon {
      fill: var(--color-primary);
    }
    .name {
      font: var(--text-xl) var(--font-ui);
      color: var(--color-primary-ink);
    }
    .tagline {
      font: var(--text-sm) var(--font-ui);
      color: var(--color-text-muted);
    }
    .scene {
      width: 100%;
      height: auto;
      border-radius: var(--radius-lg);
      background: var(--color-primary-soft);
    }
    .sun {
      fill: var(--color-accent);
    }
    .hill-back {
      fill: color-mix(in srgb, var(--color-skill-read) 35%, var(--color-surface));
    }
    .hill-front {
      fill: color-mix(in srgb, var(--color-skill-read) 60%, var(--color-surface));
    }
    .page-left,
    .page-right {
      fill: var(--color-surface);
      stroke: var(--color-primary);
      stroke-width: 2;
      stroke-linejoin: round;
    }
    .lines {
      stroke: var(--color-primary);
      stroke-width: 1.5;
      stroke-linecap: round;
      opacity: 0.5;
    }
    .star {
      fill: var(--color-primary);
      opacity: 0.6;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthHero {}
