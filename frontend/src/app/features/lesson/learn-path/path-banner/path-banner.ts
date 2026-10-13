import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Icon } from '../../../../shared/components/icon/icon';

/** Progress shown at the bottom of the banner. */
export interface BannerProgress {
  /** What the progress is of (a roadmap's name, or "Cả phần"), for the progress bar's name. */
  label: string;
  done: number;
  total: number;
  percent: number;
}

/**
 * The blue banner at the top of a part (design/lo-trinh.png): back to "Học", the part's name, the
 * roadmap named in it, how many roadmaps the part has and the progress; "Hướng dẫn" opens a short
 * card on how the path works. A part with no roadmap yet shows plain and locked.
 */
@Component({
  selector: 'lu-path-banner',
  imports: [Icon, RouterLink],
  templateUrl: './path-banner.html',
  styleUrl: './path-banner.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '[class.is-empty]': 'roadmaps() === 0' },
})
export class PathBanner {
  /** 1-based number of the part. */
  readonly number = input.required<number>();
  readonly level = input.required<string>();
  readonly levelName = input.required<string>();
  readonly title = input.required<string>();
  readonly text = input.required<string>();
  /** How many roadmaps the part has. */
  readonly roadmaps = input.required<number>();
  readonly progress = input<BannerProgress | null>(null);

  /** Id of the title, which names the part's section. */
  protected readonly titleId = computed(() => `part-${this.level()}`);
  protected readonly guideOpen = signal(false);
}
