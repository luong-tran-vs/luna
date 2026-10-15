import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component } from '@angular/core';

/** The banner over the Listening player (design): the boy with headphones and what to do. */
@Component({
  selector: 'lu-listening-hero',
  imports: [NgOptimizedImage],
  template: `
    <img class="hero-img" ngSrc="images/cau-be-music.png" alt="" width="96" height="96" />
    <div class="hero-text">
      <span class="native"><span class="dot" aria-hidden="true"></span> Giọng chuẩn bản xứ</span>
      <p class="hero-title">Lắng nghe thật kỹ nhé!</p>
      <p class="hero-sub">Bấm nút Play để nghe và gõ lại từng từ</p>
    </div>
  `,
  styles: `
    :host {
      position: relative;
      display: flex;
      align-items: center;
      gap: var(--space-3);
      padding: var(--space-2);
      overflow: hidden;
      background: linear-gradient(120deg, var(--color-primary-soft), var(--color-surface));
      border: 1px solid var(--color-line);
      border-radius: var(--radius-lg);
    }

    :host::after {
      position: absolute;
      top: -40px;
      right: -30px;
      width: 120px;
      height: 120px;
      content: '';
      background: color-mix(in srgb, var(--color-primary) 10%, transparent);
      border-radius: var(--radius-full);
      pointer-events: none;
    }

    .hero-img {
      flex: none;
      width: 96px;
      height: 96px;
      object-fit: cover;
      background: var(--color-surface);
      border-radius: var(--radius-md);
    }

    .hero-text {
      position: relative;
      z-index: 1;
      display: grid;
      gap: var(--space-1);
      min-width: 0;
    }

    .native {
      display: inline-flex;
      align-items: center;
      justify-self: start;
      gap: var(--space-1);
      padding: 2px var(--space-2);
      font: var(--text-xs) var(--font-ui);
      font-weight: 600;
      color: var(--color-ok);
      background: color-mix(in srgb, var(--color-ok) 12%, var(--color-surface));
      border: 1px solid color-mix(in srgb, var(--color-ok) 30%, transparent);
      border-radius: var(--radius-full);
    }

    .native .dot {
      width: 6px;
      height: 6px;
      background: currentColor;
      border-radius: var(--radius-full);
    }

    .hero-title {
      font: var(--text-base) var(--font-ui);
      font-weight: 700;
    }

    .hero-sub {
      font: var(--text-xs) var(--font-ui);
      color: var(--color-text-muted);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ListeningHero {}
