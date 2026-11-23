import { DOCUMENT } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  OnInit,
  signal,
  untracked,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { serverUrl } from '../../../core/api-url';
import { ApiError } from '../../../core/interceptors/error-interceptor';
import { AttemptView, PaperGroup, SaveProgress, ToeicPart } from '../../../core/models/toeic';
import { SpeechService } from '../../../core/services/speech.service';
import { ToeicApiService } from '../../../core/services/toeic-api.service';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import {
  clockLabel,
  ClockState,
  pauseClock,
  remaining,
  resumeClock,
  startClock,
} from '../../games/shared/game-clock.logic';
import { ListeningBar } from '../listening/listening-bar';
import { ListeningPlayer } from '../listening/listening-player';
import { buildScript, segmentStart } from '../listening/listening-script.logic';
import { DocView, HighlightEdit } from '../shared/doc-view';
import { GraphicView } from '../shared/graphic-view';
import {
  addHighlight,
  Highlight,
  HIGHLIGHT_PREFIX,
  parseHighlights,
  removeHighlight,
} from '../shared/highlight.logic';
import { QuestionView } from '../shared/question-view';
import { listeningPrompt, PART_NAMES } from '../shared/score.logic';
import { AnswerSheet } from './answer-sheet';
import {
  ActiveTime,
  activeMs,
  CHOICE_DELAY_MS,
  pauseActive,
  resumeActive,
  saveOnHide,
  TICK_SAVE_MS,
} from './autosave';
import {
  changes,
  ExamPage,
  ExamQuestion,
  ExamState,
  goto,
  pagesOf,
  questionsOf,
  select,
  step,
  toggleFlag,
  unanswered,
} from './exam-state.logic';
import { QuestionGrid } from './question-grid';

/** The clock reads aloud at these minutes left (not every second). */
const ANNOUNCE_AT = [10, 5, 1];

/**
 * The exam room (FR-007…FR-012), one for every mode: the title and Thoát, the part tabs, the page
 * (a Part 6/7 text beside its questions on wide screens, stacked on phones; Part 5 a few questions
 * a page), the clock, Nộp bài and the question grid. Answers are never graded here; the progress is
 * saved on the server (after a choice, every 5 seconds, when the page is hidden) and queued on this
 * device without network. The time stops while the learner is away.
 */
@Component({
  selector: 'lu-toeic-exam-room',
  imports: [DocView, QuestionView, QuestionGrid, AnswerSheet, ConfirmDialog, ListeningBar, GraphicView],
  templateUrl: './exam-room.html',
  styleUrl: './exam-room.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:visibilitychange)': 'onVisibility()' },
})
export class ExamRoom implements OnInit {
  private readonly route = inject(ActivatedRoute);
  protected readonly id = signal(this.route.snapshot.paramMap.get('id') ?? '');

  private readonly api = inject(ToeicApiService);
  private readonly router = inject(Router);
  private readonly document = inject(DOCUMENT);
  private readonly storage = this.document.defaultView?.localStorage;

  protected readonly view = signal<AttemptView | null>(null);
  protected readonly loadError = signal('');
  protected readonly state = signal<ExamState>({ answers: {}, flags: [], position: '' });
  /** What the server has. */
  private sent: ExamState = { answers: {}, flags: [], position: '' };
  private clock: ClockState | null = null;
  private active: ActiveTime = { before: 0, since: null };
  protected readonly leftMs = signal<number | null>(null);
  protected readonly announce = signal('');
  protected readonly saveNote = signal('');
  protected readonly sheetOpen = signal(false);
  protected readonly confirming = signal(false);
  protected readonly submitting = signal(false);
  protected readonly submitError = signal('');
  protected readonly highlights = signal<Highlight[]>([]);
  /** The audio of the listening parts; null when the paper has none. */
  protected readonly player = signal<ListeningPlayer | null>(null);
  private readonly speech = inject(SpeechService);
  /** The listening segment the server has (where a resumed attempt starts). */
  private sentSegment = 0;

  private choiceTimer: ReturnType<typeof setTimeout> | null = null;
  private tick: ReturnType<typeof setInterval> | null = null;
  private lastTickSave = 0;
  private announced = new Set<number>();
  private done = false;

