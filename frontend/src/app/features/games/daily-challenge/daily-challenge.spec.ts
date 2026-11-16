import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { DailyQuestion, DailyResultBody, DailyView } from '../../../core/models/games';
import { DailyChallenge } from './daily-challenge';
import { letterTiles } from './daily.logic';

const picture = (lemma: string): DailyQuestion => ({
  kind: 'picture',
  picture: { lemma, imageUrl: `/api/words/${lemma}/image`, meaningVi: 'nghĩa', options: [lemma, 'b', 'c', 'd'], answer: 0 },
});

const questions: DailyQuestion[] = [
  picture('apple'),
  { kind: 'verb', verb: { base: 'go' } },
  { kind: 'letters', letters: { lemma: 'cat', meaningVi: 'con mèo', imageUrl: '' } },
  {
    kind: 'grammar',
    grammar: {
      tokens: ['She', 'go', 'home', '.'],
      wrong: { from: 1, to: 2 },
      options: ['going', 'goes', 'gone'],
      answer: 1,
      ruleVi: 'Ngôi thứ ba số ít thêm -s.',
      topicVi: '',
      pointId: 'p',
      source: 'choice',
    },
  },
  ...['b1', 'b2', 'b3', 'b4', 'b5', 'b6'].map(picture),
];

const view = (over: Partial<DailyView> = {}): DailyView => ({
  day: '2026-11-13',
  seed: 99,
  done: false,
  streak: { current: 12, longest: 20 },
  week: [true, true, true, true, false, false, false],
  result: null,
  compare: { yesterday: 880, avg7: 845.5, best: 980 },
  questions,
  ...over,
});

describe('DailyChallenge', () => {
  let fixture: ComponentFixture<DailyChallenge>;
  let controller: HttpTestingController;
  let el: HTMLElement;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => text(b) === label) as HTMLButtonElement | undefined;
  const stable = async () => {
    fixture.detectChanges();
    await fixture.whenStable();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DailyChallenge],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(DailyChallenge);
    el = fixture.nativeElement as HTMLElement;
  });

  afterEach(() => {
    fixture.destroy();
    controller.verify();
  });

  it('shows the day, the streak and the week, then plays ten questions and sends the result', async () => {
    controller.expectOne('/games/daily').flush(view());
    await stable();
    expect(text(el)).toContain('Thứ Sáu, 13/11');
    expect(text(el)).toContain('🔥 12 ngày');
    expect(el.querySelectorAll('.day-mark.is-played')).toHaveLength(4);
    expect(text(el.querySelector('.day-mark.is-today'))).toBe('?');

    button('Bắt đầu')!.click();
    await stable();
    expect(el.querySelectorAll('.dot')).toHaveLength(10);

    // 1. picture: right.
    button('apple')!.click();
    await stable();
    expect(el.querySelector('.dot.is-right')).not.toBeNull();
    button('Câu tiếp')!.click();
    await stable();

    // 2. verb: V2 right, V3 wrong.
    const [v2, v3] = Array.from(el.querySelectorAll<HTMLInputElement>('.verb-form input'));
    v2.value = 'went';
    v2.dispatchEvent(new Event('input'));
    v3.value = 'goed';
    v3.dispatchEvent(new Event('input'));
    button('Kiểm tra')!.click();
    await stable();
    expect(el.querySelector('.dot.is-wrong')).not.toBeNull();
    button('Câu tiếp')!.click();
    await stable();

    // 3. letters: tap c, a, t.
    const tiles = letterTiles('cat', 99, 2);
    for (const ch of ['c', 'a', 't']) {
      const t = tiles.find((x) => x.ch === ch)!;
      (el.querySelectorAll<HTMLButtonElement>('.tile')[tiles.indexOf(t)]).click();
      await stable();
    }
    expect(text(el.querySelector('.verdict'))).toContain('Đúng rồi!');
    button('Câu tiếp')!.click();
    await stable();

    // 4. grammar: tap "go", then "goes".
    button('go')!.click();
    await stable();
    button('goes')!.click();
    await stable();
    expect(text(el)).toContain('She goes home.');
    button('Câu tiếp')!.click();
    await stable();

    // 5–10. pictures: the first wrong, the rest right.
    for (let i = 0; i < 6; i++) {
      button(i === 0 ? 'b' : `b${i + 1}`)!.click();
      await stable();
      button(i === 5 ? 'Xem kết quả' : 'Câu tiếp')!.click();
      await stable();
    }

    const post = controller.expectOne('/games/daily/result');
    const body = post.request.body as DailyResultBody;
    expect(body.day).toBe('2026-11-13');
    expect(body.answers.map((a) => a.right)).toEqual([true, false, true, true, false, true, true, true, true, true]);
    expect(body.answers[1].kind).toBe('verb');
    expect(body.wrongWords).toEqual(['b1']);
    expect(body.wrongVerbs).toEqual([{ base: 'go', form: 'participle', typed: 'goed' }]);
    expect(body.score).toBeGreaterThanOrEqual(800);
    expect(body.score).toBeLessThanOrEqual(2400);
    post.flush({ streak: { current: 13, longest: 20 }, compare: { yesterday: 880, avg7: null, best: 980 }, rescheduled: 2 });
    await stable();

    expect(el.querySelectorAll('.square')).toHaveLength(10);
    expect(text(el)).toContain('Chuỗi 13 ngày');
    expect(text(el.querySelector('.compare'))).toContain('—');
    expect(text(el)).toContain('Câu sai đã thêm vào lịch ôn.');
    expect(button('Chơi lại để luyện')).toBeDefined();
  });

  it('shows the counted result when today is done, and a replay only sends the practice misses', async () => {
    const answers = questions.map((q, i) => ({ kind: q.kind, right: i !== 3 }));
    controller.expectOne('/games/daily').flush(view({ done: true, result: { score: 2100, answers, finishedAt: '' } }));
    await stable();
    expect(text(el)).toContain('2.100');
    expect(el.querySelectorAll('.square.is-wrong')).toHaveLength(1);

    button('Chơi lại để luyện')!.click();
    await stable();
    button('b')!.click();
    await stable();
    expect(text(el.querySelector('.verdict'))).toContain('apple');
  });

  it('copies the share text, or shows it to copy by hand', async () => {
    const answers = questions.map((q, i) => ({ kind: q.kind, right: i !== 3 }));
    controller.expectOne('/games/daily').flush(view({ done: true, result: { score: 2100, answers, finishedAt: '' } }));
    await stable();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
    button('↗ Chia sẻ kết quả')!.click();
    await stable();
    await stable();
    expect(writeText).toHaveBeenCalledWith('Luna 13/11 · 9/10 🔥12\n🟩🟩🟩🟥🟩🟩🟩🟩🟩🟩');
    expect(text(el)).toContain('Đã sao chép');

    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true });
    button('↗ Chia sẻ kết quả')!.click();
    await stable();
    await stable();
    expect(el.querySelector<HTMLTextAreaElement>('textarea')?.value).toContain('Luna 13/11');
  });
});
