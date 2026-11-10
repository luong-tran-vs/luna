import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarImageBatch, GrammarLessonAdmin } from '../../../core/models/grammar-admin';
import { GrammarExampleImages } from './grammar-example-images';
import { GrammarImageBatchService } from './grammar-image-batch.service';

const batch = (extra: Partial<GrammarImageBatch> = {}): GrammarImageBatch => ({
  running: false,
  stopping: false,
  stopped: false,
  engine: '',
  key: '',
  label: '',
  total: 0,
  points: 0,
  done: 0,
  drawing: null,
  saved: [],
  failures: [],
  codexImages: false,
  ...extra,
});

const lesson = (images: string[]): GrammarLessonAdmin =>
  ({
    pointId: 'a1-to-be',
    status: 'draft',
    content: {
      examples: [
        { en: 'I am a student.', vi: 'Tôi là học sinh.' },
        { en: 'She is a nurse.', vi: 'Cô ấy là y tá.' },
        { en: 'They are friends.', vi: 'Họ là bạn.' },
      ],
      practice: [],
      mastery: [],
    },
    exampleImages: images,
  }) as unknown as GrammarLessonAdmin;

describe('GrammarExampleImages', () => {
  let fixture: ComponentFixture<GrammarExampleImages>;
  let el: HTMLElement;
  let http: HttpTestingController;
  const url = '/admin/grammar-image-batch';
  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const pictured = '/api/admin/grammar-lessons/a1-to-be/images/abc123';
  const radio = (value: 'gemini' | 'codex') =>
    el.querySelector<HTMLInputElement>(`input[name="grammar-images-engine"][value="${value}"]`)!;
  /** Picks what draws: nothing is chosen until the admin does (asked 2026-11-10). */
  const chooseEngine = async (value: 'gemini' | 'codex') => {
    radio(value).click();
    await settle();
  };

  const open = async (images = ['', pictured, ''], status = batch()) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarExampleImages);
    fixture.componentRef.setInput('lesson', lesson(images));
    fixture.componentRef.setInput('title', 'to be');
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne(url).flush(status);
    await settle();
  };

  beforeEach(() => localStorage.removeItem('luna.admin.imageEngine'));
  afterEach(() => http.verify());

  it('lists the examples with their picture, and draws the missing ones in the background', async () => {
    await open();
    expect(text(el.querySelector('.head .muted'))).toBe('1/3 ảnh đã có');
    const rows = el.querySelectorAll('.row');
    expect(rows[0].textContent).toContain('Chưa có ảnh');
    expect(rows[1].querySelector('img')).not.toBeNull();

    let reloaded = 0;
    fixture.componentInstance.reload.subscribe(() => reloaded++);
    await chooseEngine('gemini');
    button('Tạo 2 ảnh còn thiếu').click();
    await settle();
    const req = http.expectOne((r) => r.url === url && r.method === 'POST');
    expect(req.request.body).toEqual({
      pointIds: ['a1-to-be'],
      engine: 'gemini',
      key: 'point:a1-to-be',
      label: 'bài “to be”',
    });
    const running = batch({
      running: true,
      engine: 'gemini',
      key: 'point:a1-to-be',
      label: 'bài “to be”',
      total: 2,
      drawing: { pointId: 'a1-to-be', key: 'k1', en: 'I am a student.' },
    });
    req.flush(running);
    await settle();
    expect(text(el.querySelector('.bg-note'))).toContain('Máy chủ đang tạo ảnh ngầm: xong 0/2 ảnh');
    expect(el.querySelectorAll('.row')[0].textContent).toContain('Đang vẽ');

    // The batch ends: the page is asked to read the lesson again.
    TestBed.inject(GrammarImageBatchService).set({
      ...running,
      running: false,
      drawing: null,
      done: 2,
      saved: [
        { pointId: 'a1-to-be', key: 'k1', en: 'I am a student.' },
        { pointId: 'a1-to-be', key: 'k3', en: 'They are friends.' },
      ],
    });
    await settle();
    expect(reloaded).toBe(1);
    expect(text(el.querySelector('p.muted[role="status"]'))).toBe('Đã tạo 2/2 ảnh bằng Gemini.');
  });

  it('draws one example again, replacing its picture, and removes one', async () => {
    await open();
    await chooseEngine('gemini');
    button('Vẽ lại').click();
    await settle();
    const req = http.expectOne((r) => r.url === url && r.method === 'POST');
    expect(req.request.body.keys).toEqual(['abc123']);
    expect(req.request.body.replace).toBe(true);
    req.flush(batch());
    await settle();

    let changed: GrammarLessonAdmin | null = null;
    fixture.componentInstance.changed.subscribe((l) => (changed = l));
    button('Xoá').click();
    await settle();
    const del = http.expectOne('/admin/grammar-lessons/a1-to-be/images/abc123');
    expect(del.request.method).toBe('DELETE');
    del.flush({ lesson: lesson(['', '', '']), exampleImages: ['', '', ''] });
    await settle();
    expect(changed!.exampleImages).toEqual(['', '', '']);
  });

  it('waits while another picture batch runs', async () => {
    await open(
      ['', '', ''],
      batch({ running: true, key: 'level:A1', label: 'các bài ngữ pháp A1', total: 20, done: 4 }),
    );
    expect(button('Tạo 3 ảnh còn thiếu').disabled).toBe(true);
    expect(el.textContent).toContain(
      'Máy chủ đang tạo ảnh ví dụ ngầm cho các bài ngữ pháp A1 (xong 4/20)',
    );
  });

  it('chooses no engine at first, remembers none, and draws nothing until one is chosen', async () => {
    localStorage.setItem('luna.admin.imageEngine', 'gemini');
    await open();
    expect(radio('gemini').checked).toBe(false);
    expect(radio('codex').checked).toBe(false);
    expect(el.querySelector('.engine.selected')).toBeNull();

    button('Tạo 2 ảnh còn thiếu').click();
    await settle();
    expect(text(el.querySelector('.form-alert'))).toBe(
      'Hãy chọn tạo ảnh bằng Gemini hay Codex CLI trước.',
    );
    button('Vẽ lại').click();
    await settle();
    http.expectNone((r) => r.url === url && r.method === 'POST');

    localStorage.removeItem('luna.admin.imageEngine');
    await chooseEngine('gemini');
    expect(radio('gemini').checked).toBe(true);
    expect(localStorage.getItem('luna.admin.imageEngine')).toBeNull();
  });

  it('draws with Codex when the server offers it, and Codex counts as no choice otherwise', async () => {
    await open(['', pictured, ''], batch({ codexImages: true }));
    expect(radio('codex').disabled).toBe(false);
    await chooseEngine('codex');
    button('Tạo 2 ảnh còn thiếu').click();
    await settle();
    const req = http.expectOne((r) => r.url === url && r.method === 'POST');
    expect(req.request.body.engine).toBe('codex');
    // The server no longer offers Codex: the choice no longer stands.
    req.flush(batch({ codexImages: false }));
    await settle();
    expect(radio('codex').disabled).toBe(true);
    expect(radio('codex').checked).toBe(false);
    button('Tạo 2 ảnh còn thiếu').click();
    await settle();
    expect(text(el.querySelector('.form-alert'))).toBe(
      'Hãy chọn tạo ảnh bằng Gemini hay Codex CLI trước.',
    );
    http.expectNone((r) => r.url === url && r.method === 'POST');
  });

  it('shows why the batch could not start', async () => {
    await open();
    await chooseEngine('gemini');
    button('Tạo 2 ảnh còn thiếu').click();
    await settle();
    http
      .expectOne((r) => r.url === url && r.method === 'POST')
      .flush(
        { message: 'AI chưa được cấu hình. Liên hệ người vận hành.' },
        { status: 503, statusText: 'x' },
      );
    await settle();
    expect(text(el.querySelector('.form-alert'))).toBe(
      'AI chưa được cấu hình. Liên hệ người vận hành.',
    );
  });

  it('lists the picture exercises too, and their pictures (added 2026-11-06)', async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarExampleImages);
    const l = lesson(['', '', '']);
    l.content.practice = [
      {
        id: 'p8',
        kind: 'picture',
        text: 'The cat is ___ the sofa.',
        options: ['in', 'on', 'under'],
        answerIndex: 2,
        picture: 'a cat under a blue sofa',
        explanationVi: '',
      },
    ];
    fixture.componentRef.setInput('lesson', l);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne(url).flush(batch());
    await settle();
    const rows = el.querySelectorAll('.row');
    expect(rows.length).toBe(4);
    expect(text(rows[3])).toContain('Bài tập p8 · nhìn ảnh chọn từ');
    expect(text(rows[3])).toContain('Ảnh: a cat under a blue sofa');
    expect(button('Tạo 4 ảnh còn thiếu')).toBeTruthy();
  });

  it('draws nothing for a point that cannot be drawn', async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarExampleImages);
    fixture.componentRef.setInput('lesson', lesson(['', '', '']));
    fixture.componentRef.setInput('drawable', false);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne(url).flush(batch());
    await settle();
    expect(el.textContent).toContain('khó minh hoạ bằng ảnh');
    expect(el.querySelector('.row')).toBeNull();
    expect(el.querySelector('lu-engine-picker')).toBeNull();
  });
});
