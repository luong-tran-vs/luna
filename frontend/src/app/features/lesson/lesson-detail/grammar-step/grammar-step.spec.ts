import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { GrammarNote } from '../../../../core/models/lesson';
import { GrammarStepContent } from '../../../../core/models/practice';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { GrammarStep, markHighlights } from './grammar-step';

const content: GrammarStepContent = {
  titleVi: 'Hiện tại đơn',
  noticing: [
    { sentence: 'The singer has a sweet voice.', highlights: ['has'] },
    { sentence: 'She Plays the piano and sings.', highlights: ['plays', 'sings'] },
  ],
  questionVi: 'Vì sao có chữ s ở cuối plays?',
  options: ['"plays" là thì quá khứ', 'Chủ ngữ là she / he / it', 'Động từ đã thì thêm s'],
  answerIndex: 1,
  explanationVi: 'Với he/she/it, động từ thêm s.',
  whenVi: 'Dùng để nói về sở thích, thói quen.',
  table: [
    { subject: 'I / you / we / they', form: 'like', negative: "don't like" },
    { subject: 'he / she / it', form: 'likes', negative: "doesn't like" },
    { subject: 'he / she / it', form: 'has', negative: '—' },
  ],
  mistakes: [{ wrong: 'She play the piano.', right: 'She plays the piano.' }],
  noteVi: 'Tiếng Việt không đổi động từ.',
  exercises: [
    {
      id: 'e1',
      kind: 'choice',
      text: 'My brother ___ the guitar.',
      options: ['play', 'plays'],
      answerIndex: 1,
      explanationVi: 'brother = he.',
    },
    {
      id: 'e2',
      kind: 'fill',
      text: 'She ___ music.',
      answers: ['likes'],
      explanationVi: 'she → likes.',
    },
  ],
};

const note: GrammarNote = {
  title: 'Present simple',
  bodyVi: 'Thêm s cho he/she/it.',
  examples: ['She likes music.'],
};

describe('markHighlights', () => {
  it('marks every highlight ignoring case, as whole words', () => {
    expect(markHighlights('She Plays and plays.', ['plays'])).toEqual([
      { text: 'She ', mark: false },
      { text: 'Plays', mark: true },
      { text: ' and ', mark: false },
      { text: 'plays', mark: true },
      { text: '.', mark: false },
    ]);
    expect(markHighlights('The hash has it.', ['has'])).toEqual([
      { text: 'The hash ', mark: false },
      { text: 'has', mark: true },
      { text: ' it.', mark: false },
    ]);
  });

  it('prefers the longer highlight and keeps a sentence without highlights', () => {
    expect(markHighlights("She doesn't like it.", ["doesn't", "doesn't like"]).filter((p) => p.mark)).toEqual([
      { text: "doesn't like", mark: true },
    ]);
    expect(markHighlights('No marks.', [])).toEqual([{ text: 'No marks.', mark: false }]);
    expect(markHighlights('Odd (chars)?', ['(chars)?'])).toEqual([
      { text: 'Odd ', mark: false },
      { text: '(chars)?', mark: true },
    ]);
  });
});

