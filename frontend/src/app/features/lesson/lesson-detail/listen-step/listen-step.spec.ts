import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { ListenQuestion, PracticeView } from '../../../../core/models/practice';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { ListenStep, optionOrder } from './listen-step';

const gist: ListenQuestion = {
  kind: 'gist',
  prompt: 'Minh và Anna nói về chuyện gì?',
  promptVi: '',
  options: ['Một buổi hoà nhạc', 'Nhạc mỗi người thích', 'Cách học guitar'],
  answerIndex: 1,
  explanationVi: 'Cả hai kể về nhạc mình thích.',
  turnIndex: -1,
};
const detail: ListenQuestion = {
  kind: 'detail',
  prompt: 'What does Anna play?',
  promptVi: 'Anna chơi gì?',
  options: ['The piano', 'The guitar', 'The drums'],
  answerIndex: 0,
  explanationVi: 'Anna nói cô chơi piano.',
  turnIndex: 1,
};
const tfn: ListenQuestion = {
  kind: 'tfn',
  prompt: "Anna's favorite singer has a loud voice.",
  promptVi: 'Ca sĩ Anna thích có giọng to.',
  options: ['Đúng', 'Sai', 'Không nói'],
  answerIndex: 1,
  explanationVi: 'Anna nói giọng cô ấy ngọt.',
  turnIndex: 2,
};

const practice = (over: Partial<PracticeView> = {}): PracticeView => ({
  status: 'done',
  lessonNumber: 1,
  objectiveVi: '',
  examples: [],
  dialogue: {
    speakers: ['Minh', 'Anna'],
    turns: [
      { speaker: 0, text: 'What music do you like?', meaningVi: 'Bạn thích nhạc gì?' },
      { speaker: 1, text: 'I play the piano.', meaningVi: 'Mình chơi piano.' },
      { speaker: 1, text: 'She has a sweet voice.', meaningVi: 'Cô ấy có giọng ngọt.' },
    ],
  },
  fill: null,
  grammarTipVi: '',
  translations: [],
  steps: {
    listeningQuestions: [gist, detail, tfn],
    readingGuideVi: '',
    grammar: null,
    readAloud: [],
    speakQuestions: [],
    writing: null,
  },
  dictation: [
    { text: 'She has a sweet voice.', lemma: 'voice' },
    { text: 'I like pop music.', lemma: 'music' },
  ],
  ...over,
});

describe('optionOrder', () => {
  it('shuffles the options but keeps a tfn question in order', () => {
    const orders = [1, 2, 3, 4, 5, 6].map((seed) => optionOrder(detail, seed).join());
    expect(orders.every((o) => o.split(',').sort().join() === '0,1,2')).toBe(true);
    expect(new Set(orders).size).toBeGreaterThan(1);
    expect(optionOrder(tfn, 3)).toEqual([0, 1, 2]);
  });
});

