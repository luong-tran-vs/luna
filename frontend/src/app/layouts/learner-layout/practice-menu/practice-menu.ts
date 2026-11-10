import { DOCUMENT } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  afterNextRender,
  inject,
  Injector,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationStart, Router, RouterLink } from '@angular/router';
import { filter } from 'rxjs';

import { DashboardLesson } from '../../../core/models/dashboard';
import { VerbProgress } from '../../../core/models/verbs';
import { DashboardApiService } from '../../../core/services/dashboard-api.service';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { strugglingVerbs } from '../../../features/irregular-verbs/verb-mistakes.logic';
import { VERB_LESSONS } from '../../../features/irregular-verbs/verb-lessons.data';
import { nextLesson } from '../../../features/irregular-verbs/verb-study.logic';
import { Icon, IconName } from '../../../shared/components/icon/icon';

interface Item {
  label: string;
  icon: IconName;
  link: string;
  /** A short state under the name ("Bài 9/42", "3 từ"). */
  note: string;
  /** A count in a red badge (the verbs often typed wrong). */
  badge: number;
}

/**
 * The raised middle button of the tab bar, "Luyện" (design/de-xuat-nut-giua-noi-len.png, option 3,
 * added 2026-11-10): it opens a card over the bar with the lesson to go on with and a grid of the
 * practice sections. Tapping the button again (now ✕), outside the card, Esc or going to a page
 * closes it. A red dot on the button tells there are verbs often typed wrong.
 */
@Component({
  selector: 'lu-practice-menu',
  imports: [RouterLink, Icon],
  templateUrl: './practice-menu.html',
  styleUrl: './practice-menu.css',
  host: { '(document:keydown.escape)': 'close(true)' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PracticeMenu {
  /** Whether the page shown is one of the practice sections. */
  readonly active = input(false);

  private readonly dashboardApi = inject(DashboardApiService);
  private readonly verbsApi = inject(VerbsApiService);
  private readonly injector = inject(Injector);
  private readonly document = inject(DOCUMENT);
  private readonly toggleButton = viewChild.required<ElementRef<HTMLButtonElement>>('toggleBtn');
  private readonly panel = viewChild<ElementRef<HTMLElement>>('panel');

  protected readonly open = signal(false);
  protected readonly lesson = signal<DashboardLesson | null>(null);
  private readonly verbs = signal<ReadonlyMap<string, VerbProgress> | null>(null);

  protected readonly struggling = computed(() => {
    const v = this.verbs();
    return v ? strugglingVerbs(v, new Date()).length : 0;
  });

  protected readonly items = computed<Item[]>(() => {
    const v = this.verbs();
    const next = v ? nextLesson(VERB_LESSONS, v) : null;
    const struggling = this.struggling();
    return [
      { label: 'Ngữ pháp', icon: 'grammar', link: '/grammar', note: 'Giáo trình', badge: 0 },
      {
        label: 'Động từ',
        icon: 'table',
        link: '/irregular-verbs',
        note: !v
          ? 'Bất quy tắc'
          : next
            ? `Bài ${next.number}/${VERB_LESSONS.length}`
            : 'Đã học hết',
        badge: 0,
      },
      {
        label: 'Luyện nghe',
        icon: 'headphones',
        link: '/irregular-verbs/listen',
        note: 'V2 · V3',
        badge: 0,
      },
      {
        label: 'Từ hay sai',
        icon: 'flame',
        link: '/irregular-verbs/mistakes',
        note: struggling ? `${struggling} từ` : 'Chưa có',
        badge: struggling,
      },
      { label: 'Trò chuyện', icon: 'chat', link: '/vocabulary/talk', note: 'với AI', badge: 0 },
      { label: 'Tiến độ', icon: 'chart', link: '/stats', note: 'Thống kê', badge: 0 },
    ];
  });

  constructor() {
    this.loadVerbs();
    inject(Router)
      .events.pipe(
        filter((e) => e instanceof NavigationStart),
        takeUntilDestroyed(inject(DestroyRef)),
      )
      .subscribe(() => this.open.set(false));
  }

  protected toggle(): void {
    if (this.open()) {
      this.close(true);
      return;
    }
    this.open.set(true);
    this.loadVerbs();
    this.dashboardApi.dashboard().subscribe({
      next: (d) => this.lesson.set(d.lesson?.title ? d.lesson : null),
      error: () => this.lesson.set(null),
    });
    afterNextRender(() => this.panel()?.nativeElement.querySelector<HTMLElement>('a')?.focus(), {
      injector: this.injector,
    });
  }

  /** Closes the card; back to the button when the keyboard was in it. */
  close(refocus = false): void {
    if (!this.open()) {
      return;
    }
    const inside = this.panel()?.nativeElement.contains(this.document.activeElement) ?? false;
    this.open.set(false);
    if (refocus && inside) {
      this.toggleButton().nativeElement.focus();
    }
  }

  private loadVerbs(): void {
    this.verbsApi.progress().subscribe({
      next: (r) => this.verbs.set(new Map(r.verbs.map((v) => [v.base, v]))),
      error: () => undefined,
    });
  }
}