  protected readonly pages = computed<ExamPage[]>(() => pagesOf(this.view()?.paper ?? []));
  protected readonly questions = computed<ExamQuestion[]>(() =>
    questionsOf(this.pages(), this.view()?.paper ?? []),
  );
  protected readonly current = computed<ExamQuestion | null>(() => {
    const qs = this.questions();
    return qs.find((q) => q.id === this.state().position) ?? qs[0] ?? null;
  });
  protected readonly pageIndex = computed(() => this.current()?.page ?? 0);
  protected readonly page = computed(() => this.pages()[this.pageIndex()] ?? null);
  protected readonly pageGroups = computed<PaperGroup[]>(() => {
    const paper = this.view()?.paper ?? [];
    return (this.page()?.groups ?? []).map((gi) => paper[gi]);
  });
  /** A page with a text shows it beside its questions. */
  protected readonly withDocs = computed(() => this.pageGroups().some((g) => (g.docs?.length ?? 0) > 0));
  protected readonly partTabs = computed(() => {
    const qs = this.questions();
    const cur = this.current()?.part;
    const parts: ToeicPart[] = [];
    for (const q of qs) {
      if (!parts.includes(q.part)) {
        parts.push(q.part);
      }
    }
    return parts.map((p) => ({
      part: p,
      name: PART_NAMES[p],
      first: qs.find((q) => q.part === p)?.id ?? '',
      on: p === cur,
    }));
  });
  protected readonly unansweredCount = computed(() => unanswered(this.state(), this.questions()).length);
  protected readonly answeredCount = computed(
    () => this.questions().length - this.unansweredCount(),
  );
  protected readonly clockText = computed(() => {
    const ms = this.leftMs();
    return ms === null ? 'Không giới hạn' : clockLabel(ms);
  });
  protected readonly lowTime = computed(() => {
    const ms = this.leftMs();
    return ms !== null && ms <= 5 * 60_000;
  });
  protected readonly confirmMessage = computed(
    () =>
      `Còn ${this.unansweredCount()} câu chưa làm. Nộp bài thì các câu này tính là bỏ qua. Bạn có muốn quay lại làm tiếp?`,
  );

