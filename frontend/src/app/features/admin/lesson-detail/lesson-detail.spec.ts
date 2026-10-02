import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { Lesson } from '../../../core/models/lesson';
import { LessonDetail } from './lesson-detail';

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
  content: 'We went to the park. He gave up smoking.',
  source: 'VOA',
  license: 'Public domain',
  revision: 2,
  annotationError: '',
  sentences: [
    { index: 0, text: 'We went to the park.' },
    { index: 1, text: 'He gave up smoking.' },
  ],
  annotations: [
    { text: 'went', lemma: 'go', meaningVi: 'đã đi', sentenceIndex: 0, editedByAdmin: false },
    { text: 'gave up', lemma: 'give up', meaningVi: 'bỏ', sentenceIndex: 1, editedByAdmin: true },
  ],
  ...over,
});

describe('LessonDetail', () => {
  let fixture: ComponentFixture<LessonDetail>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;
  let speech: FakeSpeech;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const button = (text: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === text);
  const load = async (l: Lesson) => {
    http.expectOne('/api/admin/lessons/l1').flush({ lesson: l });
    await settle();
  };

  beforeEach(async () => {
    speech = new FakeSpeech();
    await TestBed.configureTestingModule({
      imports: [LessonDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: 'l1' }) } } },
        provideFakeSpeech(speech),
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    fixture = TestBed.createComponent(LessonDetail);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  describe('preview', () => {
    it('shows info, numbered sentences and status chips', async () => {
      await load(lesson());
      expect(el.querySelector('h1')?.textContent).toContain('Park');
      for (const text of ['B1', 'Công việc', 'VOA', 'Public domain', 'Chú thích: Xong']) {
        expect(el.textContent).toContain(text);
      }
      expect(el.textContent).not.toContain('Audio');
      const items = Array.from(el.querySelectorAll('.sentences li')).map((li) => li.textContent);
      expect(items[0]).toContain('We went to the park.');
      expect(items[1]).toContain('He gave up smoking.');
    });

    it('shows a not-found message', async () => {
      http.expectOne('/api/admin/lessons/l1').flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
      await settle();
      expect(el.textContent).toContain('Không tìm thấy bài học');
    });
  });

  describe('background work', () => {
    it('reads a sentence with the browser voice', async () => {
      await load(lesson());
      el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 2"]')!.click();
      expect(speech.texts()).toEqual(['He gave up smoking.']);
    });

    it('shows the failure reason and retries', async () => {
      await load(lesson({ annotationStatus: 'failed', annotationError: 'AI chưa được cấu hình', annotations: [] }));
      expect(el.textContent).toContain('AI chưa được cấu hình');

      button('Chạy lại chú thích')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/retry?job=annotate').flush({ lesson: lesson({ annotationStatus: 'running', annotations: [] }) });
      await settle();
      // Polling restarts with an immediate reload.
      await load(lesson({ annotationStatus: 'running', annotations: [] }));
      expect(el.textContent).toContain('Chú thích: Đang chạy');
    });

    it('regenerates the practice and polls while it runs (F17)', async () => {
      await load(lesson());
      expect(el.textContent).toContain('Luyện tập: Chưa có');
      button('Tạo lại phần luyện tập')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/practice/regenerate').flush({ lesson: lesson({ practiceStatus: 'running' }) });
      await settle();
      // Polling restarts with an immediate reload.
      await load(lesson({ practiceStatus: 'running' }));
      expect(el.textContent).toContain('Luyện tập: Đang chạy');
      expect(button('Tạo lại phần luyện tập')!.disabled).toBe(true);
    });

    it('runs a done annotation again at once when nothing was edited by hand (F15)', async () => {
      await load(lesson({ annotations: [] }));
      button('Chạy lại chú thích')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/retry?job=annotate').flush({ lesson: lesson({ annotationStatus: 'running' }) });
      await settle();
      await load(lesson({ annotationStatus: 'running' }));
      expect(button('Chạy lại chú thích')).toBeUndefined();
    });

    it('warns before replacing hand-edited annotations and extras (F15)', async () => {
      await load(lesson({ extrasEditedByAdmin: true }));
      button('Chạy lại chú thích')!.click();
      await settle();
      const dialog = el.querySelector('[role="alertdialog"]');
      expect(dialog?.textContent).toContain('chú thích từ đã sửa tay');
      expect(dialog?.textContent).toContain('câu hỏi, ngữ pháp, đề viết đã sửa tay');
      button('Huỷ')!.click();
      await settle();
      http.expectNone('/api/admin/lessons/l1/retry?job=annotate');

      button('Chạy lại chú thích')!.click();
      await settle();
      button('Chạy lại')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/retry?job=annotate').flush({ lesson: lesson({ annotationStatus: 'running' }) });
      await settle();
      await load(lesson({ annotationStatus: 'running' }));
    });

    it('shows the questions editor and keeps the saved lesson (F15)', async () => {
      await load(lesson());
      const extras = el.querySelector('lu-lesson-extras');
      expect(extras?.textContent).toContain('Câu hỏi, ngữ pháp, đề viết');
      button('Lưu câu hỏi, ngữ pháp, đề viết')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/extras').flush({ lesson: lesson({ extrasEditedByAdmin: true }) });
      await settle();
      expect(extras?.textContent).toContain('Đã sửa tay');
    });

    it('shows annotations read-only with the manual-edit label', async () => {
      await load(lesson());
      const rows = Array.from(el.querySelectorAll('.annotations li')).map((r) => r.textContent ?? '');
      expect(rows[0]).toContain('went');
      expect(rows[0]).toContain('go');
      expect(rows[0]).toContain('đã đi');
      expect(rows[1]).toContain('Đã sửa tay');
      expect(rows[0]).not.toContain('Đã sửa tay');
    });
  });

  describe('annotation editor', () => {
    const inputs = (name: string) => Array.from(el.querySelectorAll<HTMLInputElement>(`input[name="${name}"]`));
    const type = async (input: HTMLInputElement, value: string) => {
      input.value = value;
      input.dispatchEvent(new Event('input'));
      await fixture.whenStable();
    };

    it('is disabled while annotations are running', async () => {
      await load(lesson({ annotationStatus: 'running' }));
      expect(button('Sửa chú thích')?.disabled).toBe(true);
    });

    it('edits, adds and removes items, then saves the full list', async () => {
      await load(lesson());
      button('Sửa chú thích')!.click();
      await fixture.whenStable();

      expect(inputs('text')[0].readOnly).toBe(true);
      await type(inputs('meaningVi')[0], 'đi');
      el.querySelector<HTMLButtonElement>('button[aria-label="Xoá chú thích gave up"]')!.click();
      await fixture.whenStable();
      button('Thêm chú thích')!.click();
      await fixture.whenStable();
      const last = inputs('text').length - 1;
      expect(inputs('text')[last].readOnly).toBe(false);
      await type(inputs('text')[last], 'park');
      await type(inputs('lemma')[last], 'park');
      await type(inputs('meaningVi')[last], 'công viên');

      button('Lưu chú thích')!.click();
      await settle();
      const req = http.expectOne('/api/admin/lessons/l1/annotations');
      expect(req.request.body).toEqual({
        annotations: [
          { text: 'went', lemma: 'go', meaningVi: 'đi' },
          { text: 'park', lemma: 'park', meaningVi: 'công viên' },
        ],
      });
      req.flush({ lesson: lesson() });
      await settle();
      expect(inputs('text')).toHaveLength(0); // back to read-only view
    });

    it('shows a server error on the right row', async () => {
      await load(lesson());
      button('Sửa chú thích')!.click();
      await fixture.whenStable();
      button('Thêm chú thích')!.click();
      await fixture.whenStable();
      const last = inputs('text').length - 1;
      await type(inputs('text')[last], 'banana');
      await type(inputs('lemma')[last], 'banana');
      await type(inputs('meaningVi')[last], 'chuối');

      button('Lưu chú thích')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/annotations').flush(
        { error: 'validation_failed', fields: { 'annotations.2.text': 'Cụm từ không có trong bài' } },
        { status: 400, statusText: 'Bad Request' },
      );
      await settle();
      const rows = el.querySelectorAll('.annotation-edit');
      expect(rows[2].textContent).toContain('Cụm từ không có trong bài');
    });

    it('cancel restores the read-only list', async () => {
      await load(lesson());
      button('Sửa chú thích')!.click();
      await fixture.whenStable();
      await type(inputs('meaningVi')[0], 'thay đổi');
      button('Huỷ sửa')!.click();
      await fixture.whenStable();
      expect(el.querySelector('.annotations')?.textContent).toContain('đã đi');
    });
  });

  describe('edit and delete', () => {
    const dialog = () => el.querySelector('[role="alertdialog"]');

    it('links to the edit page, not to the learner steps', async () => {
      await load(lesson());
      expect(el.querySelector('a[href="/admin/lessons/l1/edit"]')).not.toBeNull();
      expect(el.querySelector('a[href^="/lessons/"]')).toBeNull();
    });

    it('deletes after confirmation and goes back to the list', async () => {
      await load(lesson());
      button('Xoá bài')!.click();
      await fixture.whenStable();
      expect(dialog()).not.toBeNull();
      button('Xoá')!.click();
      await settle();
      const req = http.expectOne('/api/admin/lessons/l1');
      expect(req.request.method).toBe('DELETE');
      req.flush(null, { status: 204, statusText: 'No Content' });
      await settle();
      expect(router.navigateByUrl).toHaveBeenCalledWith('/admin');
    });

    it('explains why a roadmap lesson cannot be deleted', async () => {
      await load(lesson({ inRoadmap: true }));
      button('Xoá bài')!.click();
      await fixture.whenStable();
      button('Xoá')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1').flush(
        { error: 'lesson_in_roadmap', message: 'Gỡ bài khỏi lộ trình trước khi xoá' },
        { status: 409, statusText: 'Conflict' },
      );
      await settle();
      expect(el.querySelector('[role="alert"]')?.textContent).toContain('Gỡ bài khỏi lộ trình trước khi xoá');
      expect(router.navigateByUrl).not.toHaveBeenCalled();
    });
  });
});
