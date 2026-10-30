import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarDetail } from '../../../core/models/grammar-study';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { GrammarLesson } from './grammar-lesson';

const exercise = (id: string) => ({
  id, kind: 'choice' as const, text: 'x', options: ['a', 'b', 'c', 'd'], answerIndex: 0, explanationVi: 'vì',
});

const detail: GrammarDetail = {
  point: { id: 'a1-to-be', level: 'A1', titleVi: 'Động từ to be', titleEn: 'The verb to be', pattern: 'S + be', hintVi: '', examples: [] },
  content: {
    objective: 'Bạn có thể giới thiệu bản thân.',
    explanation: ['Đoạn một.', 'Đoạn hai.'],
    usage: ['Nói về nghề nghiệp'],
    structures: [{ label: 'Khẳng định', pattern: 'S + am/is/are', example: 'I am a student.' }],
    examples: [
      { en: 'I am a student.', vi: 'Tôi là học sinh.' },
      { en: 'She is a teacher.', vi: 'Cô ấy là giáo viên.' },
      { en: 'They are happy.', vi: 'Họ vui.' },
    ],
    mistakes: [{ wrong: 'She are a teacher.', right: 'She is a teacher.', noteVi: 'She đi với is.' }],
    practice: ['p1', 'p2'].map(exercise),
    mastery: ['m1', 'm2'].map(exercise),
  },
  progress: {
    status: 'learning', practiceAttempts: 1, lastPractice: 50, masteryAttempts: 0, bestMastery: 0, mastered: false,
    weak: ['p2'],
  },
  lessonCount: 3,
};

