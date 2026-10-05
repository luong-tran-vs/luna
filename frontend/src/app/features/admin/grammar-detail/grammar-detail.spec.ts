import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarContent, GrammarLessonAdmin } from '../../../core/models/grammar-admin';
import { GrammarDetail } from './grammar-detail';

const content: GrammarContent = {
  objective: 'Bạn có thể giới thiệu bản thân.',
  explanation: ['To be có ba dạng.'],
  usage: ['Nói về tên và nghề.'],
  structures: [{ label: 'Khẳng định', pattern: 'S + am/is/are', example: 'I am a student.' }],
  examples: [{ en: 'I am a student.', vi: 'Tôi là học sinh.' }],
  mistakes: [{ wrong: 'She are a teacher.', right: 'She is a teacher.', noteVi: 'She đi với is.' }],
  practice: [
    { id: 'p1', kind: 'choice', promptVi: 'Chọn', text: 'She ___ a teacher.', options: ['am', 'is', 'are', 'be'], answerIndex: 1, explanationVi: 'is' },
    { id: 'p2', kind: 'fill', text: 'I ___ happy.', answers: ['am', "'m"], explanationVi: 'am' },
  ],
  mastery: [
    { id: 'm1', kind: 'reorder', text: 'Tôi là học sinh.', sentence: 'I am a student', words: ['a', 'I', 'student', 'am', 'is'], explanationVi: 'S + am' },
  ],
};

const lesson = (over: Partial<GrammarLessonAdmin> = {}): GrammarLessonAdmin => ({
  pointId: 'a1-to-be', status: 'draft', edited: false, updatedAt: 't', publishedAt: null, content, checks: [], checkedAt: null, verifiedAt: null, reports: [], ...over,
});

const URL = '/api/admin/grammar-lessons/a1-to-be';

