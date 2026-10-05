import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Lesson } from '../../../core/models/lesson';
import { LessonExtras } from './lesson-extras';

const lesson = (over: Partial<Lesson> = {}): Lesson => ({
  id: 'l1',
  title: 'Park',
  level: 'A1',
  topicId: 't1',
  topicName: 'Gia đình',
  annotationStatus: 'done',
  inRoadmap: false,
  createdAt: '',
  content: 'We went to the park.',
  source: 's',
  license: 'l',
  revision: 1,
  annotationError: '',
  sentences: [],
  annotations: [],
  questions: [
    { prompt: 'Where did we go?', options: ['Park', 'Home', 'School', 'Work'], answerIndex: 0, explanationVi: 'Câu 1.' },
    { prompt: 'Who went?', options: ['We', 'He', 'She', 'They'], answerIndex: 0, explanationVi: 'Câu 1.' },
  ],
  grammarNote: { title: 'Quá khứ đơn', bodyVi: 'Việc đã xong.', examples: ['We went to the park.'] },
  writingPrompt: 'Write about a park.',
  extrasEditedByAdmin: false,
  quizVersion: 1,
  practice: null,
  practiceStatus: 'none',
  practiceError: '',
  ...over,
});

describe('LessonExtras', () => {
  let fixture: ComponentFixture<LessonExtras>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let saved: Lesson[];

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const input = (id: string) => el.querySelector<HTMLInputElement | HTMLTextAreaElement>(`#${id}`)!;
  const type = async (id: string, value: string) => {
    input(id).value = value;
    input(id).dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === label);
  const setup = async (l: Lesson) => {
    await TestBed.configureTestingModule({
      imports: [LessonExtras],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LessonExtras);
    fixture.componentRef.setInput('lesson', l);
    saved = [];
    fixture.componentInstance.saved.subscribe((x) => saved.push(x));
    el = fixture.nativeElement;
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('fills the form from the lesson', async () => {
    await setup(lesson());
    expect(input('q-0-prompt').value).toBe('Where did we go?');
    expect(input('q-1-option-3').value).toBe('They');
    expect((input('q-0-answer-0') as HTMLInputElement).checked).toBe(true);
    expect(input('grammar-title').value).toBe('Quá khứ đơn');
    expect(input('grammar-example-0').value).toBe('We went to the park.');
    expect(input('writing-prompt').value).toBe('Write about a park.');
    expect(el.textContent).not.toContain('Đã sửa tay');
  });

  it('edits, adds and removes, then saves everything', async () => {
    await setup(lesson());
    (input('q-0-answer-2') as HTMLInputElement).click();
    button('Xoá câu hỏi 2')!.click();
    await fixture.whenStable();
    button('Thêm câu hỏi')!.click();
    await fixture.whenStable();
    await type('q-1-prompt', 'What is it?');
    await type('q-1-option-0', 'A park');
    await type('writing-prompt', 'Write more.');
    button('Lưu câu hỏi, ngữ pháp, đề viết')!.click();
    await settle();

    const req = http.expectOne('/api/admin/lessons/l1/extras');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({
      questions: [
        { prompt: 'Where did we go?', options: ['Park', 'Home', 'School', 'Work'], answerIndex: 2, explanationVi: 'Câu 1.' },
        { prompt: 'What is it?', options: ['A park', '', '', ''], answerIndex: -1, explanationVi: '' },
      ],
      grammarNote: { title: 'Quá khứ đơn', bodyVi: 'Việc đã xong.', examples: ['We went to the park.'] },
      writingPrompt: 'Write more.',
    });
    req.flush({ lesson: lesson({ extrasEditedByAdmin: true }) });
    await settle();
    expect(saved.length).toBe(1);
    expect(el.querySelector('[role="status"].save-status')?.textContent?.trim()).toBe('Đã lưu');
  });

  it('stops adding questions at five and examples at three', async () => {
    await setup(lesson());
    for (let i = 0; i < 3; i++) {
      button('Thêm câu hỏi')!.click();
      await fixture.whenStable();
    }
    expect(button('Thêm câu hỏi')!.disabled).toBe(true);
    button('Thêm ví dụ')!.click();
    await fixture.whenStable();
    button('Thêm ví dụ')!.click();
    await fixture.whenStable();
    expect(button('Thêm ví dụ')!.disabled).toBe(true);
  });

  it('removes the grammar note when unchecked', async () => {
    await setup(lesson());
    el.querySelector<HTMLInputElement>('.check input')!.click();
    await fixture.whenStable();
    expect(el.querySelector('#grammar-title')).toBeNull();
    button('Lưu câu hỏi, ngữ pháp, đề viết')!.click();
    await settle();
    const req = http.expectOne('/api/admin/lessons/l1/extras');
    expect(req.request.body.grammarNote).toBeNull();
    req.flush({ lesson: lesson({ grammarNote: null }) });
    await settle();
  });

  it('shows server field errors under the right inputs', async () => {
    await setup(lesson());
    button('Lưu câu hỏi, ngữ pháp, đề viết')!.click();
    await settle();
    http.expectOne('/api/admin/lessons/l1/extras').flush(
      {
        error: 'validation_failed',
        message: 'Thông tin chưa hợp lệ',
        fields: { 'questions.1.options.2': 'Lựa chọn bị trùng', 'grammarNote.examples.0': 'Ví dụ phải có trong bài' },
      },
      { status: 400, statusText: 'Bad Request' },
    );
    await settle();
    expect(input('q-1-option-2').getAttribute('aria-invalid')).toBe('true');
    expect(el.querySelector('#extras-questions-1-options-2-error')?.textContent?.trim()).toBe('Lựa chọn bị trùng');
    expect(input('grammar-example-0').getAttribute('aria-describedby')).toBe('extras-grammarNote-examples-0-error');
    expect(el.querySelector('[role="alert"]')?.textContent?.trim()).toBe('Vui lòng sửa các ô được đánh dấu.');
    expect(saved).toEqual([]);
  });

  it('cannot save while annotation is running and shows the edited tag', async () => {
    await setup(lesson({ annotationStatus: 'running', extrasEditedByAdmin: true }));
    expect(button('Lưu câu hỏi, ngữ pháp, đề viết')!.disabled).toBe(true);
    expect(el.textContent).toContain('Đã sửa tay');
    expect(el.textContent).toContain('Chú thích đang chạy');
  });

  it('shows the flag on the flagged question only and emits confirm (F22)', async () => {
    await setup(lesson());
    const flag = { area: 'question' as const, index: 1, kind: 'ambiguous' as const, noteVi: 'Hai đáp án đều đúng', confirmed: false };
    fixture.componentRef.setInput('flags', [flag]);
    const got: unknown[] = [];
    fixture.componentInstance.confirm.subscribe((f) => got.push(f));
    await fixture.whenStable();
    const notes = el.querySelectorAll('lu-flag-note');
    expect(notes.length).toBe(1);
    expect(notes[0].textContent).toContain('AI thấy câu mơ hồ');
    expect(notes[0].textContent).toContain('Hai đáp án đều đúng');
    expect(el.querySelectorAll('.question')[1].contains(notes[0])).toBe(true);
    el.querySelector<HTMLButtonElement>('button[aria-label="Giữ nguyên câu hỏi 2"]')!.click();
    expect(got).toEqual([flag]);
  });

  it('shows no flag when none is given', async () => {
    await setup(lesson());
    expect(el.querySelector('lu-flag-note')).toBeNull();
  });

  it('offers edit and delete on the flagged question, with a reminder to save', async () => {
    await setup(lesson());
    fixture.componentRef.setInput('flags', [{ area: 'question', index: 1, kind: 'wrong', noteVi: 'Sai', confirmed: false }]);
    await fixture.whenStable();
    const note = el.querySelector('lu-flag-note')!;
    expect(note.textContent).toContain('nhớ bấm Lưu');
    note.querySelector<HTMLButtonElement>('button[aria-label="Sửa câu hỏi 2"]')!.click();
    expect(document.activeElement).toBe(input('q-1-prompt'));
  });

  it('deletes a flagged question after confirming, saving the current form values', async () => {
    await setup(lesson());
    fixture.componentRef.setInput('flags', [{ area: 'question', index: 1, kind: 'wrong', noteVi: 'Sai', confirmed: false }]);
    await fixture.whenStable();
    await type('q-0-prompt', 'Where did they go?');
    el.querySelector<HTMLButtonElement>('button[aria-label="Xoá câu hỏi 2"]')!.click();
    await fixture.whenStable();
    expect(http.match('/api/admin/lessons/l1/extras')).toEqual([]);
    (Array.from(el.querySelectorAll('lu-confirm-dialog button')).find((b) => b.textContent?.trim() === 'Xoá') as HTMLButtonElement).click();
    await settle();
    const r = http.expectOne('/api/admin/lessons/l1/extras');
    expect(r.request.body.questions.map((q: { prompt: string }) => q.prompt)).toEqual(['Where did they go?']);
    r.flush({ lesson: lesson({ questions: [lesson().questions[0]] }) });
    await settle();
    expect(saved.length).toBe(1);
  });
});
