import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Lesson } from '../../../core/models/lesson';
import { Topic } from '../../../core/models/topic';
import { LessonForm } from './lesson-form';

const lesson = (over: Partial<Lesson> = {}): Lesson => ({
  id: 'l1',
  questions: [],
  grammarNote: null,
  writingPrompt: '',
  extrasEditedByAdmin: false,
  quizVersion: 0,
  practice: null,
  practiceStatus: 'none',
  practiceError: '',
  title: 'Park',
  level: 'B1',
  topicId: 't3',
  topicName: 'Công việc',
  annotationStatus: 'done',
  inRoadmap: false,
  createdAt: '2026-09-29T08:00:00Z',
  content: 'We went to the park.',
  source: 'Tự viết',
  license: 'CC BY',
  revision: 1,
  annotationError: '',
  sentences: [],
  annotations: [],
  ...over,
});

const point = (id: string, count = 0) => ({
  id, level: 'B1', titleVi: `Điểm ${id}`, titleEn: id, pattern: 'p', hintVi: 'h', examples: [], lessonCount: count,
});
const b1Points = [point('b1-a', 2), point('b1-b')];

const topic = (id: string, name: string, level: Topic['level']): Topic => ({
  id, name, level, description: '', lessonCount: 0, roadmapCount: 0, remaining: 0, warning: true, createdAt: '', wordCount: 0, usedWordCount: 0,
});
const topics = [topic('t1', 'Gia đình', 'A1'), topic('t2', 'Mua sắm', 'A1'), topic('t3', 'Công việc', 'B1')];