describe('GrammarDetail', () => {
  let fixture: ComponentFixture<GrammarDetail>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label)!;
  const area = () => el.querySelector<HTMLTextAreaElement>('#grammar-json')!;
  const type = async (value: string) => {
    area().value = value;
    area().dispatchEvent(new Event('input'));
    await settle();
  };
  const pointResponse = () =>
    http.expectOne((r) => r.url === '/api/admin/grammar').flush({
      points: [{ id: 'a1-to-be', level: 'A1', titleVi: 'Động từ to be', titleEn: 'to be', pattern: 'S + am/is/are', hintVi: 'Giới thiệu bản thân.', examples: [], lessonCount: 0 }],
    });

  const setup = async (response: { lesson?: GrammarLessonAdmin; status?: number } = { lesson: lesson() }) => {
    TestBed.configureTestingModule({
      imports: [GrammarDetail],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ pointId: 'a1-to-be' }) } } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarDetail);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    pointResponse();
    const req = http.expectOne(URL);
    if (response.lesson) {
      req.flush({ lesson: response.lesson });
    } else {
      req.flush({ error: 'not_found', message: 'Không tìm thấy.' }, { status: response.status ?? 404, statusText: 'x' });
    }
    await settle();
  };

  afterEach(() => http.verify());

  it('shows the point, its status in words and every part of the content', async () => {
    await setup();
    expect(text(el.querySelector('h1'))).toBe('Động từ to be');
    expect(text(el.querySelector('.point'))).toContain('S + am/is/are');
    expect(text(el.querySelector('.point'))).toContain('Giới thiệu bản thân.');
    expect(text(el.querySelector('.status'))).toBe('Bản nháp');
    const preview = text(el.querySelector('.preview'));
    for (const part of ['Mục tiêu', 'Giải thích', 'Cấu trúc', 'Ví dụ', 'Lỗi thường gặp', 'Bài luyện tập (2)', 'Bài kiểm tra (1)']) {
      expect(preview).toContain(part);
    }
    expect(preview).toContain('Bạn có thể giới thiệu bản thân.');
    expect(preview).toContain('She are a teacher.');
  });

  it('marks the right answers with words', async () => {
    await setup();
    const preview = text(el.querySelector('.preview'));
    expect(text(el.querySelector('.correct'))).toBe('is (đáp án đúng)');
    expect(preview).toContain("Đáp án đúng: am / 'm");
    expect(preview).toContain('Câu đúng: I am a student');
  });

  it('puts formatted JSON of the content in the editor', async () => {
    await setup();
    expect(area().value).toBe(JSON.stringify(content, null, 2));
  });

  it('says a syntax error before sending anything', async () => {
    await setup();
    await type('{ "objective": ');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
    expect(text(el.querySelector('.editor .field-error'))).toContain('JSON sai cú pháp');
    expect(area().getAttribute('aria-invalid')).toBe('true');
  });

  it('rejects JSON that is not an object', async () => {
    await setup();
    await type('[1]');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
    expect(text(el.querySelector('.editor .field-error'))).toContain('đối tượng JSON');
  });

  it('reformats valid JSON and leaves broken JSON alone', async () => {
    await setup();
    await type('{"objective":"x"}');
    button('Định dạng lại').click();
    await settle();
    expect(area().value).toBe('{\n  "objective": "x"\n}');
    await type('{oops');
    button('Định dạng lại').click();
    await settle();
    expect(area().value).toBe('{oops');
    expect(text(el.querySelector('.editor .field-error'))).toContain('JSON sai cú pháp');
  });

  it('saves the edited content and shows the new status', async () => {
    await setup();
    await type(JSON.stringify({ ...content, objective: 'Mới' }));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
    const req = http.expectOne(URL);
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ content: { ...content, objective: 'Mới' } });
    req.flush({ lesson: lesson({ edited: true, content: { ...content, objective: 'Mới' } }) });
    await settle();
    expect(text(el.querySelector('.status'))).toBe('Bản nháp · đã sửa tay');
    expect(text(el.querySelector('.preview'))).toContain('Mới');
    expect(JSON.parse(area().value).objective).toBe('Mới');
  });

  it('lists the errors of the server as "đường dẫn: thông báo" under the box', async () => {
    await setup();
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
    http.expectOne(URL).flush(
      { error: 'validation_failed', message: 'Nội dung chưa hợp lệ.', fields: { 'content.practice[2].options': 'cần 4 đáp án', 'content.objective': 'bắt buộc' } },
      { status: 400, statusText: 'x' },
    );
    await settle();
    expect(Array.from(el.querySelectorAll('.field-list li')).map((li) => text(li))).toEqual([
      'content.practice[2].options: cần 4 đáp án',
      'content.objective: bắt buộc',
    ]);
    expect(text(el.querySelector('.editor [role="alert"]'))).toBe('Nội dung chưa hợp lệ.');
    expect(area().value).toBe(JSON.stringify(content, null, 2));
  });

  it('publishes a draft and unpublishes it again', async () => {
    await setup();
    button('Đăng').click();
    await settle();
    const publish = http.expectOne(`${URL}/publish`);
    expect(publish.request.method).toBe('POST');
    publish.flush({ lesson: lesson({ status: 'published', publishedAt: 't' }) });
    await settle();
    expect(text(el.querySelector('.status'))).toBe('Đã đăng');
    button('Gỡ').click();
    await settle();
    http.expectOne(`${URL}/unpublish`).flush({ lesson: lesson() });
    await settle();
    expect(text(el.querySelector('.status'))).toBe('Bản nháp');
  });

  it('regenerates a clean draft without asking', async () => {
    await setup();
    button('Sinh lại bằng AI').click();
    await settle();
    const req = http.expectOne(`${URL}/generate`);
    expect(req.request.body).toEqual({ force: false });
    req.flush({ lesson: lesson() });
    await settle();
    expect(text(el.querySelector('.banner-note'))).toContain('Đã sinh bản nháp mới');
  });

  it('asks before regenerating a published lesson, then forces', async () => {
    await setup({ lesson: lesson({ status: 'published', publishedAt: 't' }) });
    button('Sinh lại bằng AI').click();
    await settle();
    http.expectNone(`${URL}/generate`);
    expect(text(el.querySelector('lu-confirm-dialog'))).toContain('đã đăng');
    el.querySelector<HTMLButtonElement>('lu-confirm-dialog .btn-primary')!.click();
    await settle();
    const req = http.expectOne(`${URL}/generate`);
    expect(req.request.body).toEqual({ force: true });
    req.flush({ lesson: lesson() });
    await settle();
  });

  it('explains an AI failure', async () => {
    await setup();
    button('Sinh lại bằng AI').click();
    await settle();
    http.expectOne(`${URL}/generate`).flush({ error: 'ai_quota' }, { status: 429, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('.form-alert'))).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
  });

  it('shows an empty page with Sinh bằng AI when the point has no lesson', async () => {
    await setup({ status: 404 });
    expect(text(el.querySelector('.empty'))).toContain('chưa có bài ngữ pháp');
    button('Sinh bằng AI').click();
    await settle();
    const req = http.expectOne(`${URL}/generate`);
    expect(req.request.body).toEqual({ force: false });
    req.flush({ lesson: lesson() }, { status: 201, statusText: 'Created' });
    await settle();
    expect(el.querySelector('.preview')).not.toBeNull();
    expect(text(el.querySelector('.status'))).toBe('Bản nháp');
  });

  describe('quality control (F21)', () => {
    const flagged = () =>
      lesson({
        checkedAt: '2026-10-05T08:30:00Z',
        checks: [
          { exerciseId: 'p1', kind: 'mismatch', noteVi: 'AI chọn "are".', confirmed: false },
          { exerciseId: 'm1', kind: 'ambiguous', noteVi: 'Hai cách ghép.', confirmed: false },
        ],
      });

    it('says the lesson is not checked yet', async () => {
      await setup();
      expect(text(el.querySelector('.check-status'))).toBe('Chưa kiểm tra');
    });

    it('says how many exercises need a look, and labels them in words', async () => {
      await setup({ lesson: flagged() });
      expect(text(el.querySelector('.check-status'))).toBe('2 câu cần xem');
      expect(text(el.querySelector('.check-strip'))).toContain('2026');
      const flags = Array.from(el.querySelectorAll('.flag')).map((n) => text(n));
      expect(flags).toEqual(['Cần xem: AI giải ra đáp án khác. AI chọn "are".', 'Cần xem: AI thấy câu mơ hồ. Hai cách ghép.']);
    });

    it('says a clean check', async () => {
      await setup({ lesson: lesson({ checkedAt: '2026-10-05T08:30:00Z' }) });
      expect(text(el.querySelector('.check-status'))).toBe('Đã kiểm tra, không có câu nào bị gắn cờ');
      expect(el.querySelector('.flag')).toBeNull();
    });

    it('checks with the AI and shows the result', async () => {
      await setup();
      button('Kiểm tra bằng AI').click();
      await settle();
      expect(text(button('Đang kiểm tra…'))).toBe('Đang kiểm tra…');
      const req = http.expectOne(`${URL}/check`);
      expect(req.request.method).toBe('POST');
      req.flush({ lesson: flagged() });
      await settle();
      expect(text(el.querySelector('.check-status'))).toBe('2 câu cần xem');
      expect(text(el.querySelector('.banner-note'))).toContain('Đã kiểm tra bằng AI');
    });

    for (const [code, message] of [
      [503, 'AI chưa được cấu hình. Liên hệ người vận hành.'],
      [429, 'Đã hết lượt AI, vui lòng thử lại sau.'],
      [502, 'AI trả về nội dung không dùng được, vui lòng thử lại.'],
    ] as const) {
      it(`explains a ${code} from the check`, async () => {
        await setup();
        button('Kiểm tra bằng AI').click();
        await settle();
        http.expectOne(`${URL}/check`).flush({ error: 'x' }, { status: code, statusText: 'x' });
        await settle();
        expect(text(el.querySelector('.form-alert'))).toBe(message);
        expect(text(el.querySelector('.check-status'))).toBe('Chưa kiểm tra');
      });
    }

    it('asks before publishing with flags, then acknowledges them', async () => {
      await setup({ lesson: flagged() });
      button('Đăng').click();
      await settle();
      http.expectNone(`${URL}/publish`);
      const dialog = el.querySelectorAll('lu-confirm-dialog')[1];
      expect(text(dialog)).toContain('2 câu');
      dialog.querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await settle();
      const req = http.expectOne(`${URL}/publish`);
      expect(req.request.body).toEqual({ acknowledgeFlags: true });
      req.flush({ lesson: { ...flagged(), status: 'published', publishedAt: 't' } });
      await settle();
      expect(text(el.querySelector('.status'))).toBe('Đã đăng');
    });

    it('publishes without asking when nothing is flagged', async () => {
      await setup({ lesson: lesson({ checkedAt: '2026-10-05T08:30:00Z' }) });
      button('Đăng').click();
      await settle();
      const req = http.expectOne(`${URL}/publish`);
      expect(req.request.body).toBeNull();
      req.flush({ lesson: lesson({ status: 'published', publishedAt: 't' }) });
      await settle();
    });

    it('shows the message of a 409 grammar_flags_unresolved', async () => {
      await setup({ lesson: lesson() });
      button('Đăng').click();
      await settle();
      http
        .expectOne(`${URL}/publish`)
        .flush({ error: 'grammar_flags_unresolved', message: 'Còn 3 câu bị gắn cờ.', count: 3 }, { status: 409, statusText: 'x' });
      await settle();
      expect(text(el.querySelector('.form-alert'))).toBe('Còn 3 câu bị gắn cờ.');
      expect(text(el.querySelector('.status'))).toBe('Bản nháp');
    });

    it('falls back to a clear message for a 409 without text', async () => {
      await setup({ lesson: lesson() });
      button('Đăng').click();
      await settle();
      http.expectOne(`${URL}/publish`).flush({ error: 'grammar_flags_unresolved' }, { status: 409, statusText: 'x' });
      await settle();
      expect(text(el.querySelector('.form-alert'))).toContain('gắn cờ');
    });

    it('clears the flags after saving the content', async () => {
      await setup({ lesson: flagged() });
      el.querySelector('form')!.dispatchEvent(new Event('submit'));
      await settle();
      http.expectOne(URL).flush({ lesson: lesson({ edited: true }) });
      await settle();
      expect(text(el.querySelector('.check-status'))).toBe('Chưa kiểm tra');
      expect(el.querySelector('.flag')).toBeNull();
    });

    it('shows learner reports and closes them in place', async () => {
      const withReports = lesson({
        reports: [{ exerciseId: 'p2', count: 3, reasons: { wrong_answer: 2, typo: 1 }, notes: ['Phải là is'] }],
      });
      await setup({ lesson: withReports });
      const box = text(el.querySelector('.reports'));
      expect(box).toContain('3 báo lỗi từ người học');
      expect(box).toContain('Đáp án sai: 2 · Lỗi chính tả: 1');
      expect(box).toContain('“Phải là is”');
      el.querySelector<HTMLButtonElement>('.reports button')!.click();
      await settle();
      const req = http.expectOne('/api/admin/grammar-reports/resolve');
      expect(req.request.body).toEqual({ pointId: 'a1-to-be', exerciseId: 'p2' });
      req.flush({ resolved: 3 });
      await settle();
      expect(el.querySelector('.reports')).toBeNull();
    });

    const clickIn = async (root: Element | null, label: string) => {
      const target = Array.from(root!.querySelectorAll<HTMLElement>('button')).find((b) => text(b) === label)!;
      target.click();
      await settle();
    };
    const exercise = (id: string) => el.querySelector(`#ex-${id}`)!;

    it('labels an exercise the AI did not answer', async () => {
      await setup({ lesson: lesson({ checkedAt: '2026-10-05T08:30:00Z', checks: [{ exerciseId: 'p2', kind: 'unchecked', noteVi: 'AI không trả lời câu này, hãy tự kiểm tra.', confirmed: false }] }) });
      expect(text(exercise('p2').querySelector('.flag'))).toBe('Chưa kiểm tra được: AI không trả lời câu này. AI không trả lời câu này, hãy tự kiểm tra.');
    });

    it('confirms a flag: the line turns into "Đã xác nhận" and keeps the note', async () => {
      await setup({ lesson: flagged() });
      await clickIn(exercise('p1'), 'Xác nhận đúng');
      const req = http.expectOne(`${URL}/checks/p1/confirm`);
      expect(req.request.method).toBe('POST');
      const l = flagged();
      l.checks[0].confirmed = true;
      req.flush({ lesson: l });
      await settle();
      expect(text(exercise('p1').querySelector('.flag'))).toBe('Đã xác nhận. AI chọn "are".');
      expect(text(el.querySelector('.check-status'))).toBe('1 câu cần xem');
      expect(Array.from(exercise('p1').querySelectorAll('button')).map((b) => text(b))).toEqual(['Sửa câu này', 'Xoá câu này']);
    });

    it('lists the open flags and jumps to the exercise', async () => {
      await setup({ lesson: flagged() });
      const items = Array.from(el.querySelectorAll('.open-flags li')).map((li) => text(li));
      expect(items).toEqual(['Luyện tập 1 · AI giải ra đáp án khác', 'Kiểm tra 1 · AI thấy câu mơ hồ']);
      el.querySelector<HTMLButtonElement>('.open-flags button')!.click();
      await settle();
      expect(document.activeElement).toBe(exercise('p1'));
    });

    it('offers edit and delete inside "Thao tác" for an exercise without a flag', async () => {
      await setup();
      const more = exercise('p2').querySelector('details')!;
      expect(text(more.querySelector('summary'))).toBe('Thao tác');
      expect(Array.from(more.querySelectorAll('button')).map((b) => text(b))).toEqual(['Sửa câu này', 'Xoá câu này']);
    });

    it('edits a choice exercise in place and reminds to check again', async () => {
      await setup({ lesson: flagged() });
      await clickIn(exercise('p1'), 'Sửa câu này');
      const q = (sel: string) => exercise('p1').querySelector<HTMLInputElement>(sel)!;
      expect((q('#edit-text') as unknown as HTMLTextAreaElement).value).toBe('She ___ a teacher.');
      expect(q('input[type="radio"]:checked').getAttribute('aria-label')).toContain('Lựa chọn 2');
      const textArea = q('#edit-text');
      textArea.value = 'He ___ a teacher.';
      textArea.dispatchEvent(new Event('input'));
      const third = exercise('p1').querySelectorAll<HTMLInputElement>('input[type="radio"]')[2];
      third.click();
      await settle();
      await clickIn(exercise('p1'), 'Lưu câu');
      const req = http.expectOne(`${URL}/exercises/p1`);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual({
        kind: 'choice', promptVi: 'Chọn', text: 'He ___ a teacher.', options: ['am', 'is', 'are', 'be'], answerIndex: 2, explanationVi: 'is',
      });
      const updated = lesson({ checkedAt: '2026-10-05T08:30:00Z', checks: [flagged().checks[1]] });
      updated.content = { ...content, practice: [{ ...content.practice[0], text: 'He ___ a teacher.', answerIndex: 2 }, content.practice[1]] };
      req.flush({ lesson: updated });
      await settle();
      expect(exercise('p1').querySelector('form')).toBeNull();
      expect(text(exercise('p1'))).toContain('He ___ a teacher.');
      expect(exercise('p1').querySelector('.flag')).toBeNull();
      expect(text(el.querySelector('.banner-note'))).toContain('Nên bấm Kiểm tra bằng AI lại');
    });

    it('edits a fill exercise: one answer per line', async () => {
      await setup();
      await clickIn(exercise('p2'), 'Sửa câu này');
      const answers = exercise('p2').querySelector<HTMLTextAreaElement>('#edit-answers')!;
      expect(answers.value).toBe("am\n'm");
      answers.value = "am\n'm\n  \nI am";
      answers.dispatchEvent(new Event('input'));
      await settle();
      await clickIn(exercise('p2'), 'Lưu câu');
      const req = http.expectOne(`${URL}/exercises/p2`);
      expect(req.request.body).toEqual({ kind: 'fill', text: 'I ___ happy.', answers: ['am', "'m", 'I am'], explanationVi: 'am' });
      req.flush({ lesson: lesson() });
      await settle();
    });

    it('edits a reorder exercise: words split on spaces or lines', async () => {
      await setup();
      await clickIn(exercise('m1'), 'Sửa câu này');
      const words = exercise('m1').querySelector<HTMLTextAreaElement>('#edit-words')!;
      expect(words.value).toBe('a I student am is');
      words.value = 'a\nI  student\nam';
      words.dispatchEvent(new Event('input'));
      await settle();
      await clickIn(exercise('m1'), 'Lưu câu');
      const req = http.expectOne(`${URL}/exercises/m1`);
      expect(req.request.body).toEqual({
        kind: 'reorder', text: 'Tôi là học sinh.', sentence: 'I am a student', words: ['a', 'I', 'student', 'am'], explanationVi: 'S + am',
      });
      req.flush({ lesson: lesson() });
      await settle();
    });

    it('shows a 400 per field and keeps the form open', async () => {
      await setup();
      await clickIn(exercise('p2'), 'Sửa câu này');
      await clickIn(exercise('p2'), 'Lưu câu');
      http.expectOne(`${URL}/exercises/p2`).flush(
        { error: 'validation_failed', message: 'Câu chưa hợp lệ.', fields: { 'exercise.answers': 'cần ít nhất một đáp án', 'exercise.text': 'cần có ___' } },
        { status: 400, statusText: 'x' },
      );
      await settle();
      const form = exercise('p2').querySelector('form')!;
      expect(text(form.querySelector('[role="alert"]'))).toBe('Câu chưa hợp lệ.');
      expect(text(form.querySelector('#edit-answers')!.parentElement!.querySelector('.field-error'))).toBe('cần ít nhất một đáp án');
      expect(text(form.querySelector('#edit-text')!.parentElement!.querySelector('.field-error'))).toBe('cần có ___');
    });

    it('cancels an edit without calling the server', async () => {
      await setup();
      await clickIn(exercise('p2'), 'Sửa câu này');
      await clickIn(exercise('p2'), 'Huỷ');
      expect(exercise('p2').querySelector('form')).toBeNull();
    });

    it('asks before deleting, says how many are left, then deletes', async () => {
      await setup();
      await clickIn(exercise('p2'), 'Xoá câu này');
      http.expectNone(`${URL}/exercises/p2`);
      const dialog = el.querySelectorAll('lu-confirm-dialog')[3];
      expect(text(dialog)).toContain('còn 1 câu');
      dialog.querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await settle();
      const req = http.expectOne(`${URL}/exercises/p2`);
      expect(req.request.method).toBe('DELETE');
      const after = lesson();
      after.content = { ...content, practice: [content.practice[0]] };
      req.flush({ lesson: after });
      await settle();
      expect(el.querySelector('#ex-p2')).toBeNull();
    });

    it('shows the message of the server when deleting goes under the minimum', async () => {
      await setup();
      await clickIn(exercise('p2'), 'Xoá câu này');
      el.querySelectorAll('lu-confirm-dialog')[3].querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await settle();
      http.expectOne(`${URL}/exercises/p2`).flush(
        { error: 'validation_failed', message: 'Không xoá được.', fields: { 'content.practice': 'cần ít nhất 6 bài' } },
        { status: 400, statusText: 'x' },
      );
      await settle();
      expect(text(el.querySelector('.form-alert'))).toBe('Không xoá được. content.practice: cần ít nhất 6 bài');
      expect(el.querySelector('#ex-p2')).not.toBeNull();
    });

    it('keeps "Xác nhận đã kiểm tra xong" off until the lesson was checked', async () => {
      await setup();
      expect(button('Xác nhận đã kiểm tra xong').disabled).toBe(true);
    });

    it('verifies a clean checked lesson at once', async () => {
      await setup({ lesson: lesson({ checkedAt: '2026-10-05T08:30:00Z' }) });
      button('Xác nhận đã kiểm tra xong').click();
      await settle();
      const req = http.expectOne(`${URL}/verify`);
      expect(req.request.method).toBe('POST');
      req.flush({ lesson: lesson({ checkedAt: '2026-10-05T08:30:00Z', verifiedAt: '2026-10-05T09:00:00Z' }) });
      await settle();
      expect(text(el.querySelector('.verified-line'))).toContain('Đã xác nhận bởi quản trị');
      expect(button('Đã xác nhận').disabled).toBe(true);
    });

    it('asks before verifying while flags are left', async () => {
      await setup({ lesson: flagged() });
      button('Xác nhận đã kiểm tra xong').click();
      await settle();
      http.expectNone(`${URL}/verify`);
      const dialog = el.querySelectorAll('lu-confirm-dialog')[2];
      expect(text(dialog)).toContain('Xác nhận 2 câu còn cờ là đúng?');
      dialog.querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await settle();
      http.expectOne(`${URL}/verify`).flush({ lesson: { ...flagged(), verifiedAt: '2026-10-05T09:00:00Z' } });
      await settle();
      expect(el.querySelector('.verified-line')).not.toBeNull();
    });

    it('explains a 409 grammar_not_checked', async () => {
      await setup({ lesson: lesson({ checkedAt: '2026-10-05T08:30:00Z' }) });
      button('Xác nhận đã kiểm tra xong').click();
      await settle();
      http.expectOne(`${URL}/verify`).flush({ error: 'grammar_not_checked', message: 'Bài chưa được kiểm tra.' }, { status: 409, statusText: 'x' });
      await settle();
      expect(text(el.querySelector('.form-alert'))).toBe('Bài chưa được kiểm tra.');
    });

    it('publishes straight away when every flag is confirmed', async () => {
      const l = flagged();
      l.checks.forEach((c) => (c.confirmed = true));
      await setup({ lesson: l });
      expect(text(el.querySelector('.check-status'))).toBe('Đã kiểm tra, các câu bị gắn cờ đã được xác nhận');
      button('Đăng').click();
      await settle();
      const req = http.expectOne(`${URL}/publish`);
      expect(req.request.body).toBeNull();
      req.flush({ lesson: { ...l, status: 'published', publishedAt: 't' } });
      await settle();
    });

    it('keeps the reports when closing them fails', async () => {
      await setup({ lesson: lesson({ reports: [{ exerciseId: 'p2', count: 1, reasons: { other: 1 }, notes: [] }] }) });
      el.querySelector<HTMLButtonElement>('.reports button')!.click();
      await settle();
      http.expectOne('/api/admin/grammar-reports/resolve').flush({}, { status: 500, statusText: 'x' });
      await settle();
      expect(el.querySelector('.reports')).not.toBeNull();
      expect(text(el.querySelector('.form-alert'))).toContain('Không đánh dấu được');
    });
  });

  it('offers a retry when loading fails', async () => {
    await setup({ status: 500 });
    expect(text(el.querySelector('.load-error'))).toContain('Không tải được bài ngữ pháp.');
    button('Thử lại').click();
    await settle();
    http.expectOne(URL).flush({ lesson: lesson() });
    await settle();
    expect(el.querySelector('.preview')).not.toBeNull();
  });
});
