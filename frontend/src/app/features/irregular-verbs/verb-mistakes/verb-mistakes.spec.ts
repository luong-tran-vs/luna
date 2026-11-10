import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { VerbProgress } from '../../../core/models/verbs';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { VerbGroupPage } from '../verb-group/verb-group';
import { VerbToday } from '../verb-today/verb-today';
import { VerbMistakes } from './verb-mistakes';

const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
const settle = async (f: ComponentFixture<unknown>) => {
  await new Promise((r) => setTimeout(r));
  await f.whenStable();
};
const daysAgo = (n: number) => new Date(Date.now() - n * 24 * 3600 * 1000).toISOString();
const progress = (base: string, over: Partial<VerbProgress> = {}): VerbProgress => ({
  base,
  learnedAt: daysAgo(10),
  level: 2,
  dueAt: daysAgo(-3),
  right: 1,
  wrong: 2,
  ...over,
});
const chooseMistakes = progress('choose', {
  mistakes: [
    { form: 'participle', typed: 'chose', at: daysAgo(3) },
    { form: 'participle', typed: 'choosen', at: daysAgo(1) },
  ],
});
const lieMistakes = progress('lie', {
  mistakes: [
    { form: 'past', typed: 'laid', at: daysAgo(2) },
    { form: 'past', typed: 'laid', at: daysAgo(1) },
  ],
});

describe('VerbMistakes (added 2026-11-10)', () => {
  let fixture: ComponentFixture<VerbMistakes>;
  let el: HTMLElement;
  let http: HttpTestingController;
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;

  const setup = async (verbs: VerbProgress[]) => {
    TestBed.configureTestingModule({
      imports: [VerbMistakes],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeSpeech(new FakeSpeech()),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbMistakes);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs });
    http.expectOne('/verbs/images').flush({ images: [] });
    await settle(fixture);
  };

  afterEach(() => http.verify());

  it('lists the verbs often typed wrong with the last mistake letter by letter and the pair', async () => {
    await setup([chooseMistakes, lieMistakes, progress('go')]);
    expect(text(el.querySelector('.bar-title'))).toBe('Từ hay sai · 2');
    const items = Array.from(el.querySelectorAll('.item'));
    expect(items.map((i) => text(i.querySelector('.item-title strong')))).toEqual([
      'choose',
      'lie',
    ]);
    expect(text(items[0].querySelector('.last'))).toBe('V3: choosen → chosen');
    expect(text(items[0].querySelector('.diff .part-extra'))).toBe('o');
    expect(text(items[1].querySelector('.pair'))).toContain('lay');
    expect(text(button('Luyện'))).toBe('Luyện 2 từ hay sai');
  });

  it('practises them by typing, without touching the reviews', async () => {
    await setup([chooseMistakes]);
    button('Luyện').click();
    await settle(fixture);
    const inputs = el.querySelectorAll<HTMLInputElement>('.field input');
    inputs[0].value = 'chose';
    inputs[1].value = 'chosen';
    inputs.forEach((i) => i.dispatchEvent(new Event('input')));
    await settle(fixture);
    button('Kiểm tra').click();
    await settle(fixture);
    (button('Câu tiếp theo') ?? button('Xong')).click();
    await settle(fixture);
    const req = http.expectOne('/verbs/results');
    expect(req.request.body.results).toEqual([{ base: 'choose', right: true, practice: true }]);
    req.flush({ verbs: [{ ...chooseMistakes, cleanRun: 2 }] });
    await settle(fixture);
    expect(text(el.querySelector('.summary-title'))).toBe('Gõ đúng 1/1 từ');
    expect(text(el.querySelector('.results'))).toContain('đã rời danh sách');
  });

  it('says how a verb gets on the list when there is none', async () => {
    await setup([progress('go')]);
    expect(text(el.querySelector('.empty'))).toContain('Chưa có từ hay sai');
  });
});

describe('VerbGroupPage and VerbChant (added 2026-11-10)', () => {
  let speech: FakeSpeech;

  it("shows a group's rule and lessons, and reads its verbs on a beat", async () => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({
      imports: [VerbGroupPage],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        {
          provide: ActivatedRoute,
          useValue: { paramMap: of(convertToParamMap({ key: 'i-a-u' })) },
        },
      ],
    });
    const http = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(VerbGroupPage);
    const el: HTMLElement = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/images').flush({ images: [] });
    http.expectOne('/verbs/progress').flush({ verbs: [progress('sing'), progress('swim')] });
    await settle(fixture);

    expect(text(el.querySelector('.bar-title'))).toBe('Nhóm i – a – u');
    const lessons = Array.from(el.querySelectorAll('.lesson'));
    expect(lessons.map((l) => text(l.querySelector('strong')))).toEqual([
      'Bài 13 · Nguyên âm i → a → u',
      'Bài 26 · Nguyên âm i → a → u (2)',
    ]);
    expect(text(lessons[0].querySelector('.state'))).toBe('2/6');

    const chant = el.querySelector('lu-verb-chant')!;
    expect(chant.querySelectorAll('.line').length).toBe(9);
    Array.from(chant.querySelectorAll<HTMLButtonElement>('button'))
      .find((b) => text(b).includes('Đọc cả nhóm'))!
      .click();
    await settle(fixture);
    expect(speech.texts()).toEqual(['begin']);
    expect(chant.querySelector('.line.on')?.textContent).toContain('begin');
    speech.last().handlers.ended!();
    await new Promise((r) => setTimeout(r, 300));
    await settle(fixture);
    expect(speech.texts()).toEqual(['begin', 'began']);
    expect(text(chant.querySelector('.beat.on b'))).toBe('began');
    http.verify();
  });
});

describe('VerbToday (added 2026-11-10)', () => {
  it('ticks the tasks of the day and maps the groups', async () => {
    TestBed.configureTestingModule({ imports: [VerbToday], providers: [provideRouter([])] });
    const fixture = TestBed.createComponent(VerbToday);
    fixture.componentRef.setInput(
      'progress',
      new Map([
        ['cut', progress('cut', { learnedAt: new Date().toISOString(), dueAt: daysAgo(-1) })],
        ['choose', chooseMistakes],
      ]),
    );
    await fixture.whenStable();
    const el: HTMLElement = fixture.nativeElement;
    expect(Array.from(el.querySelectorAll('.task')).map((t) => text(t))).toEqual([
      'Ôn các từ đến hạn, đã xong',
      'Đã học một bài hôm nay, đã xong',
      'Luyện 1 từ hay sai',
    ]);
    expect(text(el.querySelector('.today-title'))).toBe('Còn 1 việc nữa');
    expect(el.querySelector('.mistakes')?.getAttribute('href')).toBe('/irregular-verbs/mistakes');
    expect(text(el.querySelector('.groups .group'))).toContain('Không đổi');
  });
});
