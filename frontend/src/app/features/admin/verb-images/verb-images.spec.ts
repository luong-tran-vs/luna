import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { VerbImageBatch, VerbImageList } from '../../../core/models/verb-images';
import { IRREGULAR_VERBS } from '../../irregular-verbs/irregular-verbs.data';
import { VERB_EXAMPLES } from '../../irregular-verbs/verb-lessons.data';
import { BATCH_IDLE_POLL_MS, BATCH_POLL_MS } from '../batch-watcher';
import { VerbImages } from './verb-images';

const BATCH = '/admin/verb-image-batch';

const batch = (over: Partial<VerbImageBatch> = {}): VerbImageBatch => ({
  running: false,
  stopping: false,
  stopped: false,
  engine: '',
  total: 0,
  done: 0,
  drawing: [],
  saved: [],
  failures: [],
  codexImages: true,
  ...over,
});

const list = (over: Partial<VerbImageList> = {}): VerbImageList => ({
  images: [
    { base: 'cut', imageUrl: '/api/verbs/images/cut?v=1', updatedAt: '2026-11-10T00:00:00Z' },
  ],
  bank: [{ base: 'swim', meaningVi: 'bơi', imageUrl: '/api/admin/words/swim/image?v=1' }],
  codexImages: true,
  ...over,
});

describe('VerbImages', () => {
  let fixture: ComponentFixture<VerbImages>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const buttons = () => Array.from(el.querySelectorAll<HTMLButtonElement>('button'));
  const byText = (label: string) => buttons().find((b) => text(b).startsWith(label))!;
  const verbCard = (base: string) =>
    Array.from(el.querySelectorAll('.verb')).find(
      (c) => text(c.querySelector('.forms strong')) === base,
    )!;

  const setup = async (data: VerbImageList = list(), status: VerbImageBatch = batch()) => {
    TestBed.configureTestingModule({
      imports: [VerbImages],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: BATCH_POLL_MS, useValue: 60000 },
        { provide: BATCH_IDLE_POLL_MS, useValue: 60000 },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbImages);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const req = http.expectOne((r) => r.url === '/admin/verb-images');
    expect(req.request.params.get('bases')?.split(',')).toEqual(IRREGULAR_VERBS.map((v) => v.base));
    req.flush(data);
    http.expectOne(BATCH).flush(status);
    await settle();
  };

  afterEach(() => http.verify());

  it('lists the verbs by lesson with their pictures and the word bank ones to reuse', async () => {
    await setup();
    const total = IRREGULAR_VERBS.length;
    expect(text(el.querySelector('.count'))).toBe(`Có ảnh 1/${total} động từ`);
    expect(text(byText('Sinh ảnh còn thiếu'))).toBe(`Sinh ảnh còn thiếu (${total - 1})`);
    expect(verbCard('cut').querySelector('img')?.getAttribute('src')).toBe(
      '/api/verbs/images/cut?v=1',
    );
    expect(text(verbCard('swim').querySelector('.bank'))).toContain('Kho từ vựng có ảnh "bơi"');
  });

  it('makes the admin pick Gemini or Codex before drawing: there is no default', async () => {
    await setup();
    byText('Sinh ảnh còn thiếu').click();
    await settle();
    const dialog = el.querySelector('[role="dialog"]')!;
    const confirm = Array.from(dialog.querySelectorAll<HTMLButtonElement>('button')).find(
      (b) => text(b) === 'Tạo ảnh',
    )!;
    expect(dialog.querySelectorAll('input:checked').length).toBe(0);
    expect(confirm.disabled).toBe(true);

    dialog.querySelector<HTMLInputElement>('input[value="codex"]')!.click();
    await settle();
    expect(confirm.disabled).toBe(false);
    confirm.click();
    await settle();

    const req = http.expectOne((r) => r.url === BATCH && r.method === 'POST');
    expect(req.request.body.engine).toBe('codex');
    expect(req.request.body.items.length).toBe(IRREGULAR_VERBS.length - 1);
    const go = req.request.body.items.find((i: { base: string }) => i.base === 'go');
    expect(go).toEqual({
      base: 'go',
      meaningVi: expect.any(String),
      sentence: VERB_EXAMPLES['go'].en,
    });
    req.flush(batch({ running: true, engine: 'codex', total: 236, drawing: ['abide'] }));
    await settle();
    expect(text(el.querySelector('.batch'))).toContain('Đang tạo ảnh bằng Codex CLI');
  });

  it('copies a word bank picture and removes a picture after asking', async () => {
    await setup();
    byText('Dùng ảnh này').click();
    http
      .expectOne((r) => r.url === '/admin/verb-images/swim/from-bank' && r.method === 'POST')
      .flush({
        base: 'swim',
        imageUrl: '/api/verbs/images/swim?v=2',
        updatedAt: '2026-11-10T00:00:00Z',
      });
    http.expectOne((r) => r.url === '/admin/verb-images').flush(list());
    await settle();

    Array.from(verbCard('cut').querySelectorAll('button'))
      .find((b) => text(b).startsWith('Xoá'))!
      .click();
    await settle();
    expect(text(el.querySelector('[role="alertdialog"]'))).toContain('Xoá ảnh động từ?');
    Array.from(el.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button'))
      .find((b) => text(b) === 'Xoá ảnh')!
      .click();
    http.expectOne((r) => r.url === '/admin/verb-images/cut' && r.method === 'DELETE').flush(null);
    http.expectOne((r) => r.url === '/admin/verb-images').flush(list({ images: [] }));
    await settle();
    expect(verbCard('cut').querySelector('img')).toBeNull();
  });
});
