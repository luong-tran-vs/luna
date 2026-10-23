import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { SummarySkill, SummarySkillScore, TopicRating, TopicSummary } from '../../core/models/study';
import { formatScore, MISTAKE_LABELS, MistakeType } from '../../core/models/writing';
import { StudyApiService } from '../../core/services/study-api.service';
import { Icon } from '../../shared/components/icon/icon';
import { Loading } from '../../shared/components/loading/loading';
import { ProgressRing } from '../../shared/components/progress-ring/progress-ring';

/** A skill at or above this is a strength, below WEAK it needs work. */
const STRONG = 80;
const WEAK = 60;

export const RATING_TEXT: Record<TopicRating, { title: string; line: string }> = {
  excellent: { title: 'Xuất sắc', line: 'Bạn nắm rất chắc chủ đề này.' },
  good: { title: 'Tốt', line: 'Bạn đã nắm được phần lớn chủ đề.' },
  fair: { title: 'Khá', line: 'Bạn đã hiểu chủ đề, nên ôn lại vài phần bên dưới.' },
  practice: { title: 'Cần ôn thêm', line: 'Hãy ôn lại các phần bên dưới trước khi đi tiếp.' },
  none: { title: 'Chưa có điểm', line: 'Làm các bước Nghe, Đọc và Viết của bài để có đánh giá.' },
};

const SKILL_LABELS: Record<SummarySkill, string> = {
  listening: 'Nghe',
  reading: 'Đọc hiểu',
  writing: 'Viết',
  review: 'Ôn từ',
};

/** What to do about a weak skill, and where. */
const SKILL_ADVICE: Record<SummarySkill, { text: string; link: string | null }> = {
  listening: { text: 'Nghe chưa chắc: mở lại các bài và chép chính tả thêm một lượt.', link: null },
  reading: { text: 'Đọc hiểu chưa chắc: đọc lại bài và làm lại câu hỏi hiểu bài.', link: null },
  writing: { text: 'Bài viết còn nhiều lỗi: xem lại phần sửa và viết lại.', link: '/writings' },
  review: { text: 'Còn quên từ khi ôn theo bài: ôn lại từ của chủ đề.', link: '/vocabulary/review' },
};

const dateFormat = new Intl.DateTimeFormat('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });

export interface SkillRow {
  skill: SummarySkill;
  label: string;
  percent: number;
  detail: string;
  /** 'strong', 'weak' or '' in between. */
  mark: 'strong' | 'weak' | '';
}

export interface Advice {
  text: string;
  link: string | null;
}

export function skillRow(s: SummarySkillScore, writingAverage: number): SkillRow {
  const detail =
    s.skill === 'writing'
      ? `Điểm trung bình ${formatScore(writingAverage)}/5 · ${s.correct} bài`
      : s.skill === 'listening'
        ? `Chép đúng ${s.correct}/${s.total} từ`
        : `Đúng ${s.correct}/${s.total} câu`;
  const mark = s.percent >= STRONG ? 'strong' : s.percent < WEAK ? 'weak' : '';
  return { skill: s.skill, label: SKILL_LABELS[s.skill], percent: s.percent, detail, mark };
}

export function mistakeLabel(type: string): string {
  return MISTAKE_LABELS[type as MistakeType] ?? type;
}

/** What to study next, from the weak skills, the words not remembered and the writing mistakes. */
export function advice(s: TopicSummary): Advice[] {
  const out: Advice[] = [];
  const left = s.lessons.length - s.completed;
  if (left > 0) {
    out.push({ text: `Còn ${left} bài của chủ đề chưa học.`, link: null });
  }
  for (const k of s.skills) {
    if (k.percent < WEAK) {
      out.push(SKILL_ADVICE[k.skill]);
    }
  }
  const unsure = s.words - s.remembered;
  if (unsure > 0) {
    out.push({ text: `Còn ${unsure} từ của chủ đề chưa nhớ chắc: ôn trong Sổ từ.`, link: '/vocabulary/review' });
  }
  const top = s.mistakes[0];
  if (top) {
    out.push({ text: `Lỗi viết hay gặp nhất: ${mistakeLabel(top.type)} (${top.count} lần).`, link: '/writings' });
  }
  return out;
}

/**
 * The topic summary (added 2026-10-23): shown when the learner finishes a topic's last lesson and
 * from the learning path. It rates the topic from the scores the server keeps per lesson
 * (listening, comprehension questions, writings, lesson reviews), says what went well and what to
 * study again, with the topic's can-do and grammar from the course.
 */
@Component({
  selector: 'lu-topic-summary',
  imports: [RouterLink, Icon, Loading, ProgressRing],
  templateUrl: './topic-summary.html',
  styleUrl: './topic-summary.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TopicSummaryPage {
  private readonly api = inject(StudyApiService);
  private readonly route = inject(ActivatedRoute);

  protected readonly summary = signal<TopicSummary | null>(null);
  protected readonly loadError = signal(false);

  protected readonly finished = computed(() => {
    const s = this.summary();
    return !!s && s.lessons.length > 0 && s.completed === s.lessons.length;
  });
  protected readonly rating = computed(() => RATING_TEXT[this.summary()?.rating ?? 'none']);
  protected readonly skills = computed(() => {
    const s = this.summary();
    return s ? s.skills.map((k) => skillRow(k, s.writingAverage)) : [];
  });
  protected readonly advice = computed(() => {
    const s = this.summary();
    return s ? advice(s) : [];
  });
  /** "12/10/2026 – 20/10/2026", or one day. */
  protected readonly span = computed(() => {
    const s = this.summary();
    if (!s?.startedAt || !s.finishedAt) {
      return '';
    }
    const from = dateFormat.format(new Date(s.startedAt));
    const to = dateFormat.format(new Date(s.finishedAt));
    return from === to ? from : `${from} – ${to}`;
  });

  protected readonly mistakeLabel = mistakeLabel;

  constructor() {
    this.load();
  }

  protected load(): void {
    const id = this.route.snapshot.paramMap.get('id') ?? '';
    const level = this.route.snapshot.queryParamMap.get('level') ?? '';
    this.loadError.set(false);
    this.api.topicSummary(id, level).subscribe({
      next: (s) => this.summary.set(s),
      error: () => this.loadError.set(true),
    });
  }
}