describe('ListenStep', () => {
  let fixture: ComponentFixture<ListenStep>;
  let el: HTMLElement;
  let speech: FakeSpeech;
  let http: HttpTestingController;
  let events: string[];

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const buttons = () => Array.from(el.querySelectorAll<HTMLButtonElement>('button'));
  const named = (label: string) => buttons().find((b) => text(b) === label);
  const option = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.option')).find((b) => text(b).includes(label))!;
  const main = () => el.querySelector<HTMLButtonElement>('.step-nav .btn-primary')!;
  const stable = () => fixture.whenStable();

  const setup = async (inputs: Record<string, unknown> = {}) => {
    events = [];
    speech = new FakeSpeech();
    TestBed.configureTestingModule({
      providers: [
        provideFakeSpeech(speech),
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({}), queryParamMap: convertToParamMap({}) } } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ListenStep);
    fixture.componentRef.setInput('lessonId', 'l1');
    fixture.componentRef.setInput('nextLabel', 'Sang bước Đọc');
    for (const [k, v] of Object.entries(inputs)) {
      fixture.componentRef.setInput(k, v);
    }
    const c = fixture.componentInstance;
    c.back.subscribe(() => events.push('back'));
    c.leave.subscribe(() => events.push('leave'));
    c.completed.subscribe(() => events.push('completed'));
    el = fixture.nativeElement as HTMLElement;
    await stable();
  };

  /** Picks the option and checks it. */
  const answer = async (label: string) => {
    option(label).click();
    await stable();
    main().click();
    await stable();
  };

  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn();
  });

  afterEach(() => {
    http?.verify();
    vi.restoreAllMocks();
  });

  it('starts with the gist: the speakers, no text, and the dialogue read with one voice per speaker', async () => {
    await setup({ practice: practice() });
    expect(text(el.querySelector('.step-title'))).toContain('lượt 1');
    expect(text(el.querySelector('.people'))).toContain('Minh và Anna');
    expect(el.textContent).not.toContain('What music do you like?');
    expect(el.querySelectorAll('.option').length).toBe(3);

    el.querySelector<HTMLButtonElement>('lu-audio-bar .play')!.click();
    await stable();
    expect(speech.last()).toMatchObject({ text: 'What music do you like?', voice: 0 });
    speech.last().handlers.ended!();
    expect(speech.last()).toMatchObject({ text: 'I play the piano.', voice: 1 });
  });

  it('checks the gist in the browser, then moves to the details', async () => {
    await setup({ practice: practice() });
    expect(main().disabled).toBe(true);
    expect(text(main())).toBe('Kiểm tra');
    await answer('Một buổi hoà nhạc');
    expect(text(el.querySelector('.feedback'))).toContain('Chưa đúng. Đáp án: Nhạc mỗi người thích');
    expect(text(el.querySelector('.explain'))).toBe('Cả hai kể về nhạc mình thích.');
    expect(text(main())).toBe('Sang lượt 2');

    main().click();
    await stable();
    expect(text(el.querySelector('.step-title'))).toContain('lượt 2');
    expect(text(el.querySelector('.badge'))).toBe('Chi tiết · 1/2');
    expect(Array.from(el.querySelectorAll('.turn-chip')).map((b) => text(b))).toEqual(['M1', 'A2', 'A3']);
  });

  it('keeps the right answer whatever the shuffled order', async () => {
    for (const r of [0.1, 0.5, 0.9]) {
      TestBed.resetTestingModule();
      vi.spyOn(Math, 'random').mockReturnValue(r);
      await setup({ practice: practice() });
      await answer('Nhạc mỗi người thích');
      expect(text(el.querySelector('.result'))).toBe('Đúng rồi!');
      vi.restoreAllMocks();
    }
  });

  it('shows the turn holding the answer with a way to hear it, and the text once all is answered', async () => {
    await setup({ practice: practice() });
    await answer('Nhạc mỗi người thích');
    main().click();
    await stable();

    el.querySelector<HTMLButtonElement>('.turn-chip')!.click();
    expect(speech.last().text).toBe('What music do you like?');

    await answer('The piano');
    expect(text(el.querySelector('.evidence'))).toContain('Anna nói: I play the piano.');
    el.querySelector<HTMLButtonElement>('.evidence .icon-btn')!.click();
    expect(speech.last()).toMatchObject({ text: 'I play the piano.', voice: 1 });
    expect(named('Hiện lời')).toBeUndefined();

    main().click();
    await stable();
    // tfn: the fixed three options in their order, no letters.
    expect(Array.from(el.querySelectorAll('.option')).map((b) => text(b))).toEqual(['Đúng', 'Sai', 'Không nói']);
    await answer('Sai');
    named('Hiện lời')!.click();
    await stable();
    expect(text(el.querySelector('.transcript'))).toContain('Cô ấy có giọng ngọt.');
    expect(text(main())).toBe('Sang phần chép lại');

    main().click();
    await stable();
    expect(el.querySelector('lu-listening')).not.toBeNull();
    expect(text(el.querySelector('.head-card'))).toContain('Chỉ 2 câu ngắn');
    http.expectOne('/lessons/l1/dictation/summary').flush({ summary: null });
  });

  it('goes back from lượt 2 to lượt 1 and from lượt 1 to the step before', async () => {
    await setup({ practice: practice(), canBack: true });
    await answer('Nhạc mỗi người thích');
    main().click();
    await stable();
    el.querySelector<HTMLButtonElement>('.step-nav .nav-btn')!.click();
    await stable();
    expect(text(el.querySelector('.step-title'))).toContain('lượt 1');
    expect(text(el.querySelector('.result'))).toBe('Đúng rồi!');
    el.querySelector<HTMLButtonElement>('.step-nav .nav-btn')!.click();
    expect(events).toEqual(['back']);
  });

  it('offers to leave for later, and in review to go on', async () => {
    await setup({ practice: practice(), laterLabel: 'Để sau, sang bước Đọc' });
    named('Để sau, sang bước Đọc')!.click();
    expect(events).toEqual(['leave']);

    TestBed.resetTestingModule();
    await setup({ practice: practice(), mode: 'review' });
    named('Sang bước Đọc')!.click();
    expect(events).toEqual(['leave']);
  });

  describe('dictation', () => {
    it('goes straight to the dictation without the new content', async () => {
      await setup({ practice: practice({ steps: null }) });
      expect(el.querySelector('.option')).toBeNull();
      http.expectOne('/lessons/l1/dictation/summary').flush({ summary: null });
      await stable();
      await stable();
      // The given sentences, not the lesson's: no lesson is fetched and the counter says 2.
      expect(text(el.querySelector('.counter'))).toBe('Câu 1 / 2');
      expect(el.querySelector('.head-card')).toBeNull();
    });

    it('opens at the dictation when it was reached before', async () => {
      await setup({ practice: practice(), startSentence: 1 });
      expect(el.querySelector('lu-listening')).not.toBeNull();
      http.expectOne('/lessons/l1/dictation/summary').flush({ summary: null });
    });

    it('dictates the lesson sentences as before while there is no practice', async () => {
      await setup();
      http.expectOne('/lessons/l1');
      http.expectOne('/lessons/l1/dictation/summary');
    });

    it('records each sentence and reports the end', async () => {
      await setup({ practice: practice({ steps: null }) });
      http.expectOne('/lessons/l1/dictation/summary').flush({ summary: null });
      await new Promise((r) => setTimeout(r));
      await stable();
      for (const typed of ['She has a sweet voise.', 'I like pop music.']) {
        const box = el.querySelector<HTMLTextAreaElement>('#answer')!;
        box.value = typed;
        box.dispatchEvent(new Event('input'));
        await stable();
        el.querySelector<HTMLButtonElement>('.bottom-bar .main')!.click();
        await new Promise((r) => setTimeout(r));
        await stable();
        const post = http.expectOne((req) => req.url === '/lessons/l1/dictation' && req.method === 'POST');
        if (typed.includes('voise')) {
          expect(post.request.body).toMatchObject({ sentenceIndex: 0, correctWords: 5, totalWords: 5 });
          expect(text(el.querySelector('.slip-note'))).toContain('vẫn tính là nghe đúng');
          el.querySelector<HTMLButtonElement>('.bottom-bar .main')!.click();
          await stable();
        }
        post.flush({ summary: null });
        await new Promise((r) => setTimeout(r));
      }
      expect(events).toEqual(['completed']);
    });

    it('goes back from the first sentence to the questions', async () => {
      await setup({ practice: practice(), startSentence: 1 });
      http.expectOne('/lessons/l1/dictation/summary').flush({ summary: null });
      await new Promise((r) => setTimeout(r));
      await stable();
      const prev = el.querySelector<HTMLButtonElement>('.bottom-bar .nav-btn')!;
      prev.click(); // sentence 2 → 1
      await stable();
      prev.click();
      await stable();
      expect(text(el.querySelector('.step-title'))).toContain('lượt 2');
    });
  });
});
