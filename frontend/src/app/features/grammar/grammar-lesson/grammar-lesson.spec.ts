import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
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
  const tab = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('[role="tab"]')).find((t) => text(t) === label)!;
  const panel = (id: string) => el.querySelector<HTMLElement>(`#gpanel-${id}`)!;

  const setup = async (supported = true) => {
    speech = new FakeSpeech();
    speech.supported = supported;
    TestBed.configureTestingModule({
      imports: [GrammarLesson],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
        { provide: ActivatedRoute, useValue: { paramMap: of(convertToParamMap({ pointId: 'a1-to-be' })) } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarLesson);
    el = fixture.nativeElement;
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  describe('with a published lesson', () => {
    beforeEach(async () => {
      await setup();
      http.expectOne('/grammar/a1-to-be').flush(detail);
      await settle();
    });

    it('shows the title, the status in words and every part of the Học tab', () => {
      expect(text(el.querySelector('h1'))).toBe('Động từ to be');
      expect(text(el.querySelector('.status'))).toBe('Đang học');
      const learn = text(panel('learn'));
      expect(learn).toContain('Bạn có thể giới thiệu bản thân.');
      expect(learn).toContain('Đoạn hai.');
      expect(learn).toContain('Nói về nghề nghiệp');
      expect(learn).toContain('S + am/is/are');
      expect(learn).toContain('Tôi là học sinh.');
      expect(learn).toContain('Sai She are a teacher.');
      expect(learn).toContain('Đúng She is a teacher.');
      expect(learn).toContain('Có 3 bài học theo chủ đề');
    });

    it('reads an example aloud with the browser voice', () => {
      el.querySelectorAll<HTMLButtonElement>('.icon-btn')[1].click();
      expect(speech.texts()).toEqual(['She is a teacher.']);
    });

    it('is a set of tabs with one selected panel, moved by the arrow keys', async () => {
      const tabs = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
      expect(tabs.map((t) => text(t))).toEqual(['Học', 'Luyện tập', 'Kiểm tra']);
      expect(tabs.map((t) => t.getAttribute('aria-selected'))).toEqual(['true', 'false', 'false']);
      expect(tabs.map((t) => t.tabIndex)).toEqual([0, -1, -1]);
      expect(tabs[0].getAttribute('aria-controls')).toBe('gpanel-learn');
      expect(panel('learn').getAttribute('aria-labelledby')).toBe('gtab-learn');
      expect(panel('practice').hidden).toBe(true);

      tabs[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
      await settle();
      expect(tab('Luyện tập').getAttribute('aria-selected')).toBe('true');
      expect(document.activeElement).toBe(tab('Luyện tập'));
      expect(panel('practice').hidden).toBe(false);
      expect(panel('learn').hidden).toBe(true);

      tab('Luyện tập').dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true }));
      await settle();
      expect(tab('Kiểm tra').getAttribute('aria-selected')).toBe('true');
      tab('Kiểm tra').dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
      await settle();
      expect(tab('Học').getAttribute('aria-selected')).toBe('true');
    });

    it('records a practice round and updates the status when the mastery test is passed', async () => {
      tab('Kiểm tra').click();
      await settle();
      const test = panel('test');
      const click = async (label: string) => {
        Array.from(test.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!.click();
        await settle();
      };
      await click('a');
      await click('Câu tiếp theo');
      await click('a');
      await click('Xem kết quả');
      const req = http.expectOne('/grammar/a1-to-be/attempts');
      expect(req.request.body).toEqual({ kind: 'mastery', correct: 2, total: 2, wrong: [] });
      req.flush({
        progress: { ...detail.progress, status: 'mastered', mastered: true, bestMastery: 100, weak: [] },
        passed: true,
      });
      await settle();
      expect(text(test.querySelector('.verdict'))).toContain('Đã nắm vững ✓');
      expect(text(el.querySelector('.status'))).toBe('Đã nắm vững · Điểm kiểm tra tốt nhất: 100%');
    });

    it('offers to redo the exercises answered wrong last time', async () => {
      tab('Luyện tập').click();
      await settle();
      expect(text(panel('practice'))).toContain('Luyện lại các câu sai (1)');
    });
  });

  it('says the point is coming soon when it has no published lesson', async () => {
    await setup();
    http.expectOne('/grammar/a1-to-be').flush({ error: 'not_found' }, { status: 404, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('h1'))).toBe('Điểm này sắp có');
    expect(el.querySelector('.empty a')?.getAttribute('href')).toBe('/grammar');
    expect(el.querySelector('[role="tab"]')).toBeNull();
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