describe('GrammarLesson', () => {
  let fixture: ComponentFixture<GrammarLesson>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let speech: FakeSpeech;

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!;
  const shown = () => Array.from(el.querySelectorAll<HTMLElement>('.panel')).filter((p) => !p.hidden).map((p) => p.id);
  const progress = () => el.querySelector('[role="progressbar"]')!.getAttribute('aria-valuenow');
  /** Opens a step from the "i" overview. */
  const openStep = async (label: string) => {
    el.querySelector<HTMLButtonElement>('.info-btn')!.click();
    await settle();
    Array.from(el.querySelectorAll<HTMLButtonElement>('.info-step')).find((b) => text(b).includes(label))!.click();
    await settle();
  };
  /** Clicks a button of one step, found by the start of its text. */
  const clickIn = async (step: string, label: string) => {
    const panel = el.querySelector<HTMLElement>(`#gstep-${step}`)!;
    Array.from(panel.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!.click();
    await settle();
  };

  /** Opens the point, with the query of the link that led there (?step=…). */
  const setup = async (supported = true, query: Record<string, string> = {}) => {
    speech = new FakeSpeech();
    speech.supported = supported;
    TestBed.configureTestingModule({
      imports: [GrammarLesson],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(convertToParamMap({ pointId: 'a1-to-be' })),
            snapshot: { queryParamMap: convertToParamMap(query) },
          },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarLesson);
    el = fixture.nativeElement;
    await fixture.whenStable();
  };
  /** Opens the point with other progress than the default one. */
  const openWith = async (progress: Partial<GrammarDetail['progress']>, query: Record<string, string> = {}) => {
    await setup(true, query);
    http.expectOne('/grammar/a1-to-be').flush({ ...detail, progress: { ...detail.progress, ...progress } });
    await settle();
  };
  const resume = () => el.querySelector<HTMLElement>('.resume');

  beforeEach(() => localStorage.clear());
  afterEach(() => {
    http.verify();
    localStorage.clear();
  });

  describe('when coming back to a point', () => {
    it('offers to redo the wrong answers or take the test, and starts the redo at once', async () => {
      await openWith({});
      expect(text(resume()?.querySelector('.resume-title'))).toBe('Học tiếp từ chỗ dở');
      expect(text(resume())).toContain('Lần trước luyện tập: 50% · còn 1 câu sai. Chưa làm bài kiểm tra.');
      const buttons = Array.from(resume()!.querySelectorAll('button')).map((b) => text(b));
      expect(buttons).toEqual(['Luyện lại 1 câu sai', 'Làm kiểm tra']);

      await clickIn('theory', 'Luyện lại 1 câu sai');
      expect(shown()).toEqual(['gstep-practice']);
      expect(text(el.querySelector('#gstep-practice .count'))).toBe('Câu 1/1');
      expect(text(el.querySelector('#gstep-practice'))).toContain('không tính vào điểm');
    });

    it('offers a quick review on a mastered point', async () => {
      await openWith({ status: 'mastered', mastered: true, bestMastery: 90, weak: [] });
      expect(resume()?.classList).toContain('is-mastered');
      expect(text(resume()?.querySelector('.resume-title'))).toBe('Đã nắm vững · 90%');
      await clickIn('theory', 'Làm lại kiểm tra');
      expect(shown()).toEqual(['gstep-test']);
    });

    it('remembers the step a new learner stopped at in this browser', async () => {
      const fresh = { status: 'new' as const, practiceAttempts: 0, lastPractice: 0, weak: [] };
      await openWith(fresh);
      expect(resume()).toBeNull();
      button('Đã quen? Bỏ qua').click();
      await settle();
      expect(localStorage.getItem('luna.grammar.step.a1-to-be')).toBe('1');

      fixture.destroy();
      TestBed.resetTestingModule();
      await openWith(fresh);
      expect(shown()).toEqual(['gstep-theory']);
      expect(text(resume())).toContain('Lần trước bạn dừng ở Bước 2 · Cấu trúc & ví dụ.');
      await clickIn('theory', 'Tiếp tục Bước 2');
      expect(shown()).toEqual(['gstep-structure']);
    });

    it('opens the step a link names, and redoes the wrong answers when asked', async () => {
      await openWith({}, { step: 'practice', redo: 'weak' });
      expect(shown()).toEqual(['gstep-practice']);
      expect(progress()).toBe('3');
      expect(text(el.querySelector('#gstep-practice .count'))).toBe('Câu 1/1');
    });

    it('ignores a step it does not know', async () => {
      await openWith({}, { step: 'nope' });
      expect(shown()).toEqual(['gstep-theory']);
    });
  });

  describe('with a published lesson', () => {
    beforeEach(async () => {
      await setup();
      http.expectOne('/grammar/a1-to-be').flush(detail);
      await settle();
    });

    it('opens on Lý thuyết with the objective and "Nhớ nhanh", then one card per paragraph and the uses', async () => {
      expect(text(el.querySelector('h1'))).toBe('Động từ to be');
      expect(text(el.querySelector('.unit'))).toBe('A1');
      expect(shown()).toEqual(['gstep-theory']);
      expect(progress()).toBe('1');
      const theory = el.querySelector('#gstep-theory')!;
      const card = () => text(theory.querySelector('.theory-card'));
      expect(text(theory)).toContain('Bạn có thể giới thiệu bản thân.');
      expect(card()).toContain('Nhớ nhanh');
      expect(Array.from(theory.querySelectorAll('.piece')).map((p) => text(p))).toEqual(['S', 'am/is/are']);
      expect(card()).toContain('I am a student.Tôi là học sinh.');
      expect(card()).not.toContain('Đoạn một.');
      expect(theory.querySelectorAll('.dot').length).toBe(4);
      expect(el.querySelector('.bottom-bar .nav-btn')).toBeNull();

      button('Đọc giải thích').click();
      await settle();
      expect(text(theory.querySelector('#g-card'))).toBe('Giải thích · 1/2');
      expect(card()).toContain('Đoạn một.');
      expect(document.activeElement?.id).toBe('g-card');
      button('Thẻ tiếp').click();
      await settle();
      expect(card()).toContain('Đoạn hai.');
      button('Thẻ tiếp').click();
      await settle();
      expect(card()).toContain('Khi nào dùng');
      expect(card()).toContain('Nói về nghề nghiệp');
      expect(button('Sang Cấu trúc & ví dụ')).toBeTruthy();

      el.querySelector<HTMLButtonElement>('.bottom-bar [aria-label="Thẻ trước"]')!.click();
      await settle();
      expect(card()).toContain('Đoạn hai.');
      theory.querySelectorAll<HTMLButtonElement>('.dot')[0].click();
      await settle();
      expect(card()).toContain('Nhớ nhanh');
    });

    it('places the examples of a lesson that names no structure under the tab their form shows', () => {
      const structure = el.querySelector('#gstep-structure')!;
      const tab = structure.querySelector('[role="tab"]')!;
      expect(text(tab)).toBe('Khẳng định');
      expect(tab.classList).toContain('is-positive');
      const panel = structure.querySelector('#gtab-panel')!;
      expect(Array.from(panel.querySelectorAll('.piece')).map((p) => text(p))).toEqual(['S', 'am/is/are']);
      expect(text(panel)).toContain('I am a student.');
      // Not marked: the words the pattern names are coloured from it.
      expect(Array.from(panel.querySelectorAll('.hl')).map((h) => text(h))).toContain('am');
      // All three are affirmative: nothing is left for a list under the tabs.
      expect(text(panel)).toContain('They are happy.Họ vui.');
      expect(structure.querySelector('#g-other-examples')).toBeNull();
      // The mistake whose right sentence has this form is told under the tab; the quiz waits.
      expect(text(structure.querySelector('.tip'))).toBe('Hay sai: She are a teacher. → She is a teacher.');
      expect(structure.querySelector<HTMLElement>('lu-mistake-check')!.hidden).toBe(true);
    });

    it('reads an example aloud with the browser voice', () => {
      // "Nhớ nhanh" comes first, then the examples of the tab.
      el.querySelectorAll<HTMLButtonElement>('.icon-btn')[2].click();
      expect(speech.texts()).toEqual(['She is a teacher.']);
    });

    it('moves through the steps with the bottom bar and back', async () => {
      for (const label of ['Đọc giải thích', 'Thẻ tiếp', 'Thẻ tiếp', 'Sang Cấu trúc & ví dụ']) {
        button(label).click();
        await settle();
      }
      expect(shown()).toEqual(['gstep-structure']);
      expect(progress()).toBe('2');
      expect(document.activeElement?.textContent).toContain('Bước 2');

      // Back lands on the last card of Lý thuyết.
      el.querySelector<HTMLButtonElement>('.bottom-bar [aria-label="Bước trước"]')!.click();
      await settle();
      expect(shown()).toEqual(['gstep-theory']);
      expect(text(el.querySelector('.theory-card'))).toContain('Khi nào dùng');

      button('Sang Cấu trúc & ví dụ').click();
      await settle();
      // After the last tab, the common mistakes on their own screen.
      button('Ôn lỗi thường gặp').click();
      await settle();
      const check = el.querySelector<HTMLElement>('lu-mistake-check')!;
      expect(check.hidden).toBe(false);
      expect(el.querySelector('#gtab-panel')).toBeNull();
      expect(text(check.querySelector('.count'))).toBe('1 / 1');
      expect(button('Câu tiếp').disabled).toBe(true);
      Array.from(check.querySelectorAll<HTMLButtonElement>('.choice')).find((b) => text(b) === 'She is a teacher.')!.click();
      await settle();
      // Back goes to the last tab; forward comes back to the mistake as it was left.
      el.querySelector<HTMLButtonElement>('.bottom-bar [aria-label="Cấu trúc trước"]')!.click();
      await settle();
      expect(el.querySelector('#gtab-panel')).not.toBeNull();
      button('Ôn lỗi thường gặp').click();
      await settle();
      expect(text(check.querySelector('.why'))).toContain('Chính xác!');
      button('Sang Luyện tập').click();
      await settle();
      expect(shown()).toEqual(['gstep-practice']);
      expect(el.querySelector('.grammar > .bottom-bar')).toBeNull();
      expect(text(el.querySelector('#gstep-practice .eyebrow'))).toBe('Bước 3 · Luyện tập');
      expect(text(el.querySelector('#gstep-practice'))).toContain('Luyện lại các câu sai (1)');
    });

    it('opens any step from the overview, which shows the status and the topic lessons', async () => {
      const info = el.querySelector<HTMLButtonElement>('.info-btn')!;
      const overview = el.querySelector<HTMLElement>('#grammar-overview')!;
      expect(overview.hidden).toBe(true);
      info.click();
      await settle();
      expect(info.getAttribute('aria-expanded')).toBe('true');
      expect(overview.hidden).toBe(false);
      expect(text(overview.querySelector('.status'))).toBe('Đang học');
      expect(text(overview)).toContain('Có 3 bài học theo chủ đề');
      expect(overview.querySelector('[aria-current="step"]')?.textContent).toContain('Lý thuyết');

      Array.from(overview.querySelectorAll<HTMLButtonElement>('.info-step')).find((b) => text(b).includes('Kiểm tra'))!.click();
      await settle();
      expect(shown()).toEqual(['gstep-test']);
      expect(progress()).toBe('4');
      expect(overview.hidden).toBe(true);
    });

    it('records the mastery test, updates the status and leads to the next point when passed', async () => {
      const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
      await openStep('Kiểm tra');
      await clickIn('test', 'a');
      await clickIn('test', 'Câu tiếp theo');
      await clickIn('test', 'a');
      await clickIn('test', 'Xem kết quả');
      const req = http.expectOne('/grammar/a1-to-be/attempts');
      expect(req.request.body).toEqual({ kind: 'mastery', correct: 2, total: 2, wrong: [] });
      req.flush({
        progress: { ...detail.progress, status: 'mastered', mastered: true, bestMastery: 100, weak: [] },
        passed: true,
      });
      await settle();
      http.expectOne((r) => r.url === '/grammar' && r.params.get('level') === 'A1').flush({
        next: 'a1-articles',
        points: [
          { id: 'a1-to-be', level: 'A1', titleVi: 'Động từ to be', titleEn: '', hintVi: '', available: true, status: 'mastered', bestMastery: 100 },
          { id: 'a1-articles', level: 'A1', titleVi: 'Mạo từ', titleEn: '', hintVi: '', available: true, status: 'new', bestMastery: 0 },
        ],
      });
      await settle();
      expect(text(el.querySelector('#gstep-test .verdict'))).toContain('Đã nắm vững');
      expect(text(el.querySelector('.status'))).toBe('Đã nắm vững · Điểm kiểm tra tốt nhất: 100%');

      await clickIn('test', 'Học tiếp: Mạo từ');
      expect(navigate).toHaveBeenCalledWith(['/grammar', 'a1-articles']);
    });

    it('goes back to the list after a test that is not passed', async () => {
      const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
      await openStep('Kiểm tra');
      await clickIn('test', 'b');
      await clickIn('test', 'Câu tiếp theo');
      await clickIn('test', 'b');
      await clickIn('test', 'Xem kết quả');
      http.expectOne('/grammar/a1-to-be/attempts').flush({ progress: detail.progress, passed: false });
      await settle();
      http.expectNone('/grammar');
      await clickIn('test', 'Về danh sách ngữ pháp');
      expect(navigate).toHaveBeenCalledWith(['/grammar']);
    });

    it('goes from the practice result to the test', async () => {
      await openStep('Luyện tập');
      await clickIn('practice', 'a');
      await clickIn('practice', 'Câu tiếp theo');
      await clickIn('practice', 'a');
      await clickIn('practice', 'Xem kết quả');
      http.expectOne('/grammar/a1-to-be/attempts').flush({ progress: detail.progress, passed: false });
      await settle();
      await clickIn('practice', 'Sang Kiểm tra');
      expect(shown()).toEqual(['gstep-test']);
    });
  });

  describe('with examples that name their structure', () => {
    const tabbed: GrammarDetail = {
      ...detail,
      content: {
        ...detail.content,
        structures: [
          { label: 'Khẳng định', pattern: 'S + am/is/are', example: 'I am a student.' },
          { label: 'Phủ định', pattern: 'S + am/is/are + not', example: 'I am not tired.' },
          { label: 'Rút gọn', pattern: "I'm", example: "I'm fine." },
        ],
        examples: [
          { en: 'She is a teacher.', vi: 'Cô ấy là giáo viên.', structure: 1, marked: '{1:She} {2:is} a teacher.' },
          { en: 'They are happy.', vi: 'Họ vui.', structure: 1 },
          { en: 'Is she here? – Yes, she is.', vi: 'Cô ấy ở đây không? – Có.', structure: 1 },
          { en: 'He is not here.', vi: 'Anh ấy không ở đây.', structure: 2, marked: '{1:He} {2:is} {3:not} here.' },
          { en: 'Hello!', vi: 'Xin chào!' },
        ],
      },
    };
    const panel = () => el.querySelector<HTMLElement>('#gtab-panel')!;
    const tabs = () => Array.from(el.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
    const marks = () => Array.from(panel().querySelectorAll('.hl')).map((h) => `${text(h)}:${h.className.split(' ').pop()}`);

    beforeEach(async () => {
      await setup();
      http.expectOne('/grammar/a1-to-be').flush(tabbed);
      await settle();
    });

    it('colours the marked words of "Nhớ nhanh" like the pattern', () => {
      const words = Array.from(el.querySelectorAll('.theory-card .hl')).map((h) => text(h));
      expect(words).toEqual(['She', 'is']);
    });

    it('shows one structure at a time, with its own examples coloured like its pattern', async () => {
      await openStep('Cấu trúc');
      expect(tabs().map((t) => text(t))).toEqual(['Khẳng định', 'Phủ định', 'Rút gọn']);
      expect(tabs()[0].getAttribute('aria-selected')).toBe('true');
      expect(tabs()[0].classList).toContain('is-positive');
      expect(text(panel())).toContain('She is a teacher.Cô ấy là giáo viên.');
      expect(text(panel())).toContain('They are happy.');
      const qa = Array.from(panel().querySelectorAll('.example'))[2];
      // The question with its meaning, then the answer with its own, set in under it.
      const lines = Array.from(qa.querySelectorAll('.line')).map((l) => text(l));
      expect(lines).toEqual(['Is she here?Cô ấy ở đây không?', 'Yes, she is.Có.']);
      expect(qa.querySelectorAll('.line')[1].classList).toContain('answer');
      expect(text(panel())).not.toContain('He is not here.');
      expect(marks().slice(0, 3)).toEqual(['She:tone-0', 'is:tone-1', 'are:tone-1']);
      expect(el.querySelector('.structures')).toBeNull();
      expect(text(el.querySelector('#gstep-structure'))).toContain('Ví dụ khácHello!');

      button('Xem Phủ định').click();
      await settle();
      expect(tabs()[1].getAttribute('aria-selected')).toBe('true');
      // The pieces keep their colour across the structures: "not" gets a new one.
      expect(marks()).toEqual(['He:tone-0', 'is:tone-1', 'not:tone-2']);

      // A structure without examples of its own shows its example.
      button('Xem Rút gọn').click();
      await settle();
      expect(text(panel())).toContain("I'm fine.");
      expect(button('Ôn lỗi thường gặp')).toBeTruthy();

      el.querySelector<HTMLButtonElement>('.bottom-bar [aria-label="Cấu trúc trước"]')!.click();
      await settle();
      expect(tabs()[1].getAttribute('aria-selected')).toBe('true');
    });

    it('moves between the tabs with the arrow keys', async () => {
      await openStep('Cấu trúc');
      tabs()[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
      await settle();
      expect(tabs()[2].getAttribute('aria-selected')).toBe('true');
      expect(document.activeElement).toBe(tabs()[2]);
      expect(tabs()[2].tabIndex).toBe(0);
      expect(tabs()[0].tabIndex).toBe(-1);
      tabs()[2].dispatchEvent(new KeyboardEvent('keydown', { key: 'Home' }));
      await settle();
      expect(tabs()[0].getAttribute('aria-selected')).toBe('true');
    });
  });

  it('says the point is coming soon when it has no published lesson', async () => {
    await setup();
    http.expectOne('/grammar/a1-to-be').flush({ error: 'not_found' }, { status: 404, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('h1'))).toBe('Điểm này sắp có');
    expect(el.querySelector('.empty a')?.getAttribute('href')).toBe('/grammar');
    expect(el.querySelector('.lesson-head')).toBeNull();
  });

  it('shows an error with a retry for other failures', async () => {
    await setup();
    http.expectOne('/grammar/a1-to-be').flush({}, { status: 500, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không tải được bài ngữ pháp');
    Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Thử lại')!.click();
    http.expectOne('/grammar/a1-to-be').flush(detail);
    await settle();
    expect(text(el.querySelector('h1'))).toBe('Động từ to be');
  });

  it('hides the listen buttons when the browser has no voice', async () => {
    await setup(false);
    http.expectOne('/grammar/a1-to-be').flush(detail);
    await settle();
    expect(el.querySelectorAll('.icon-btn').length).toBe(0);
  });
});