describe('LessonForm', () => {
  let fixture: ComponentFixture<LessonForm>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const field = (name: string) =>
    el.querySelector<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(`[name="${name}"]`)!;
  const type = async (name: string, value: string) => {
    const f = field(name);
    f.value = value;
    f.dispatchEvent(new Event(f instanceof HTMLSelectElement ? 'change' : 'input'));
    f.dispatchEvent(new Event('blur'));
    await fixture.whenStable();
  };
  const errorOf = (name: string) => {
    const id = field(name).getAttribute('aria-describedby')?.split(' ').find((x) => x.endsWith('-error'));
    return id ? el.querySelector(`#${id}`)?.textContent?.trim() : undefined;
  };
  const submit = async () => {
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
  };
  const flushPoints = async (level: string, topicId: string, points = b1Points) => {
    http.expectOne(`/api/admin/grammar?level=${level}&topicId=${topicId}`).flush({ points });
    await settle();
  };
  const fillValid = async () => {
    await type('title', '  Park  ');
    await type('topicId', 't3');
    await flushPoints('B1', 't3');
    await type('source', 'Tự viết');
    await type('license', 'CC BY');
    await type('content', 'We went to the park.');
  };

  const setup = async (id?: string, topicList: Topic[] = topics) => {
    await TestBed.configureTestingModule({
      imports: [LessonForm],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap(id ? { id } : {}) } } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    fixture = TestBed.createComponent(LessonForm);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/admin/topics').flush({ topics: topicList });
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  describe('create', () => {
    beforeEach(() => setup());

    it('shows Vietnamese errors for missing fields and sends nothing', async () => {
      await submit();
      expect(errorOf('title')).toBe('Vui lòng nhập tiêu đề');
      expect(errorOf('content')).toBe('Vui lòng dán nội dung bài');
      expect(errorOf('topicId')).toBe('Vui lòng chọn chủ đề');
      expect(errorOf('source')).toBe('Vui lòng nhập nguồn');
      expect(errorOf('license')).toBe('Vui lòng nhập giấy phép');
      http.expectNone('/api/admin/lessons');
    });

    it('limits lengths', async () => {
      await type('title', 'a'.repeat(201));
      expect(errorOf('title')).toBe('Tối đa 200 ký tự');
    });

    it('offers one topic select grouped by level instead of level and topic inputs', async () => {
      expect(el.querySelector('[name="level"]')).toBeNull();
      expect(el.querySelector('[name="topic"]')).toBeNull();
      const groups = Array.from(el.querySelectorAll<HTMLOptGroupElement>('select[name="topicId"] optgroup'));
      expect(groups.map((g) => g.label)).toEqual(['A1', 'B1']);
      expect(Array.from(groups[0].querySelectorAll('option')).map((o) => o.textContent?.trim())).toEqual([
        'A1 · Gia đình',
        'A1 · Mua sắm',
      ]);
    });

    it('posts trimmed values, disables submit while pending, then opens the lesson', async () => {
      await fillValid();
      await submit();
      expect(el.querySelector<HTMLButtonElement>('button[type="submit"]')!.disabled).toBe(true);

      const req = http.expectOne('/api/admin/lessons');
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({
        title: 'Park', topicId: 't3', source: 'Tự viết', license: 'CC BY', content: 'We went to the park.', grammarPointId: '',
      });
      req.flush({ lesson: lesson() }, { status: 201, statusText: 'Created' });
      await settle();
      expect(router.navigateByUrl).toHaveBeenCalledWith('/admin/lessons/l1');
    });

    it('disables the grammar select with a hint until a topic is chosen', () => {
      expect(field('grammarPointId').disabled).toBe(true);
      expect(el.querySelector('#lesson-grammar-help')?.textContent).toContain('Chọn chủ đề trước');
      expect(el.querySelector('label[for="lesson-grammar"]')?.textContent).toContain('Điểm ngữ pháp (tuỳ chọn)');
    });

    it('offers the points of the topic level and sends the chosen one', async () => {
      await type('title', 'Park');
      await type('topicId', 't3');
      await flushPoints('B1', 't3');
      const sel = field('grammarPointId') as HTMLSelectElement;
      expect(sel.disabled).toBe(false);
      expect(Array.from(sel.options).map((o) => o.textContent?.trim())).toEqual([
        'Không gán', 'Điểm b1-a (2 bài)', 'Điểm b1-b (0 bài)',
      ]);
      await type('grammarPointId', 'b1-b');
      await type('source', 'Tự viết');
      await type('license', 'CC BY');
      await type('content', 'We went.');
      await submit();
      const req = http.expectOne('/api/admin/lessons');
      expect(req.request.body.grammarPointId).toBe('b1-b');
      req.flush({ lesson: lesson() }, { status: 201, statusText: 'Created' });
      await settle();
    });

    it('clears the point and reloads when the topic changes to another level', async () => {
      await type('topicId', 't3');
      await flushPoints('B1', 't3');
      await type('grammarPointId', 'b1-a');
      await type('topicId', 't1');
      expect(field('grammarPointId').value).toBe('');
      await flushPoints('A1', 't1', [{ ...point('a1-x'), level: 'A1' }]);
      expect((field('grammarPointId') as HTMLSelectElement).options.length).toBe(2);
    });

    it('shows a server error for the grammar point', async () => {
      await fillValid();
      await submit();
      http
        .expectOne('/api/admin/lessons')
        .flush(
          { error: 'validation_failed', message: 'x', fields: { grammarPointId: 'Điểm ngữ pháp không thuộc trình độ này' } },
          { status: 400, statusText: 'Bad Request' },
        );
      await settle();
      expect(errorOf('grammarPointId')).toBe('Điểm ngữ pháp không thuộc trình độ này');
    });

    it('shows server field errors under the inputs', async () => {
      await fillValid();
      await submit();
      http
        .expectOne('/api/admin/lessons')
        .flush(
          { error: 'validation_failed', message: 'Thông tin chưa hợp lệ', fields: { content: 'Bài có 250 câu, tối đa 200 câu' } },
          { status: 400, statusText: 'Bad Request' },
        );
      await settle();
      expect(errorOf('content')).toBe('Bài có 250 câu, tối đa 200 câu');
    });
  });

  it('points to the topic page when there are no topics', async () => {
    await setup(undefined, []);
    expect(el.textContent).toContain('Chưa có chủ đề nào');
    expect(el.querySelector('a[href="/admin/topics"]')?.textContent?.trim()).toBe('Tạo chủ đề');
  });

  describe('edit', () => {
    const load = async (l: Lesson) => {
      await setup('l1');
      http.expectOne('/api/admin/lessons/l1').flush({ lesson: l });
      await settle();
      await flushPoints('B1', 't3');
    };
    const dialog = () => el.querySelector('[role="alertdialog"]');

    it('prefills the form', async () => {
      await load(lesson());
      expect(field('title').value).toBe('Park');
      expect(field('topicId').value).toBe('t3');
      expect(field('content').value).toBe('We went to the park.');
      expect(el.querySelector('h1')?.textContent).toContain('Sửa bài');
    });

    it('keeps the lesson point selected and sends it back', async () => {
      await load(lesson({ grammarPointId: 'b1-a', grammarPointTitle: 'Điểm b1-a' }));
      expect(field('grammarPointId').value).toBe('b1-a');
      await submit();
      const req = http.expectOne('/api/admin/lessons/l1');
      expect(req.request.body.grammarPointId).toBe('b1-a');
      req.flush({ lesson: lesson() });
      await settle();
    });

    it('leaves the point out when the list could not be loaded', async () => {
      await setup('l1');
      http.expectOne('/api/admin/lessons/l1').flush({ lesson: lesson({ grammarPointId: 'b1-a' }) });
      await settle();
      http.expectOne('/api/admin/grammar?level=B1&topicId=t3').flush({}, { status: 500, statusText: 'Error' });
      await settle();
      expect(el.querySelector('#lesson-grammar-help')?.textContent).toContain('Không tải được');
      await submit();
      const req = http.expectOne('/api/admin/lessons/l1');
      expect('grammarPointId' in req.request.body).toBe(false);
      req.flush({ lesson: lesson() });
      await settle();
    });

    it('saves without a dialog when the content is unchanged', async () => {
      await load(lesson({ annotations: [{ text: 'went', lemma: 'go', meaningVi: 'đi', sentenceIndex: 0, editedByAdmin: true }] }));
      await type('title', 'Park 2');
      await submit();
      expect(dialog()).toBeNull();
      const req = http.expectOne('/api/admin/lessons/l1');
      expect(req.request.method).toBe('PUT');
      req.flush({ lesson: lesson({ title: 'Park 2' }) });
      await settle();
      expect(router.navigateByUrl).toHaveBeenCalledWith('/admin/lessons/l1');
    });

    it('asks before replacing manually edited annotations', async () => {
      await load(lesson({ annotations: [{ text: 'went', lemma: 'go', meaningVi: 'đi', sentenceIndex: 0, editedByAdmin: true }] }));
      await type('content', 'They went home.');
      await submit();
      expect(dialog()?.textContent).toContain('Các chú thích đã sửa tay sẽ bị thay mới');
      http.expectNone('/api/admin/lessons/l1');

      Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Huỷ')!.click();
      await settle();
      expect(dialog()).toBeNull();
      http.expectNone('/api/admin/lessons/l1');

      await submit();
      Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Lưu và làm lại')!.click();
      await settle();
      const req = http.expectOne('/api/admin/lessons/l1');
      expect(req.request.body.content).toBe('They went home.');
      req.flush({ lesson: lesson({ revision: 2 }) });
      await settle();
    });

    it('does not ask when there are no manual edits', async () => {
      await load(lesson());
      await type('content', 'They went home.');
      await submit();
      expect(dialog()).toBeNull();
      http.expectOne('/api/admin/lessons/l1').flush({ lesson: lesson({ revision: 2 }) });
      await settle();
    });
  });
});
