import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { spellings } from '../irregular-verbs.logic';
import { LISTEN_SIZE, listenPool, listenRound } from '../verb-listen.logic';
import { VERB_LESSONS } from '../verb-lessons.data';
import { lessonVerbs } from '../verb-study.logic';
import { VerbListen } from './verb-listen';

const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
const verb = (base: string) => IRREGULAR_VERBS.find((v) => v.base === base)!;

describe('listenRound (added 2026-11-10)', () => {
  it('never offers a verb that also has the word heard among its forms', () => {
    for (let seed = 1; seed < 40; seed++) {
      for (const q of listenRound(IRREGULAR_VERBS, seed, 30)) {
        expect(q.options.map((o) => o.base)).toContain(q.verb.base);
        expect(new Set(q.options.map((o) => o.base)).size).toBe(q.options.length);
        for (const o of q.options.filter((x) => x !== q.verb)) {
          const forms = [o.base, ...spellings(o.past), ...spellings(o.participle)];
          expect(forms).not.toContain(q.word);
        }
        // A form spelt like the base, or one the voice misreads, is never asked.
        expect(q.word).not.toBe(q.verb.base);
        expect(q.word).not.toBe('wound');
      }
    }
  });

  it("asks the learned verbs once there are enough, a lesson's when given, else the common ones", () => {
    const lesson8 = lessonVerbs(VERB_LESSONS[7]);
    expect(listenPool(new Map(), lesson8).map((v) => v.base)).toEqual(lesson8.map((v) => v.base));
    expect(listenPool(new Map()).every((v) => v.common)).toBe(true);
    expect(listenRound(listenPool(new Map()), 3).length).toBe(LISTEN_SIZE);
    // Lesson 1 (cut, put…) has no form worth hearing.
    expect(listenPool(new Map(), lessonVerbs(VERB_LESSONS[0]))).toEqual([]);
  });
});

describe('VerbListen (added 2026-11-10)', () => {
  let fixture: ComponentFixture<VerbListen>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let speech: FakeSpeech;
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };

  const setup = async (lesson?: number) => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({
      imports: [VerbListen],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: convertToParamMap(lesson ? { lesson: String(lesson) } : {}),
            },
          },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbListen);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/images').flush({ images: [] });
    http.expectOne('/verbs/progress').flush({ verbs: [] });
    await settle();
  };

  afterEach(() => http.verify());

  it('reads a form, takes a guess and then shows how each form is said', async () => {
    await setup(8); // buy, bring, think, fight, catch, teach
    expect(text(el.querySelector('.bar-title'))).toBe('Luyện nghe · Bài 8');
    el.querySelector<HTMLButtonElement>('.play')!.click();
    const heard = speech.last().text;
    const answer = IRREGULAR_VERBS.find(
      (v) => spellings(v.past)[0] === heard || spellings(v.participle)[0] === heard,
    )!;
    expect(['buy', 'bring', 'think', 'fight', 'catch', 'teach']).toContain(answer.base);

    Array.from(el.querySelectorAll<HTMLButtonElement>('.option'))
      .find((o) => text(o).startsWith(answer.base))!
      .click();
    await settle();
    expect(text(el.querySelector('.verdict'))).toBe('✓ Đúng');
    expect(text(el.querySelector('.answer'))).toContain(`V2 và V3 của ${answer.base}`);
    const hear = el.querySelectorAll<HTMLButtonElement>('.answer .hear-btn');
    expect(hear.length).toBe(3);
    hear[0].click();
    expect(speech.last().text).toBe(answer.base);

    button('Câu tiếp theo').click();
    await settle();
    // Going on reads the next form at once.
    expect(speech.texts().length).toBe(3);
  });

  it('lists the forms said otherwise than written, "read" heard as /red/', async () => {
    await setup();
    button('Đọc khác viết').click();
    await settle();
    const read = Array.from(el.querySelectorAll('.sound')).find((s) => text(s).includes('/red/'))!;
    read.querySelectorAll<HTMLButtonElement>('.hear-btn')[1].click();
    expect(speech.last().text).toBe('red');
    expect(verb('read').past).toBe('read');
  });
});
