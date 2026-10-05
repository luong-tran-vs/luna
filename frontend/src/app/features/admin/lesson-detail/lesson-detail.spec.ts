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

    it('shows the grammar point only when the lesson has one', async () => {
      await load(lesson({ grammarPointId: 'b1-a', grammarPointTitle: 'Thì quá khứ đơn' }));
      expect(el.textContent).toContain('Điểm ngữ pháp: Thì quá khứ đơn');
    });

    it('hides the grammar line when none is assigned', async () => {
      await load(lesson({ grammarPointId: '', grammarPointTitle: '' }));
      expect(el.textContent).not.toContain('Điểm ngữ pháp:');
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

  describe('AI check (F22)', () => {
    type Flag = NonNullable<Lesson['review']>['flags'][number];
    const review = (flags: Flag[], verifiedAt: string | null = null) => ({
      checkedAt: '2026-10-05T08:00:00Z',
      verifiedAt,
      flags,
    });
    const flag = (area: Flag['area'], index: number, kind: Flag['kind'] = 'wrong', confirmed = false): Flag => ({
      area, index, kind, noteVi: 'Ghi chú ' + area, confirmed,
    });
    const err = () => el.querySelector('[role="alert"]')?.textContent ?? '';
    const withContent = {
      questions: [
        { prompt: 'Q1?', options: ['a', 'b', 'c', 'd'], answerIndex: 0, explanationVi: 'x' },
        { prompt: 'Q2?', options: ['a', 'b', 'c', 'd'], answerIndex: 1, explanationVi: 'y' },
      ],
      practice: {
        objectiveVi: 'm', examples: [], dialogue: null, grammarTipVi: 't',
        translations: [{ vi: 'Xin chào', en: 'Hello', distractors: [] }],
      },
      practiceStatus: 'done' as const,
    };

    it('starts unchecked and runs the check, showing flags at the right spots', async () => {
      await load(lesson({ ...withContent, review: null }));
      expect(el.querySelector('.check-strip')?.textContent).toContain('Chưa kiểm tra');
      button('Kiểm tra bằng AI')!.click();
      await settle();
      expect(button('Đang kiểm tra…')!.disabled).toBe(true);
      const r = http.expectOne('/api/admin/lessons/l1/check');
      expect(r.request.method).toBe('POST');
      r.flush({
        lesson: lesson({
          ...withContent,
          review: review([flag('sentence', 1, 'ambiguous'), flag('annotation', 0, 'wrong'), flag('question', 1, 'mismatch'), flag('translation', 0, 'unchecked')]),
        }),
      });
      await settle();
      expect(el.querySelector('.check-strip')?.textContent).toContain('4 chỗ cần xem');
      expect(button('Kiểm tra lại')).toBeTruthy();
      const sentences = Array.from(el.querySelectorAll('.sentences li'));
      expect(sentences[0].textContent).not.toContain('AI thấy câu mơ hồ');
      expect(sentences[1].textContent).toContain('AI thấy câu mơ hồ');
      expect(el.querySelector('.annotations li')?.textContent).toContain('AI thấy có thể sai');
      expect(el.querySelector('lu-lesson-extras')?.textContent).toContain('Cần xem: AI giải ra đáp án khác');
      expect(el.querySelector('lu-practice-section')?.textContent).toContain('AI chưa kiểm tra được câu này');
    });

    it('disables the check until annotation is done', async () => {
      await load(lesson({ annotationStatus: 'failed' }));
      expect(button('Kiểm tra bằng AI')!.disabled).toBe(true);
      expect(el.textContent).toContain('Cần chú thích xong trước');
    });

    it('says nothing is flagged and lets the admin verify', async () => {
      await load(lesson({ review: review([]) }));
      expect(el.querySelector('.check-strip')?.textContent).toContain('Đã kiểm tra, không có chỗ nào bị gắn cờ');
      button('Xác nhận đã kiểm tra xong')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/check/verify').flush({ lesson: lesson({ review: review([], '2026-10-05T09:00:00Z') }) });
      await settle();
      expect(el.querySelector('.check-strip')?.textContent).toContain('Đã xác nhận kiểm tra xong');
      expect(button('Xác nhận đã kiểm tra xong')).toBeUndefined();
    });

    for (const [status, text] of [
      [503, 'AI chưa được cấu hình'],
      [429, 'Đã hết lượt AI'],
      [502, 'AI trả về nội dung không dùng được'],
    ] as const) {
      it('shows a clear message for ' + status, async () => {
        await load(lesson());
        button('Kiểm tra bằng AI')!.click();
        await settle();
        http.expectOne('/api/admin/lessons/l1/check').flush({ error: 'x' }, { status, statusText: 'E' });
        await settle();
        expect(err()).toContain(text);
        expect(button('Kiểm tra bằng AI')!.disabled).toBe(false);
      });
    }

    it('shows the annotation_not_done conflict', async () => {
      await load(lesson());
      button('Kiểm tra bằng AI')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/check').flush({ error: 'annotation_not_done' }, { status: 409, statusText: 'Conflict' });
      await settle();
      expect(err()).toContain('Chú thích chưa chạy xong');
    });

    it('keeps a flag as it is and relabels it', async () => {
      await load(lesson({ review: review([flag('sentence', 0, 'wrong'), flag('sentence', 1, 'wrong')]) }));
      el.querySelector<HTMLButtonElement>('button[aria-label="Giữ nguyên câu 1"]')!.click();
      await settle();
      const r = http.expectOne('/api/admin/lessons/l1/check/confirm');
      expect(r.request.body).toEqual({ area: 'sentence', index: 0 });
      r.flush({ lesson: lesson({ review: review([flag('sentence', 0, 'wrong', true), flag('sentence', 1, 'wrong')]) }) });
      await settle();
      expect(el.querySelector('.sentences li')?.textContent).toContain('Đã xem, giữ nguyên');
      expect(el.querySelector('.check-strip')?.textContent).toContain('1 chỗ cần xem');
    });

    it('blocks verifying while flags are open', async () => {
      await load(lesson({ review: review([flag('sentence', 0, 'wrong')]) }));
      expect(button('Xác nhận đã kiểm tra xong')!.disabled).toBe(true);
      expect(el.textContent).toContain('Còn 1 chỗ chưa xác nhận');
    });

    it('shows the count when the server answers flags_unresolved', async () => {
      await load(lesson({ review: review([flag('sentence', 0, 'wrong', true)]) }));
      button('Xác nhận đã kiểm tra xong')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/check/verify').flush({ error: 'flags_unresolved', count: 3 }, { status: 409, statusText: 'Conflict' });
      await settle();
      expect(err()).toContain('Còn 3 chỗ bị AI gắn cờ chưa xác nhận');
    });

    it('shows not_checked from verify', async () => {
      await load(lesson({ review: review([]) }));
      button('Xác nhận đã kiểm tra xong')!.click();
      await settle();
      http.expectOne('/api/admin/lessons/l1/check/verify').flush({ error: 'not_checked' }, { status: 409, statusText: 'Conflict' });
      await settle();
      expect(err()).toContain('chưa được kiểm tra bằng AI');
    });

    describe('fixing flagged spots', () => {
      const flagged = (flags: Flag[]) => lesson({ ...withContent, review: review(flags) });
      const dialogYes = () =>
        (Array.from(el.querySelectorAll('lu-confirm-dialog button')).find((b) => b.textContent?.trim() === 'Xoá') as HTMLButtonElement).click();

      it('links a flagged sentence to the lesson editor', async () => {
        await load(flagged([flag('sentence', 1)]));
        const link = el.querySelector<HTMLAnchorElement>('.sentences lu-flag-note a')!;
        expect(link.textContent?.trim()).toBe('Sửa nội dung bài');
        expect(link.getAttribute('href')).toBe('/admin/lessons/l1/edit');
        expect(el.querySelector('.sentences lu-flag-note')?.textContent).toContain('chạy lại chú thích và xoá kết quả kiểm tra');
      });

      it('edits a flagged annotation in place and keeps its flag visible while editing', async () => {
        await load(flagged([flag('annotation', 1)]));
        el.querySelector<HTMLButtonElement>('lu-flag-note button[aria-label="Sửa chú thích gave up"]')!.click();
        await settle();
        expect(el.querySelector('.editor')).not.toBeNull();
        expect(document.activeElement?.id).toBe('ann-meaning-1');
        const rows = el.querySelectorAll('.annotation-edit');
        expect(rows[0].querySelector('lu-flag-note')).toBeNull();
        expect(rows[1].querySelector('lu-flag-note')?.textContent).toContain('AI thấy có thể sai');
      });

      it('deletes a flagged annotation after confirming', async () => {
        await load(flagged([flag('annotation', 0)]));
        el.querySelector<HTMLButtonElement>('lu-flag-note button[aria-label="Xoá chú thích went khỏi bài"]')!.click();
        await settle();
        expect(http.match('/api/admin/lessons/l1/annotations')).toEqual([]);
        dialogYes();
        await settle();
        const r = http.expectOne('/api/admin/lessons/l1/annotations');
        expect(r.request.method).toBe('PUT');
        expect(r.request.body).toEqual({ annotations: [{ text: 'gave up', lemma: 'give up', meaningVi: 'bỏ' }] });
        r.flush({ lesson: { ...flagged([]), annotations: flagged([]).annotations.slice(1) } });
        await settle();
        expect(el.querySelector('.annotations')?.textContent).not.toContain('went');
      });

      it('deleting from the editor keeps the other rows as typed', async () => {
        await load(flagged([flag('annotation', 0)]));
        button('Sửa chú thích')!.click();
        await settle();
        const meaning = el.querySelector<HTMLInputElement>('#ann-meaning-1')!;
        meaning.value = 'từ bỏ';
        meaning.dispatchEvent(new Event('input'));
        el.querySelector<HTMLButtonElement>('lu-flag-note button[aria-label="Xoá chú thích went khỏi bài"]')!.click();
        await settle();
        dialogYes();
        await settle();
        expect(http.expectOne('/api/admin/lessons/l1/annotations').request.body).toEqual({ annotations: [{ text: 'gave up', lemma: 'give up', meaningVi: 'từ bỏ' }] });
      });

      it('shows edit and delete on a flagged question and translation', async () => {
        await load(flagged([flag('question', 0), flag('translation', 0)]));
        expect(el.querySelector('lu-lesson-extras lu-flag-note')?.textContent).toContain('Sửa câu này');
        expect(el.querySelector('lu-lesson-extras lu-flag-note')?.textContent).toContain('Xoá câu này');
        expect(el.querySelector('lu-practice-section lu-flag-note')?.textContent).toContain('Sửa');
        expect(el.querySelector('lu-practice-section lu-flag-note')?.textContent).toContain('Xoá');
      });
    });

  });
});
