import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Level, LEVEL_NAMES, LEVELS } from '../../../core/models/lesson';
import { LearningPath, Milestone } from '../../../core/models/study';
import { StudyApiService } from '../../../core/services/study-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';

/** The name and one-line summary of each part, shown on its card. */
const PART_INFO: Record<Level, { name: string; summary: string }> = {
  A1: { name: 'Khởi Động', summary: 'Làm quen với các câu giao tiếp cơ bản hằng ngày' },
  A2: { name: 'Giao Tiếp', summary: 'Hội thoại thực tế & đời sống' },
  B1: { name: 'Tự Tin Làm Chủ', summary: 'Phản xạ nhanh & từ vựng nâng cao' },
  B2: { name: 'Bày Tỏ Quan Điểm', summary: 'Tranh luận và trình bày ý kiến rõ ràng' },
  C1: { name: 'Lưu Loát', summary: 'Diễn đạt tự nhiên trong mọi tình huống' },
  C2: { name: 'Thành Thạo', summary: 'Hiểu gần như mọi thứ bạn nghe và đọc' },
};

/** What a learner can say at the end of each part, shown in the card's speech bubble. */
const SAYINGS: Record<Level, string> = {
  A1: 'Hello!',
  A2: "I'm starting to learn English!",
  B1: 'I know some words in English!',
  B2: 'I can share my opinions clearly.',
  C1: 'I can express myself fluently.',
  C2: 'I understand almost everything.',
};

interface PartView {
  milestone: Milestone;
  /** 1-based number of the part, in level order. */
  number: number;
  levelName: string;
  name: string;
  summary: string;
  saying: string;
  percent: number;
  /** Lessons still to finish in the part. */
  left: number;
}

/**
 * The "Học" tab (design/danh-sach-cac-phan.png): one card per part (level), with its progress.
 * The part being studied says "Tiếp tục học", the others "Nhảy tới phần N"; a part with no lesson
 * yet has nothing to open. A card opens the part's path (LearnPath).
 */
@Component({
  selector: 'lu-learn-parts',
  imports: [Loading, Icon, RouterLink, NgOptimizedImage],
  templateUrl: './learn-parts.html',
  styleUrl: './learn-parts.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LearnParts {
  protected readonly data = signal<LearningPath | null>(null);
  protected readonly error = signal(false);

  protected readonly parts = computed<PartView[]>(() =>
    (this.data()?.milestones ?? []).map((m) => ({
      milestone: m,
      number: LEVELS.indexOf(m.level) + 1,
      levelName: LEVEL_NAMES[m.level],
      ...PART_INFO[m.level],
      saying: SAYINGS[m.level],
      percent: m.total ? Math.round((m.completed / m.total) * 100) : 0,
      left: m.total - m.completed,
    })),
  );

  constructor() {
    inject(StudyApiService)
      .path()
      .subscribe({ next: (p) => this.data.set(p), error: () => this.error.set(true) });
  }
}
