import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { VerbProgress } from '../../../core/models/verbs';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { spellings } from '../irregular-verbs.logic';
import { VERB_LESSONS } from '../verb-lessons.data';
import { VERB_USAGE } from '../verb-usage.data';
import { VerbLessonPage } from './verb-lesson';

describe('VerbLessonPage', () => {
  let fixture: ComponentFixture<VerbLessonPage>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const lesson = VERB_LESSONS[2]; // V3 giống nguyên mẫu: come, become, run
  const verbOf = (base: string) => IRREGULAR_VERBS.find((v) => v.base === base)!;
  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };

  const setup = async (number: number, verbs: VerbProgress[] = []) => {
    TestBed.configureTestingModule({
      imports: [VerbLessonPage],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeSpeech(new FakeSpeech()),
        {
          provide: ActivatedRoute,
          useValue: { paramMap: of(convertToParamMap({ number: String(number) })) },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbLessonPage);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs });
    http
      .expectOne('/verbs/images')
      .flush({ images: [{ base: 'come', imageUrl: '/api/verbs/images/come?v=1' }] });
    await settle();
  };

  /** Goes through the rule and the guess (the last verb's V2), right. */
  const ruleAndGuess = async () => {
    button('Đoán thử một từ').click();
    await settle();
    const last = verbOf(lesson.bases[lesson.bases.length - 1]);
    Array.from(el.querySelectorAll<HTMLButtonElement>('.option'))
      .find((o) => text(o) === spellings(last.past)[0])!
      .click();
    await settle();
    button('Xem từng từ').click();
    await settle();
  };

  afterEach(() => http.verify());

  /** Goes through Làm quen and Chọn, choosing the right option each time. */
  const learnAndChoose = async () => {
    await ruleAndGuess();
    for (let i = 0; i < lesson.bases.length; i++) {
      button(i + 1 < lesson.bases.length ? 'Từ tiếp theo' : 'Xem bảng tóm tắt').click();
      await settle();
    }
    button('Sang bước Chọn').click();
    await settle();
    // Chọn trong câu: V2 and V3 in turn, the verbs in their order.
    for (let i = 0; i < lesson.bases.length; i++) {
      const v = verbOf(lesson.bases[i]);
      const answer = spellings(i % 2 === 0 ? v.past : v.participle)[0];
      Array.from(el.querySelectorAll<HTMLButtonElement>('.option'))
        .find((o) => text(o) === answer)!
        .click();
      await settle();
      expect(text(el.querySelector('.feedback'))).toContain('Đúng.');
      button(i + 1 < lesson.bases.length ? 'Câu tiếp theo' : 'Sang bước Gõ').click();
      await settle();
    }
  };

  /** Types every verb of the lesson, the first one wrong once. */
  const typeAll = async () => {
    const typed = new Set<string>();
    for (let guard = 0; guard < 10 && el.querySelector('lu-verb-type-round #verb-base'); guard++) {
      const v = verbOf(text(el.querySelector('#verb-base')));
      const wrongFirst = v.base === lesson.bases[0] && !typed.has(v.base);
      typed.add(v.base);
      const inputs = el.querySelectorAll<HTMLInputElement>('.field input');
      inputs[0].value = spellings(v.past)[0];
      inputs[1].value = wrongFirst ? 'nope' : spellings(v.participle)[0];
      inputs.forEach((i) => i.dispatchEvent(new Event('input')));
      await settle();
      button('Kiểm tra').click();
      await settle();
      (button('Câu tiếp theo') ?? button('Xong')).click();
      await settle();
    }
  };

  it('starts with the rule of the group, then a guess of one verb before the cards', async () => {
    await setup(3);
    expect(text(el.querySelector('.title'))).toBe(`Bài 3 · ${lesson.title}`);
    expect(el.querySelector('.step.is-current')?.textContent).toContain('Làm quen');
    expect(text(el.querySelector('.bubble'))).toBe(lesson.tipVi);
    const rows = Array.from(el.querySelectorAll('.forms-table tbody tr'));
    const cells = (r: Element) =>
      Array.from(r.querySelectorAll('td'))
        .map((td) => text(td))
        .join(' ');
    expect(rows.map(cells)).toEqual(['come came come', 'become became become', 'run ? ?']);
    // What changes is coloured: came → "ame" after "c".
    expect(text(rows[0].querySelector('mark.v2'))).toBe('ame');
    expect(rows[0].querySelector('img')?.getAttribute('src')).toContain('/api/verbs/images/come');

    button('Đoán thử một từ').click();
    await settle();
    expect(text(el.querySelector('.guess-title'))).toBe('Theo quy luật, V2 của run là gì?');
    Array.from(el.querySelectorAll<HTMLButtonElement>('.option'))
      .find((o) => text(o) === 'ran')!
      .click();
    await settle();
    expect(text(el.querySelector('.feedback'))).toContain('Đúng rồi!');
    expect(text(el.querySelector('.feedback'))).toContain('run – ran – run');
  });

  it('shows each verb on a card with its picture, beats, mistake and uses', async () => {
    await setup(3);
    await ruleAndGuess();
    const card = () => el.querySelector('lu-verb-card')!;
    expect(card().querySelector('img.picture')?.getAttribute('src')).toContain(
      '/api/verbs/images/come',
    );
    expect(Array.from(card().querySelectorAll('.beat-word')).map((b) => text(b))).toEqual([
      'come',
      'came',
      'come',
    ]);
    expect(text(card().querySelector('.mistake'))).toContain(VERB_USAGE['come'].mistakeVi);
    Array.from(card().querySelectorAll<HTMLButtonElement>('.face'))
      .find((b) => text(b) === 'Dùng khi nào')!
      .click();
    await settle();
    const uses = Array.from(card().querySelectorAll('.use-en')).map((u) => text(u));
    expect(uses).toEqual([VERB_USAGE['come'].past.en, VERB_USAGE['come'].perfect.en]);

    button('Từ tiếp theo').click();
    await settle();
    // A verb without a picture shows no frame; the card starts on its word face.
    expect(card().querySelector('img.picture')).toBeNull();
    expect(Array.from(card().querySelectorAll('.beat-word')).map((b) => text(b))[0]).toBe('become');
    el.querySelector<HTMLButtonElement>('[aria-label="Từ trước"]')!.click();
    await settle();
    expect(text(card().querySelector('.head strong'))).toBe('come');
  });

  it('ends Làm quen with the table, a column of which can be hidden to test oneself', async () => {
    await setup(3);
    await ruleAndGuess();
    for (let i = 0; i < lesson.bases.length; i++) {
      button(i + 1 < lesson.bases.length ? 'Từ tiếp theo' : 'Xem bảng tóm tắt').click();
      await settle();
    }
    const summary = el.querySelector('lu-verb-summary')!;
    expect(summary.querySelectorAll('tbody tr').length).toBe(3);
    button('Che V2').click();
    await settle();
    const covers = summary.querySelectorAll<HTMLButtonElement>('.cover');
    expect(covers.length).toBe(3);
    covers[0].click();
    await settle();
    expect(summary.querySelectorAll('.cover').length).toBe(2);
    expect(text(summary.querySelector('.checked'))).toBe('came');
  });

  it('asks the forms in sentences, the signal words marked', async () => {
    await setup(3);
    await ruleAndGuess();
    for (let i = 0; i < lesson.bases.length; i++) {
      button(i + 1 < lesson.bases.length ? 'Từ tiếp theo' : 'Xem bảng tóm tắt').click();
      await settle();
    }
    button('Sang bước Chọn').click();
    await settle();
    expect(el.querySelector('.step.is-current')?.textContent).toContain('Chọn');
    const sentence = text(el.querySelector('#ask-title'));
    expect(sentence).toContain('?');
    expect(sentence).not.toContain(' came ');
    expect(el.querySelector('#ask-title mark.signal')).not.toBeNull();
    const options = Array.from(el.querySelectorAll('.option')).map((o) => text(o));
    expect(options).toEqual(expect.arrayContaining(['come', 'came', 'comed']));
    // A wrong choice is explained by the signal of the sentence.
    Array.from(el.querySelectorAll<HTMLButtonElement>('.option'))
      .find((o) => text(o) === 'comed')!
      .click();
    await settle();
    expect(text(el.querySelector('.feedback'))).toContain('dấu hiệu quá khứ');
  });

  it('goes through choose and type, then marks the ticked verbs as learned', async () => {
    await setup(3);
    await learnAndChoose();
    expect(el.querySelector('.step.is-current')?.textContent).toContain('Gõ');
    await typeAll();

    // The verb typed wrong first is not ticked; the others are.
    const checks = Array.from(el.querySelectorAll<HTMLInputElement>('.check input'));
    expect(checks.map((c) => c.checked)).toEqual([false, true, true]);
    expect(text(el.querySelector('.check'))).toContain('Gõ sai');
    checks[0].click();
    await settle();
    expect(text(button('Xác nhận đã học'))).toBe('Xác nhận đã học 3 từ');
    button('Xác nhận đã học').click();
    const req = http.expectOne('/verbs/learned');
    expect(req.request.body).toEqual({ bases: ['come', 'become', 'run'], learned: true });
    req.flush({
      verbs: lesson.bases.map((b) => ({
        base: b,
        learnedAt: '2026-10-19T00:00:00Z',
        level: 1,
        dueAt: null,
        right: 0,
        wrong: 0,
      })),
    });
    await settle();

    expect(text(el.querySelector('.done-title'))).toBe('Xong bài 3');
    expect(text(el.querySelector('.gain'))).toBe('Đã học 3 động từ');
    expect(el.querySelector('.bottom-bar a.btn-primary')?.getAttribute('href')).toBe(
      '/irregular-verbs/lessons/4',
    );
  });

  it('takes a verb learned before out when it is unticked', async () => {
    const learned = {
      base: 'run',
      learnedAt: '2026-10-01T00:00:00Z',
      level: 2,
      dueAt: null,
      right: 1,
      wrong: 0,
    };
    await setup(3, [learned]);
    await learnAndChoose();
    await typeAll();
    const checks = Array.from(el.querySelectorAll<HTMLInputElement>('.check input'));
    expect(checks.map((c) => c.checked)).toEqual([false, true, true]);
    checks[2].click();
    await settle();
    button('Xác nhận đã học').click();
    const learn = http.expectOne((r) => r.url === '/verbs/learned' && r.body.learned === true);
    expect(learn.request.body.bases).toEqual(['become']);
    learn.flush({ verbs: [] });
    const unlearn = http.expectOne((r) => r.url === '/verbs/learned' && r.body.learned === false);
    expect(unlearn.request.body.bases).toEqual(['run']);
    unlearn.flush({ verbs: [] });
    await settle();
    expect(el.querySelector('.done')).not.toBeNull();
  });

  it('says when the lesson does not exist', async () => {
    await setup(99);
    expect(text(el)).toContain('Không có bài này');
  });
});
