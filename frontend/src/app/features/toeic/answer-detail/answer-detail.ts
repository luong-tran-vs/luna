import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { serverUrl } from '../../../core/api-url';
import { QuestionDetail, VocabItem } from '../../../core/models/toeic';
import { SpeechService } from '../../../core/services/speech.service';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { ListeningPlayer } from '../listening/listening-player';
import { replayScript } from '../listening/listening-script.logic';
import { DocView } from '../shared/doc-view';
import { GraphicView } from '../shared/graphic-view';
import { QuestionView } from '../shared/question-view';
import { letter, outcomeLabel, PART_NAMES } from '../shared/score.logic';
import { ToeicBack } from '../shared/toeic-back';

/**
 * The answer detail of a question (FR-028): the part and question types (a grammar type links to its
 * grammar point), "you chose X · answer Y", the text with the evidence marked and its translation,
 * the Vietnamese explanation, and the words worth noting with "+ Sổ từ".
 */
@Component({
  selector: 'lu-toeic-answer-detail',
  imports: [RouterLink, DocView, QuestionView, GraphicView, ToeicBack],
  templateUrl: './answer-detail.html',
  styleUrls: ['../shared/toeic-page.css', './answer-detail.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AnswerDetail {
  private readonly api = inject(ToeicApiService);
  private readonly route = inject(ActivatedRoute);

  protected readonly id = signal('');
  protected readonly data = signal<QuestionDetail | null>(null);
  protected readonly error = signal('');
  protected readonly translated = signal(false);
  /** The words added to the notebook ("" while sending, else the state shown). */
  protected readonly saved = signal<Record<string, 'saving' | 'done' | 'error'>>({});
  protected readonly names = PART_NAMES;

  protected readonly summary = computed(() => {
    const q = this.data()?.question;
    if (!q) {
      return '';
    }
    const chosen = q.chosen >= 0 ? `Bạn chọn ${letter(q.chosen)}` : 'Bạn bỏ qua';
    return `${chosen} · Đáp án ${letter(q.answer)} · ${outcomeLabel(q.state)}`;
  });
  /** The shown number of every blank of a Part 6 text: the group's questions follow each other. */
  protected readonly blankNumbers = computed<Record<number, number>>(() => {
    const q = this.data()?.question;
    const out: Record<number, number> = {};
    if (q) {
      for (let n = 1; n <= 4; n++) {
        out[n] = q.number - q.no + n;
      }
    }
    return out;
  });
  protected readonly image = computed(() => {
    const u = this.data()?.group.imageUrl;
    return u ? serverUrl(u) : null;
  });

  private readonly speech = inject(SpeechService);
  /** Plays the group again (listening parts); null for a reading question. */
  protected readonly replay = computed(() => {
    const d = this.data();
    if (!d || d.group.part > 4) {
      return null;
    }
    const q = d.question;
    const script = replayScript({
      key: d.group.key,
      part: d.group.part,
      number: q.number,
      imageUrl: null,
      graphic: d.group.graphic,
      script: d.group.script,
      questions: [{ id: q.id, no: q.no, number: q.number, prompt: q.prompt, options: q.options }],
    });
    return script.length ? new ListeningPlayer(script, this.speech) : null;
  });
  /** The turns of the script that give the answer away. */
  protected readonly evidenceTurns = computed(
    () => new Set((this.data()?.question.evidence ?? []).filter((e) => e.doc === -1).map((e) => e.turn)),
  );

  constructor() {
    inject(DestroyRef).onDestroy(() => this.replay()?.destroy());
    this.route.paramMap.pipe(takeUntilDestroyed()).subscribe((p) => {
      this.id.set(p.get('id') ?? '');
      this.load(p.get('qid') ?? '');
    });
  }

  private load(qid: string): void {
    this.replay()?.destroy();
    this.data.set(null);
    this.error.set('');
    this.translated.set(false);
    this.api.question(this.id(), qid).subscribe({
      next: (d) => this.data.set(d),
      error: () => this.error.set('Không tải được đáp án của câu này.'),
    });
  }

  protected speakerLabel(s: string): string {
    const n = s.replace(/D/g, '');
    const who = s.startsWith('w') ? 'Nữ' : s.startsWith('m') ? 'Nam' : s;
    return n ? `${who} ${n}` : who;
  }

  protected accentLabel(a: string): string {
    return ({ us: 'giọng Mỹ', gb: 'giọng Anh', au: 'giọng Úc', ca: 'giọng Canada' } as Record<string, string>)[a] ?? a;
  }

  protected addWord(v: VocabItem): void {
    const sentence = this.data()?.question.prompt?.replace('___', '_____') ?? '';
    this.saved.update((s) => ({ ...s, [v.word]: 'saving' }));
    this.api.saveWord({ word: v.word, ipa: v.ipa, meaningVi: v.meaningVi, sentence }).subscribe({
      next: () => this.saved.update((s) => ({ ...s, [v.word]: 'done' })),
      error: () => this.saved.update((s) => ({ ...s, [v.word]: 'error' })),
    });
  }
}
