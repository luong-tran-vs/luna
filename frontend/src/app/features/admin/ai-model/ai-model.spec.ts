import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AiModelState } from '../../../core/models/ai-model';
import { advisedOps, AiModel, modelName, opAdvice, opGroups, testText } from './ai-model';

const OPTIONS = [
  'gemini-3.5-flash-lite',
  'gemini-3.5-flash',
  'gemini-3.6-flash',
  'gemini-3.7-flash',
  'gemini-3.8-flash',
];
const OPS = ['explain', 'talk', 'generate', 'curriculum_draft'];

const state = (over: Partial<AiModelState> = {}): AiModelState => ({
  model: 'gemini-3.5-flash-lite',
  default: 'gemini-3.5-flash-lite',
  saved: '',
  updatedAt: null,
  options: OPTIONS,
  ops: OPS,
  byOp: {},
  ...over,
});

describe('ai model helpers', () => {
  it('names the models and says how a test went', () => {
    expect(modelName('gemini-3.6-flash')).toBe('Gemini 3.6 Flash');
    expect(modelName('gemini-9')).toBe('gemini-9');
    expect(modelName('openai/gpt-oss-120b')).toBe('GPT-OSS 120B · Groq');
    expect(modelName('gpt-6-sol')).toBe('GPT-6 Sol · ChatGPT');
    expect(testText({ outcome: 'ok', durationMs: 4800 })).toBe('Trả lời sau 4,8 giây');
    expect(testText({ outcome: 'busy', durationMs: 3000 })).toBe('Đang quá tải (503) sau 3 giây');
    expect(testText({ outcome: 'timeout', durationMs: 60000 })).toBe('Không trả lời trong 60 giây');
  });
});

describe('op advice', () => {
  it('says which provider suits an op and gives the advised models', () => {
    expect(opAdvice('explain')).toBe('Nên dùng Groq: người học đang chờ câu trả lời');
    expect(opAdvice('curriculum_draft')).toBe('Nên dùng Gemini: cần hiểu khung CEFR');
    expect(opAdvice('unknown')).toBe('');
    expect(advisedOps(OPS, [...OPTIONS, 'openai/gpt-oss-120b'])).toEqual({
      explain: 'openai/gpt-oss-120b',
      talk: 'openai/gpt-oss-120b',
    });
    // Without the Groq model every op follows the model of every op.
    expect(advisedOps(OPS, OPTIONS)).toEqual({});
  });
});

describe('op groups', () => {
  it('lists the ops a learner waits on first', () => {
    expect(opGroups(OPS)).toEqual([
      { title: 'Người học đang chờ', ops: ['explain', 'talk'] },
      { title: 'Soạn nội dung và việc chạy nền', ops: ['generate', 'curriculum_draft'] },
    ]);
  });
});