  /** The group of each question. */
  private readonly groupOf = computed(() => {
    const out = new Map<string, PaperGroup>();
    for (const g of this.view()?.paper ?? []) {
      for (const q of g.questions) {
        out.set(q.id, g);
      }
    }
    return out;
  });
  /** "41–43" for each group key, for the audio bar. */
  protected readonly groupNumbers = computed(() => {
    const out: Record<string, string> = {};
    for (const g of this.view()?.paper ?? []) {
      const first = g.questions[0]?.number;
      const last = g.questions.at(-1)?.number;
      out[g.key] = first === last ? `${first}` : `${first}–${last}`;
    }
    return out;
  });
  protected readonly practice = computed(() => this.view()?.mode !== 'test');

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stop());
    // The page follows the audio: when a new group starts, its first question is shown.
    effect(() => {
      const pl = this.player();
      const key = pl?.group();
      if (pl && key && pl.playing()) {
        untracked(() => this.followAudio(key));
      }
    });
  }

  private followAudio(key: string): void {
    const cur = this.state().position;
    if (this.groupOf().get(cur)?.key === key) {
      return;
    }
    const g = this.view()?.paper.find((p) => p.key === key);
    const first = g?.questions[0]?.id;
    if (first) {
      this.state.update((s) => goto(s, first));
      this.document.defaultView?.scrollTo({ top: 0 });
    }
  }

  ngOnInit(): void {
    this.load();
  }

  private load(): void {
    const id = this.id();
    this.api.flush(id).subscribe({
      error: () => undefined,
      complete: () => this.open(id),
    });
  }

  private open(id: string): void {
    this.api.attempt(id).subscribe({
      next: (v) => {
        if (v.status === 'submitted') {
          void this.router.navigate(['/toeic/attempts', id, 'result'], { replaceUrl: true });
          return;
        }
        this.view.set(v);
        const st: ExamState = {
          answers: { ...v.answers },
          flags: [...v.flags],
          position: v.position || v.paper[0]?.questions[0]?.id || '',
        };
        this.sent = st;
        // Answers queued on this device but not sent yet go on top.
        const pending = this.api.pending(id);
        if (pending?.answers) {
          const answers = { ...st.answers };
          for (const [q, c] of Object.entries(pending.answers)) {
            if (c < 0) {
              delete answers[q];
            } else {
              answers[q] = c;
            }
          }
          st.answers = answers;
        }
        this.state.set(st);
        this.highlights.set(this.readHighlights(id));
        const script = buildScript(v.paper);
        if (script.length > 0) {
          const pl = new ListeningPlayer(script, this.speech);
          if (v.listeningSegment > 0) {
            pl.seek(v.listeningSegment);
          }
          this.sentSegment = v.listeningSegment;
          this.player.set(pl);
        }
        const now = performance.now();
        this.active = { before: v.elapsedMs, since: now };
        if (v.timeLimitMs > 0) {
          const left = Math.min(v.remainingMs, pending?.remainingMs ?? v.remainingMs);
          this.clock = startClock(left, now);
          this.leftMs.set(left);
        }
        this.lastTickSave = now;
        this.tick = setInterval(() => this.onTick(), 1000);
      },
      error: (err: unknown) =>
        this.loadError.set(
          err instanceof ApiError && err.status === 404
            ? 'Không tìm thấy bài làm này.'
            : 'Không tải được bài làm. Kiểm tra mạng rồi thử lại.',
        ),
    });
  }

  private onTick(): void {
    const now = performance.now();
    if (this.clock) {
      const left = remaining(this.clock, now);
      this.leftMs.set(left);
      const min = Math.ceil(left / 60_000);
      if (ANNOUNCE_AT.includes(min) && !this.announced.has(min) && left > 0) {
        this.announced.add(min);
        this.announce.set(`Còn ${min} phút`);
      }
      if (left <= 0) {
        this.announce.set('Hết giờ, bài được nộp');
        this.submit(true);
        return;
      }
    }
    if (now - this.lastTickSave >= TICK_SAVE_MS) {
      this.lastTickSave = now;
      this.save();
    }
  }

  /** The progress to send: what changed, with the time. */
  private progress(): SaveProgress {
    const now = performance.now();
    const p = changes(this.sent, this.state());
    if (this.clock) {
      p.remainingMs = Math.round(remaining(this.clock, now));
    }
    p.elapsedMs = Math.round(activeMs(this.active, now));
    const seg = this.listeningSegment();
    if (seg !== null && seg !== this.sentSegment) {
      p.listeningSegment = seg;
    }
    return p;
  }

  /** Where a resumed attempt starts the audio: the start of the group playing; null without audio. */
  private listeningSegment(): number | null {
    const pl = this.player();
    if (!pl || pl.index() < 0) {
      return null;
    }
    return segmentStart(pl.script, pl.index());
  }

  private save(): void {
    if (this.done) {
      return;
    }
    const id = this.id();
    const snapshot = this.state();
    const p = this.progress();
    this.api.save(id, p).subscribe({
      next: (ok) => {
        this.sent = snapshot;
        if (p.listeningSegment !== undefined) {
          this.sentSegment = p.listeningSegment;
        }
        this.saveNote.set(ok ? '' : 'Mất mạng: tiến độ được giữ trên máy, sẽ gửi khi có mạng.');
      },
      error: (err: unknown) => {
        if (err instanceof ApiError && err.status === 409) {
          this.stop();
          void this.router.navigate(['/toeic/attempts', id, 'result'], { replaceUrl: true });
        }
      },
    });
  }

  private saveSoon(): void {
    if (this.choiceTimer) {
      clearTimeout(this.choiceTimer);
    }
    this.choiceTimer = setTimeout(() => {
      this.choiceTimer = null;
      this.save();
    }, CHOICE_DELAY_MS);
  }

  protected choose(id: string, choice: number): void {
    this.state.update((s) => select(s, id, choice));
    this.saveSoon();
  }

  protected flag(id: string): void {
    this.state.update((s) => toggleFlag(s, id));
    this.saveSoon();
  }

  protected go(id: string): void {
    this.state.update((s) => goto(s, id));
    this.sheetOpen.set(false);
    this.document.defaultView?.scrollTo({ top: 0 });
    // While practising, going to a listening question moves the audio to the start of its group.
    const g = this.groupOf().get(id);
    if (g && g.part <= 4 && this.practice()) {
      this.player()?.jumpToGroup(g.key);
    }
  }

  protected move(dir: 1 | -1): void {
    this.state.update((s) => step(s, this.questions(), this.pages(), dir));
    this.document.defaultView?.scrollTo({ top: 0 });
  }

  protected onVisibility(): void {
    const now = performance.now();
    if (this.document.visibilityState === 'hidden') {
      if (!this.done && this.view()) {
        const p = this.progress();
        saveOnHide(this.id(), p);
        this.sent = this.state();
      }
      if (this.clock) {
        this.clock = pauseClock(this.clock, now);
      }
      this.active = pauseActive(this.active, now);
    } else {
      if (this.clock) {
        this.clock = resumeClock(this.clock, now);
      }
      this.active = resumeActive(this.active, now);
    }
  }

  protected askSubmit(): void {
    if (this.unansweredCount() > 0) {
      this.confirming.set(true);
    } else {
      this.submit(false);
    }
  }

  /** Submits; a failed send keeps everything and says to try again. */
  protected submit(auto: boolean): void {
    if (this.submitting() || this.done) {
      return;
    }
    this.confirming.set(false);
    this.submitting.set(true);
    this.submitError.set('');
    const id = this.id();
    const last: SaveProgress = { ...this.progress(), answers: { ...this.state().answers }, flags: [...this.state().flags] };
    if (this.clock) {
      this.clock = pauseClock(this.clock, performance.now());
    }
    this.api.submit(id, last).subscribe({
      next: () => {
        this.stop();
        void this.router.navigate(['/toeic/attempts', id, 'result'], { replaceUrl: true });
      },
      error: () => {
        this.submitting.set(false);
        this.submitError.set(
          auto
            ? 'Hết giờ nhưng chưa gửi được bài. Bài của bạn vẫn còn, bấm Nộp bài để thử lại.'
            : 'Chưa gửi được, thử lại. Bài của bạn vẫn còn nguyên.',
        );
        if (this.clock && !auto) {
          this.clock = resumeClock(this.clock, performance.now());
        }
      },
    });
  }

  /** Saves, then leaves to the test (or the library). */
  protected exit(): void {
    const v = this.view();
    const back = v?.source.kind === 'test' && v.source.testKey ? ['/toeic/tests', v.source.testKey] : ['/toeic'];
    if (!v || this.done) {
      void this.router.navigate(back);
      return;
    }
    this.api.save(this.id(), this.progress()).subscribe({
      next: () => void this.router.navigate(back),
      error: () => void this.router.navigate(back),
    });
    this.stop();
  }

  private stop(): void {
    this.done = true;
    this.player()?.destroy();
    if (this.tick) {
      clearInterval(this.tick);
      this.tick = null;
    }
    if (this.choiceTimer) {
      clearTimeout(this.choiceTimer);
      this.choiceTimer = null;
    }
  }

  // --- what the page shows ---

  protected chosenOf(id: string): number | null {
    return this.state().answers[id] ?? null;
  }

  protected flagged(id: string): boolean {
    return this.state().flags.includes(id);
  }

  protected blankNumbers(g: PaperGroup): Record<number, number> {
    const out: Record<number, number> = {};
    for (const q of g.questions) {
      out[q.no] = q.number;
    }
    return out;
  }

  protected activeBlank(g: PaperGroup): number | null {
    return g.questions.find((q) => q.id === this.state().position)?.no ?? null;
  }

  /** Part 1/2 are only heard: the question shows no words, only the choice letters. */
  protected shown(g: PaperGroup, q: PaperGroup['questions'][number]): PaperGroup['questions'][number] {
    const prompt = listeningPrompt(g.part);
    return prompt ? { ...q, prompt } : q;
  }

  protected image(url: string | null): string | null {
    return url ? serverUrl(url) : null;
  }

  // --- highlights (this device only) ---

  protected editHighlight(e: HighlightEdit): void {
    this.highlights.update((list) =>
      e.remove ? removeHighlight(list, e.highlight) : addHighlight(list, e.highlight),
    );
    try {
      this.storage?.setItem(HIGHLIGHT_PREFIX + this.id(), JSON.stringify(this.highlights()));
    } catch {
      // Storage off: the highlights last while the page is open.
    }
  }

  private readHighlights(id: string): Highlight[] {
    try {
      return parseHighlights(this.storage?.getItem(HIGHLIGHT_PREFIX + id) ?? null);
    } catch {
      return [];
    }
  }
}

