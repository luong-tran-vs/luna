import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { map } from 'rxjs';

import { VerbProgress } from '../../../core/models/verbs';
import { VerbsApiService } from '../../../core/services/verbs-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { VerbChant } from '../verb-chant/verb-chant';
import { verbGroups } from '../verb-groups.logic';
import { VerbPictures } from '../verb-pictures.service';
import { learnedIn, lessonVerbs } from '../verb-study.logic';

/**
 * One group of the "Động từ" tab (design/de-xuat-dong-tu.png part 7, added 2026-11-10): its rule,
 * its lessons with their verbs (pictures when drawn) and how many are learned, and "Đọc theo nhịp"
 * over all its verbs.
 */
@Component({
  selector: 'lu-verb-group',
  imports: [NgOptimizedImage, RouterLink, Icon, VerbChant],
  templateUrl: './verb-group.html',
  styleUrl: './verb-group.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class VerbGroupPage {
  private readonly api = inject(VerbsApiService);
  protected readonly pictures = inject(VerbPictures).urls;

  private readonly key = toSignal(
    inject(ActivatedRoute).paramMap.pipe(map((p) => p.get('key') ?? '')),
    {
      initialValue: '',
    },
  );
  protected readonly group = computed(() => verbGroups().find((g) => g.key === this.key()) ?? null);
  protected readonly verbs = computed(() => {
    const g = this.group();
    return g ? g.lessons.flatMap((l) => lessonVerbs(l, IRREGULAR_VERBS)) : [];
  });
  protected readonly progress = signal<ReadonlyMap<string, VerbProgress>>(new Map());

  constructor() {
    inject(VerbPictures).load();
    this.api.progress().subscribe({
      next: (r) => this.progress.set(new Map(r.verbs.map((v) => [v.base, v]))),
      // Without it the lessons show no count.
      error: () => this.progress.set(new Map()),
    });
  }

  protected learned(lessonNumber: number): number {
    const l = this.group()?.lessons.find((x) => x.number === lessonNumber);
    return l ? learnedIn(l, this.progress()) : 0;
  }
}