describe('GrammarStep', () => {
  let fixture: ComponentFixture<GrammarStep>;
  let el: HTMLElement;
  let speech: FakeSpeech;
  let done: number;
  let backs: number;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const buttons = () => Array.from(el.querySelectorAll<HTMLButtonElement>('button'));
  const named = (label: string) => buttons().find((b) => text(b) === label);
  const main = () => el.querySelector<HTMLButtonElement>('.step-nav .btn-primary')!;
  const backBtn = () => el.querySelector<HTMLButtonElement>('.step-nav .back-step');
  const link = () => el.querySelector<HTMLAnchorElement>('a.full-link');
  const click = async (b: HTMLElement) => {
    b.click();
    await fixture.whenStable();
  };

  const render = async (inputs: Record<string, unknown>) => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(speech), provideRouter([])] });
    fixture = TestBed.createComponent(GrammarStep);
    for (const [k, v] of Object.entries(inputs)) {
      fixture.componentRef.setInput(k, v);
    }
    done = 0;
    backs = 0;
    fixture.componentInstance.done.subscribe(() => done++);
    fixture.componentInstance.back.subscribe(() => backs++);
    await fixture.whenStable();
    el = fixture.nativeElement;
  };

  it('shows the sentences with the verbs marked, and reads one aloud', async () => {
    await render({ content, number: 4 });
    expect(text(el.querySelector('h2'))).toBe('4. Ngữ pháp');
    expect(text(el.querySelector('.tag'))).toBe('Nhận ra');
    expect(text(el.querySelector('.step-hint'))).toContain('tô vàng');
    expect(Array.from(el.querySelectorAll('mark.hl')).map((m) => text(m))).toEqual(['has', 'Plays', 'sings']);
    expect(el.querySelectorAll('.sentence p')[1].textContent).toBe('She Plays the piano and sings.');
    await click(el.querySelectorAll<HTMLButtonElement>('.say')[0]);
    expect(speech.texts()).toEqual(['The singer has a sweet voice.']);
    expect(text(main())).toBe('Xem quy tắc');
  });

  it('gives feedback on the why-question, right or wrong', async () => {
    await render({ content });
    const status = () => text(el.querySelector('[role="status"]'));
    expect(status()).toBe('');
    await click(el.querySelectorAll<HTMLButtonElement>('.option')[0]);
    expect(status()).toContain('Chưa đúng');
    expect(status()).toContain('Với he/she/it, động từ thêm s.');
    const opts = Array.from(el.querySelectorAll<HTMLButtonElement>('.option'));
    expect(opts[0].classList).toContain('is-wrong');
    expect(opts[1].classList).toContain('is-right');
    expect(opts.every((o) => o.disabled)).toBe(true);
  });

  it('marks a right answer to the why-question', async () => {
    await render({ content });
    await click(el.querySelectorAll<HTMLButtonElement>('.option')[1]);
    expect(text(el.querySelector('[role="status"]'))).toContain('Đúng rồi!');
  });

  it('skips the question when it is empty', async () => {
    await render({ content: { ...content, questionVi: '' } });
    expect(el.querySelectorAll('.option').length).toBe(0);
    expect(el.querySelectorAll('mark.hl').length).toBe(3);
  });

  it('shows the rule card, and ← goes back a phase before leaving the step', async () => {
    await render({ content, canBack: true });
    await click(main());
    expect(text(el.querySelector('.tag'))).toBe('Quy tắc');
    expect(text(el.querySelector('h2'))).toBe('Hiện tại đơn');
    const row = (i: number) =>
      Array.from(el.querySelectorAll('.rule-table tbody tr')[i].querySelectorAll('td')).map((td) => text(td));
    expect(row(1)).toEqual(['he / she / it', 'likes', "doesn't like"]);
    expect(row(2)).toEqual(['he / she / it', 'has', '—']);
    expect(text(el.querySelector('.mistakes .wrong'))).toContain('She play the piano.');
    expect(text(el.querySelector('.mistakes .right'))).toContain('She plays the piano.');
    expect(text(el.querySelector('.muted'))).toBe('Tiếng Việt không đổi động từ.');
    expect(text(main())).toBe('Luyện 2 câu');

    await click(backBtn()!);
    expect(text(el.querySelector('.tag'))).toBe('Nhận ra');
    expect(backs).toBe(0);
    await click(backBtn()!);
    expect(backs).toBe(1);
  });

  it('hides ← on the first screen when there is no step before', async () => {
    await render({ content });
    expect(backBtn()).toBeNull();
    await click(main());
    expect(backBtn()).not.toBeNull();
  });

  it('runs the exercises one at a time, then links to the full point and finishes', async () => {
    await render({ content, grammarPointId: 'present-simple', grammarTitleVi: 'Thì hiện tại đơn', finishLabel: 'Sang bước Nói' });
    await click(main());
    await click(main());
    expect(text(el.querySelector('.count'))).toBe('1/2');
    expect(el.querySelector('lu-grammar-exercise')).not.toBeNull();
    expect(main().disabled).toBe(true);
    expect(link()).toBeNull();

    // A choice is checked when tapped.
    await click(named('plays')!);
    expect(text(el.querySelector('lu-grammar-exercise [role="status"]'))).toContain('Đúng');
    expect(text(main())).toBe('Câu tiếp');
    await click(main());

    expect(text(el.querySelector('.count'))).toBe('2/2');
    expect(main().disabled).toBe(true);
    const input = el.querySelector<HTMLInputElement>('lu-grammar-exercise input')!;
    input.value = 'like';
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(text(main())).toBe('Kiểm tra');
    expect(main().disabled).toBe(false);
    await click(main());
    const result = text(el.querySelector('lu-grammar-exercise [role="status"]'));
    expect(result).toContain('Sai');
    expect(result).toContain('likes');

    expect(text(link())).toContain('Học đầy đủ: Thì hiện tại đơn');
    expect(link()!.getAttribute('href')).toBe('/grammar/present-simple');
    expect(text(main())).toBe('Sang bước Nói');
    expect(done).toBe(0);
    await click(main());
    expect(done).toBe(1);
  });

  it('finishes from the rule when there are no exercises', async () => {
    await render({ content: { ...content, exercises: [] }, grammarPointId: 'p1', finishLabel: 'Tiếp' });
    await click(main());
    expect(link()).not.toBeNull();
    expect(text(main())).toBe('Tiếp');
    await click(main());
    expect(done).toBe(1);
  });

  it('shows the old grammar note when there is no content', async () => {
    await render({ note, grammarPointId: 'p1', grammarTitleVi: 'Hiện tại đơn', number: 4, finishLabel: 'Tiếp' });
    expect(text(el.querySelector('h2'))).toBe('4. Ngữ pháp');
    expect(text(el.querySelector('.sub'))).toBe('Present simple');
    expect(text(el.querySelector('.body'))).toBe('Thêm s cho he/she/it.');
    expect(text(el.querySelector('.examples'))).toBe('She likes music.');
    expect(link()!.getAttribute('href')).toBe('/grammar/p1');
    await click(main());
    expect(done).toBe(1);
  });

  it('says there is no grammar when there is neither content nor note', async () => {
    await render({ finishLabel: 'Tiếp' });
    expect(text(el.querySelector('.none'))).toBe('Bài này chưa có phần ngữ pháp.');
    expect(link()).toBeNull();
    expect(text(main())).toBe('Tiếp');
    await click(main());
    expect(done).toBe(1);
  });
});
