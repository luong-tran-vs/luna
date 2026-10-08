import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Lesson, LessonFlag } from '../../../core/models/lesson';
import { FixHelper } from './fix-helper';

describe('FixHelper', () => {
  let fixture: ComponentFixture<FixHelper>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let applied: Lesson[];

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label));
  const input = (id: string) => el.querySelector<HTMLInputElement | HTMLTextAreaElement>(`#${id}`)!;
  const type = (id: string, value: string) => {
    input(id).value = value;
    input(id).dispatchEvent(new Event('input'));
  };

  const render = async (flag: LessonFlag) => {
    await TestBed.configureTestingModule({
      imports: [FixHelper],
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(FixHelper);
    fixture.componentRef.setInput('lessonId', 'l1');
    fixture.componentRef.setInput('flag', flag);
    fixture.componentRef.setInput('subject', 'câu 2');
    applied = [];
    fixture.componentInstance.applied.subscribe((l) => applied.push(l));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };
  const ask = async (suggestion: object) => {
    button('AI gợi ý sửa')!.click();
    await settle();
    const req = http.expectOne('/api/admin/lessons/l1/check/suggest');
    expect(req.request.method).toBe('POST');
    req.flush({ suggestion });
    await settle();
    return req;
  };

  afterEach(() => http.verify());

  it('asks the AI for the flagged sentence, lets the admin edit it and applies it', async () => {
    await render({ area: 'sentence', index: 1, kind: 'wrong', noteVi: 'Sai thì.', confirmed: false });
    expect(button('AI gợi ý sửa')!.getAttribute('aria-label')).toBe('AI gợi ý sửa câu 2');
    const req = await ask({ area: 'sentence', index: 1, text: 'He gave up smoking.', noteVi: 'Đổi sang quá khứ.' });
    expect(req.request.body).toEqual({ area: 'sentence', index: 1 });
    expect(text(el.querySelector('.fix-note'))).toBe('Đổi sang quá khứ.');
    expect(input('fix-sentence-1-text').value).toBe('He gave up smoking.');
    expect(text(el.querySelector('.fix-warn'))).toContain('chú thích được tạo lại');

    type('fix-sentence-1-text', 'He gave up smoking last year.');
    button('Áp dụng')!.click();
    await settle();
    const apply = http.expectOne('/api/admin/lessons/l1/check/apply');
    expect(apply.request.body).toEqual({ area: 'sentence', index: 1, text: 'He gave up smoking last year.' });
    apply.flush({ lesson: { id: 'l1' } });
    await settle();
    expect(applied).toEqual([{ id: 'l1' }]);
    expect(el.querySelector('form')).toBeNull();
  });

  it('edits a question: prompt, options, the right answer and the explanation', async () => {
    await render({ area: 'question', index: 0, kind: 'mismatch', noteVi: '', confirmed: false });
    await ask({
      area: 'question',
      index: 0,
      question: { prompt: 'Where?', options: ['Park', 'Home'], answerIndex: 0, explanationVi: 'Câu 1.' },
    });
    type('fix-question-0-option-1', 'School');
    el.querySelector<HTMLInputElement>('#fix-question-0-answer-1')!.click();
    button('Áp dụng')!.click();
    await settle();
    const apply = http.expectOne('/api/admin/lessons/l1/check/apply');
    expect(apply.request.body).toEqual({
      area: 'question',
      index: 0,
      question: { prompt: 'Where?', options: ['Park', 'School'], answerIndex: 1, explanationVi: 'Câu 1.' },
    });
    apply.flush({ lesson: { id: 'l1' } });
    await settle();
  });

  it('sends only the fields of a translation or an annotation', async () => {
    await render({ area: 'translation', index: 2, kind: 'wrong', noteVi: '', confirmed: false });
    await ask({ area: 'translation', index: 2, vi: 'Tôi đi.', en: 'I go.' });
    button('Áp dụng')!.click();
    await settle();
    const apply = http.expectOne('/api/admin/lessons/l1/check/apply');
    expect(apply.request.body).toEqual({ area: 'translation', index: 2, vi: 'Tôi đi.', en: 'I go.' });
    apply.flush({ lesson: { id: 'l1' } });
    await settle();
  });

  it('shows why the AI could not help, and why applying failed', async () => {
    await render({ area: 'annotation', index: 0, kind: 'wrong', noteVi: '', confirmed: false });
    button('AI gợi ý sửa')!.click();
    await settle();
    http
      .expectOne('/api/admin/lessons/l1/check/suggest')
      .flush({ error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' }, { status: 429, statusText: 'Too Many' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Đã hết lượt AI, vui lòng thử lại sau.');

    await ask({ area: 'annotation', index: 0, meaningVi: '' });
    button('Áp dụng')!.click();
    await settle();
    http
      .expectOne('/api/admin/lessons/l1/check/apply')
      .flush({ error: 'validation', fields: { 'annotations.0.meaningVi': 'Vui lòng nhập nghĩa' } }, { status: 400, statusText: 'Bad' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Vui lòng nhập nghĩa');
    expect(el.querySelector('form')).not.toBeNull();
    expect(applied).toEqual([]);

    button('Bỏ')!.click();
    await settle();
    expect(el.querySelector('form')).toBeNull();
  });
});