describe('AiModel', () => {
  let fixture: ComponentFixture<AiModel>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await fixture.whenStable();
    fixture.detectChanges();
  };
  const radios = () => Array.from(el.querySelectorAll<HTMLInputElement>('input[type="radio"]'));
  const button = (text: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      b.textContent?.trim().startsWith(text),
    );

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(AiModel);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('lists the models with the one in use ticked, and saves another', async () => {
    http.expectOne('/admin/ai-model').flush(state());
    await settle();

    expect(radios().map((r) => r.value)).toEqual(OPTIONS);
    expect(radios()[0].checked).toBe(true);
    expect(el.querySelector('.current')?.textContent).toContain('Mặc định theo GEMINI_MODEL');
    expect(button('Lưu')?.disabled).toBe(true);
    expect(button('Về mặc định')).toBeUndefined();

    radios()[2].click();
    await settle();
    expect(button('Lưu')?.disabled).toBe(false);
    button('Lưu')?.click();
    const req = http.expectOne('/admin/ai-model');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ model: 'gemini-3.6-flash', byOp: {} });
    req.flush(
      state({
        model: 'gemini-3.6-flash',
        saved: 'gemini-3.6-flash',
        updatedAt: '2026-10-23T09:00:00Z',
      }),
    );
    await settle();

    expect(el.querySelector('.status')?.textContent).toContain('Model chung: Gemini 3.6 Flash.');
    expect(el.querySelector('.option-selected .tag')?.textContent).toContain('Đang dùng');

    button('Về mặc định')?.click();
    const reset = http.expectOne('/admin/ai-model');
    expect(reset.request.body).toEqual({ model: '', byOp: {} });
    reset.flush(state());
    await settle();
    expect(button('Về mặc định')).toBeUndefined();
  });

  it('tests a model and says how it went', async () => {
    http.expectOne('/admin/ai-model').flush(state());
    await settle();

    el.querySelector<HTMLButtonElement>('button[aria-label="Thử Gemini 3.8 Flash"]')?.click();
    await settle();
    const req = http.expectOne('/admin/ai-model/test');
    expect(req.request.body).toEqual({ model: 'gemini-3.8-flash' });
    req.flush({ outcome: 'ok', durationMs: 2500 });
    await settle();
    expect(el.querySelectorAll('.test-result')[4].textContent).toContain('Trả lời sau 2,5 giây');
  });

  it('gives an op its own model and saves it with the model of every op', async () => {
    http
      .expectOne('/admin/ai-model')
      .flush(state({ options: [...OPTIONS, 'openai/gpt-oss-120b'] }));
    await settle();

    expect(el.querySelector('#hint-explain')?.textContent).toBe(
      'Nên dùng Groq: người học đang chờ câu trả lời',
    );
    expect(el.querySelector('#op-explain')?.getAttribute('aria-describedby')).toBe('hint-explain');
    const select = el.querySelector<HTMLSelectElement>('#op-explain')!;
    expect(el.querySelector('label[for="op-explain"]')?.textContent).toBe('Người học hỏi nghĩa');
    expect(select.options[0].textContent?.trim()).toBe('Theo model chung (Gemini 3.5 Flash Lite)');
    expect(select.value).toBe('');
    select.value = 'openai/gpt-oss-120b';
    select.dispatchEvent(new Event('change'));
    await settle();
    expect(el.querySelector('.op-own label')?.textContent).toBe('Người học hỏi nghĩa');
    expect(button('Lưu')?.disabled).toBe(false);

    button('Lưu')?.click();
    const req = http.expectOne('/admin/ai-model');
    expect(req.request.body).toEqual({
      model: 'gemini-3.5-flash-lite',
      byOp: { explain: 'openai/gpt-oss-120b' },
    });
    req.flush(
      state({
        saved: 'gemini-3.5-flash-lite',
        options: [...OPTIONS, 'openai/gpt-oss-120b'],
        byOp: { explain: 'openai/gpt-oss-120b' },
      }),
    );
    await settle();
    expect(el.querySelector('.status')?.textContent).toContain('1 việc dùng model riêng');
    expect(el.querySelector<HTMLSelectElement>('#op-explain')!.value).toBe('openai/gpt-oss-120b');
    expect(button('Lưu')?.disabled).toBe(true);

    // Back to the default: every op on GEMINI_MODEL.
    button('Về mặc định')?.click();
    const reset = http.expectOne('/admin/ai-model');
    expect(reset.request.body).toEqual({ model: '', byOp: {} });
    reset.flush(state());
    await settle();
    expect(el.querySelector('.op-own')).toBeNull();
  });

  it('applies the advice in one click, saved with the button', async () => {
    http
      .expectOne('/admin/ai-model')
      .flush(state({ options: [...OPTIONS, 'openai/gpt-oss-120b'] }));
    await settle();
    expect(el.querySelector('.dirty-note')).toBeNull();
    button('Áp dụng gợi ý')?.click();
    await settle();
    expect(Array.from(el.querySelectorAll('.op-own label')).map((l) => l.textContent)).toEqual([
      'Người học hỏi nghĩa',
      'Luyện nói với AI (lượt đáp)',
    ]);
    expect(el.querySelector('.dirty-note')?.textContent).toBe('Có thay đổi chưa lưu');
    button('Bỏ hết lựa chọn riêng')?.click();
    await settle();
    expect(el.querySelector('.op-own')).toBeNull();
  });

  it('offers no advice button without the Groq model', async () => {
    http.expectOne('/admin/ai-model').flush(state());
    await settle();
    expect(button('Áp dụng gợi ý')).toBeUndefined();
    expect(Array.from(el.querySelectorAll('.group-heading')).map((h) => h.textContent)).toEqual([
      'Gemini',
      'Người học đang chờ',
      'Soạn nội dung và việc chạy nền',
    ]);
  });

  it('offers to try again when loading fails', async () => {
    http.expectOne('/admin/ai-model').flush(null, { status: 500, statusText: 'Error' });
    await settle();
    expect(el.querySelector('[role="alert"]')?.textContent).toContain(
      'Không tải được model đang dùng.',
    );
  });
});
