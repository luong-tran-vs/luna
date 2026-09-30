import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Lesson } from '../../../core/models/lesson';
import { LessonDetail } from './lesson-detail';

const lesson = (over: Partial<Lesson> = {}): Lesson => ({
  id: 'l1',
  title: 'Park',
  level: 'B1',
  topicId: 't3',
  topicName: 'Công việc',
  audioStatus: 'done',
  annotationStatus: 'done',
  inRoadmap: false,
  createdAt: '2026-09-29T08:00:00Z',
  content: 'We went to the park. He gave up smoking.',
  source: 'VOA',
  license: 'Public domain',
  revision: 2,
  audioError: '',
  annotationError: '',
  sentences: [
    { index: 0, text: 'We went to the park.', audioUrl: '/api/audio/l1/2/0' },
    { index: 1, text: 'He gave up smoking.', audioUrl: '/api/audio/l1/2/1' },
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
    await TestBed.configureTestingModule({
      imports: [LessonDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: 'l1' }) } } },
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
      for (const text of ['B1', 'Công việc', 'VOA', 'Public domain', 'Audio: Xong', 'Chú thích: Xong']) {
        expect(el.textContent).toContain(text);
      }
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
    it('plays a sentence through the shared audio element', async () => {
      const play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue();
      await load(lesson());
      el.querySelector<HTMLButtonElement>('button[aria-label="Nghe câu 2"]')!.click();
      expect(play).toHaveBeenCalled();
      expect(el.querySelector('audio')?.getAttribute('src')).toBe('/api/audio/l1/2/1');
    });

    it('hides play buttons until audio exists', async () => {
      await load(lesson({ audioStatus: 'running', sentences: [{ index: 0, text: 'Hi.', audioUrl: null }] }));
      expect(el.querySelector('button[aria-label="Nghe câu 1"]')).toBeNull();
      http.expectNone('/api/admin/lessons/l1'); // next poll only after 5 s
    });

    it('shows the failure reason and retries', async () => {
      await load(lesson({ annotationStatus: 'failed', annotationError: 'AI chưa được cấu hình', annotations: [] }));
      expect(el.textContent).toContain('AI chưa được cấu hình');
      expect(button('Chạy lại audio')).toBeUndefined();

      button('Chạy lại chú thích')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/retry?job=annotate').flush({ lesson: lesson({ annotationStatus: 'running', annotations: [] }) });
      await settle();
      // Polling restarts with an immediate reload.
      await load(lesson({ annotationStatus: 'running', annotations: [] }));
      expect(el.textContent).toContain('Chú thích: Đang chạy');
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

    it('links to the edit page and to the Reading step', async () => {
      await load(lesson());
      expect(el.querySelector('a[href="/admin/lessons/l1/edit"]')).not.toBeNull();
      expect(el.querySelector('a[href="/lessons/l1/read"]')?.textContent?.trim()).toBe('Mở bước Đọc');
      expect(el.querySelector('a[href="/lessons/l1/listen"]')?.textContent?.trim()).toBe('Mở bước Nghe');
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
