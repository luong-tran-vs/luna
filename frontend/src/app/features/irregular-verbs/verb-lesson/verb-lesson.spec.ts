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
import { VerbLessonPage } from './verb-lesson';

describe('VerbLessonPage', () => {
  let fixture: ComponentFixture<VerbLessonPage>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const lesson = VERB_LESSONS[2]; // V3 giống nguyên mẫu: come, become, run
  const verbOf = (base: string) => IRREGULAR_VERBS.find((v) => v.base === base)!;
  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!;
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
        { provide: ActivatedRoute, useValue: { paramMap: of(convertToParamMap({ number: String(number) })) } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbLessonPage);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs });
    await settle();
  };

  afterEach(() => http.verify());

  /** Goes through Làm quen and Chọn, choosing the right option each time. */
  const learnAndChoose = async () => {
    for (let i = 0; i < lesson.bases.length; i++) {
      button(i + 1 < lesson.bases.length ? 'Từ tiếp theo' : 'Sang bước Chọn').click();
      await settle();
    }
    for (let i = 0; i < lesson.bases.length; i++) {
      const base = text(el.querySelector('#ask-title'));
      const v = verbOf(base);
      const asked = text(el.querySelector('.ask .muted')).startsWith('Quá khứ (V2)') ? v.past : v.participle;
      Array.from(el.querySelectorAll<HTMLButtonElement>('.option'))
        .find((o) => text(o) === spellings(asked)[0])!
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

  it('teaches the verbs on cards, with the tip, the forms and an example', async () => {
    await setup(3);
    expect(text(el.querySelector('.title'))).toBe(`Bài 3 · ${lesson.title}`);
    expect(text(el.querySelector('.bubble'))).toBe(lesson.tipVi);
    const card = el.querySelector('.flashcard')!;
    expect(text(card.querySelector('.meaning'))).toBe(verbOf('come').meaningVi);
    expect(Array.from(card.querySelectorAll('.form-word')).map((f) => text(f))).toEqual(['come', 'came', 'come']);
    expect(text(card.querySelector('.changed'))).toBe('ame');
    expect(text(card.querySelector('.example-en'))).toContain('came');
    expect(el.querySelector('.step.is-current')?.textContent).toContain('Làm quen');

    button('Từ tiếp theo').click();
    await settle();
    expect(text(el.querySelector('#card-title'))).toContain('become');
    el.querySelector<HTMLButtonElement>('[aria-label="Từ trước"]')!.click();
    await settle();
    expect(text(el.querySelector('#card-title'))).toContain('come');
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
    expect(el.querySelector('.bottom-bar a.btn-primary')?.getAttribute('href')).toBe('/irregular-verbs/lessons/4');
  });

  it('takes a verb learned before out when it is unticked', async () => {
    const learned = { base: 'run', learnedAt: '2026-10-01T00:00:00Z', level: 2, dueAt: null, right: 1, wrong: 0 };
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
