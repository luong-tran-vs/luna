import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarPoint } from '../../../core/models/grammar';
import { GrammarLessonSummary, GrammarReportGroup } from '../../../core/models/grammar-admin';
import { GrammarList } from './grammar-list';

const point = (id: string, titleVi: string, level = 'A1'): GrammarPoint => ({
  id, level, titleVi, titleEn: id, pattern: `mẫu ${id}`, hintVi: 'gợi ý', examples: [], lessonCount: 0,
});
const summary = (pointId: string, over: Partial<GrammarLessonSummary> = {}): GrammarLessonSummary => ({
  pointId, status: 'draft', edited: false, updatedAt: '2026-10-01T00:00:00Z', publishedAt: null, flags: 0, checked: true, verified: false, ...over,
});

const points = [point('a1-to-be', 'Động từ to be'), point('a1-articles', 'Mạo từ'), point('a1-this-that', 'This / that')];
const lessons = [summary('a1-to-be', { status: 'published', publishedAt: 't' }), summary('a1-articles', { edited: true })];

describe('GrammarList', () => {
  let fixture: ComponentFixture<GrammarList>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const row = (id: string) => el.querySelector(`[data-point="${id}"]`)!;
  const status = (id: string) => text(row(id).querySelector('.status'));
  const labels = (id: string) => Array.from(row(id).querySelectorAll('.actions > *')).map((n) => text(n));
  const click = async (id: string, label: string) => {
    const target = Array.from(row(id).querySelectorAll<HTMLElement>('.actions > *')).find((n) => text(n).startsWith(label))!;
    target.click();
    await settle();
  };
  const lessonUrl = (id: string, action: string) => `/admin/grammar-lessons/${id}/${action}`;
  const flushLevel = (level: string, list: GrammarPoint[], existing: GrammarLessonSummary[]) => {
    const req = http.expectOne((r) => r.url === '/admin/grammar');
    expect(req.request.params.get('level')).toBe(level);
    req.flush({ points: list });
    http.expectOne('/admin/grammar-lessons').flush({ lessons: existing });
  };
  const detail = (s: GrammarLessonSummary) => ({
    lesson: { ...s, content: { objective: 'o' }, checks: [], checkedAt: s.checked ? 't' : null, verifiedAt: s.verified ? 't' : null },
  });
  const flushReports = (reports: GrammarReportGroup[] | 'error' = []) => {
    const req = http.expectOne('/admin/grammar-reports');
    if (reports === 'error') {
      req.flush({}, { status: 500, statusText: 'x' });
    } else {
      req.flush({ reports });
    }
  };
  const group = (over: Partial<GrammarReportGroup> = {}): GrammarReportGroup => ({
    pointId: 'a1-to-be', exerciseId: 'p3', count: 2, reasons: { wrong_answer: 1, typo: 1 }, notes: ['Phải là is'], latestAt: 't', ...over,
  });

  const setup = async (reports: GrammarReportGroup[] | 'error' = [], existing = lessons) => {
    TestBed.configureTestingModule({
      imports: [GrammarList],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarList);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    flushLevel('A1', points, existing);
    flushReports(reports);
    await settle();
  };

  afterEach(() => http.verify());

  it('shows the state of each point in words, with the published count', async () => {
    await setup();
    expect(status('a1-to-be')).toBe('Đã đăng');
    expect(status('a1-articles')).toBe('Bản nháp · đã sửa tay');
    expect(status('a1-this-that')).toBe('Chưa có bài');
    expect(text(el.querySelector('.summary'))).toBe('1/3 điểm đã đăng ở trình độ A1');
  });

  describe('quality control (F21)', () => {
    const badges = (id: string) => text(row(id).querySelector('.badges'));

    it('labels flagged rows in words, and unchecked ones lightly', async () => {
      await setup([], [
        summary('a1-to-be', { status: 'published', flags: 2 }),
        summary('a1-articles', { checked: false }),
      ]);
      expect(badges('a1-to-be')).toBe('2 câu cần xem');
      expect(badges('a1-articles')).toBe('Chưa kiểm tra');
      expect(row('a1-this-that').querySelector('.badges')).toBeNull();
    });

    it('labels a verified lesson in words, and "Chưa kiểm tra" only when never checked', async () => {
      await setup([], [
        summary('a1-to-be', { status: 'published', verified: true }),
        summary('a1-articles', { checked: false }),
      ]);
      expect(badges('a1-to-be')).toBe('Đã xác nhận');
      expect(badges('a1-articles')).toBe('Chưa kiểm tra');
    });

    it('shows nothing extra for a clean checked lesson', async () => {
      await setup();
      expect(badges('a1-to-be')).toBe('');
    });

    it('counts open reports per point', async () => {
      await setup([group({ count: 2 }), group({ exerciseId: 'p5', count: 1 })]);
      expect(badges('a1-to-be')).toBe('3 báo lỗi');
    });

    it('lists reported exercises above the list, linking to the detail page', async () => {
      await setup([group()]);
      const section = el.querySelector('.reports')!;
      expect(text(section)).toContain('Câu bị báo lỗi');
      expect(text(section)).toContain('Động từ to be');
      expect(text(section)).toContain('2 lượt · Đáp án sai: 1 · Lỗi chính tả: 1');
      expect(text(section)).toContain('“Phải là is”');
      expect(section.querySelector('a')!.getAttribute('href')).toBe('/admin/grammar/a1-to-be');
    });

    it('hides the section when there are no reports', async () => {
      await setup([]);
      expect(el.querySelector('.reports')).toBeNull();
    });

    it('still shows the list when the reports fail to load', async () => {
      await setup('error');
      expect(el.querySelector('.reports')).toBeNull();
      expect(el.querySelectorAll('[data-point]').length).toBe(3);
    });
  });

  it('offers the actions that fit each state', async () => {
    await setup();
    expect(labels('a1-to-be')).toEqual(['Sinh bằng AI Động từ to be', 'Xem / sửa Động từ to be', 'Gỡ Động từ to be']);
    expect(labels('a1-articles')).toEqual(['Sinh bằng AI Mạo từ', 'Xem / sửa Mạo từ', 'Đăng Mạo từ']);
    expect(labels('a1-this-that')).toEqual(['Sinh bằng AI This / that']);
    expect(row('a1-to-be').querySelector('a')!.getAttribute('href')).toBe('/admin/grammar/a1-to-be');
  });

  it('loads another level from its tab', async () => {
    await setup();
    Array.from(el.querySelectorAll<HTMLButtonElement>('.levels button')).find((b) => text(b) === 'B1')!.click();
    await settle();
    flushLevel('B1', [point('b1-passive', 'Bị động', 'B1')], []);
    await settle();
    expect(text(el.querySelector('.summary'))).toBe('0/1 điểm đã đăng ở trình độ B1');
    expect(el.querySelector('.levels button[aria-pressed="true"]')!.textContent).toContain('B1');
  });

  it('generates without asking when the point has no lesson', async () => {
    await setup();
    await click('a1-this-that', 'Sinh bằng AI');
    const req = http.expectOne(lessonUrl('a1-this-that', 'generate'));
    expect(req.request.body).toEqual({ force: false });
    expect(labels('a1-this-that')[0]).toBe('Đang sinh… This / that');
    req.flush(detail(summary('a1-this-that')), { status: 201, statusText: 'Created' });
    await settle();
    expect(status('a1-this-that')).toBe('Bản nháp');
    expect(text(el.querySelector('.banner-note'))).toContain('Đã sinh bản nháp');
  });

  it('asks before replacing an edited or published lesson, then sends force', async () => {
    await setup();
    await click('a1-articles', 'Sinh bằng AI');
    http.expectNone(lessonUrl('a1-articles', 'generate'));
    expect(text(el.querySelector('lu-confirm-dialog'))).toContain('đã sửa tay');
    el.querySelector<HTMLButtonElement>('lu-confirm-dialog .btn-primary')!.click();
    await settle();
    const req = http.expectOne(lessonUrl('a1-articles', 'generate'));
    expect(req.request.body).toEqual({ force: true });
    req.flush(detail(summary('a1-articles')));
    await settle();
    expect(status('a1-articles')).toBe('Bản nháp');
  });

  it('does not generate when the confirmation is cancelled', async () => {
    await setup();
    await click('a1-to-be', 'Sinh bằng AI');
    expect(text(el.querySelector('lu-confirm-dialog'))).toContain('đã đăng');
    el.querySelector<HTMLButtonElement>('lu-confirm-dialog .btn:not(.btn-primary)')!.click();
    await settle();
    expect(el.querySelector('lu-confirm-dialog [role="alertdialog"]')).toBeNull();
  });

  for (const [code, error, message] of [
    [503, 'ai_not_configured', 'AI chưa được cấu hình. Liên hệ người vận hành.'],
    [429, 'ai_quota', 'Đã hết lượt AI, vui lòng thử lại sau.'],
    [502, 'ai_failed', 'AI trả về nội dung không dùng được, vui lòng thử lại.'],
  ] as const) {
    it(`explains a ${code} from the AI`, async () => {
      await setup();
      await click('a1-this-that', 'Sinh bằng AI');
      http.expectOne(lessonUrl('a1-this-that', 'generate')).flush({ error }, { status: code, statusText: 'x' });
      await settle();
      expect(text(row('a1-this-that').querySelector('.row-error'))).toBe(message);
      expect(status('a1-this-that')).toBe('Chưa có bài');
    });
  }

  it('prefers the message of the server', async () => {
    await setup();
    await click('a1-this-that', 'Sinh bằng AI');
    http
      .expectOne(lessonUrl('a1-this-that', 'generate'))
      .flush({ error: 'ai_quota', message: 'Hết lượt hôm nay.' }, { status: 429, statusText: 'x' });
    await settle();
    expect(text(row('a1-this-that').querySelector('.row-error'))).toBe('Hết lượt hôm nay.');
  });

  it('publishes a draft and unpublishes a published lesson', async () => {
    await setup();
    await click('a1-articles', 'Đăng');
    http
      .expectOne(lessonUrl('a1-articles', 'publish'))
      .flush(detail(summary('a1-articles', { status: 'published', edited: true, publishedAt: 't' })));
    await settle();
    expect(status('a1-articles')).toBe('Đã đăng · đã sửa tay');
    expect(text(el.querySelector('.summary'))).toBe('2/3 điểm đã đăng ở trình độ A1');

    await click('a1-to-be', 'Gỡ');
    http.expectOne(lessonUrl('a1-to-be', 'unpublish')).flush(detail(summary('a1-to-be')));
    await settle();
    expect(status('a1-to-be')).toBe('Bản nháp');
    expect(text(el.querySelector('.summary'))).toBe('1/3 điểm đã đăng ở trình độ A1');
  });

  it('shows why a lesson cannot be published', async () => {
    await setup();
    await click('a1-articles', 'Đăng');
    http
      .expectOne(lessonUrl('a1-articles', 'publish'))
      .flush(
        { error: 'validation_failed', message: 'Nội dung chưa hợp lệ.', fields: { 'content.practice': 'cần 6–12 bài' } },
        { status: 400, statusText: 'x' },
      );
    await settle();
    expect(text(row('a1-articles').querySelector('.row-error'))).toContain('content.practice: cần 6–12 bài');
  });

  it('offers a retry when the list cannot be loaded', async () => {
    TestBed.configureTestingModule({
      imports: [GrammarList],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarList);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne((r) => r.url === '/admin/grammar').flush({}, { status: 500, statusText: 'x' });
    http.expectOne('/admin/grammar-lessons').flush({ lessons: [] });
    flushReports();
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không tải được danh sách điểm ngữ pháp.');
    el.querySelector<HTMLButtonElement>('[role="alert"] button')!.click();
    await settle();
    flushLevel('A1', points, lessons);
    await settle();
    expect(el.querySelectorAll('[data-point]').length).toBe(3);
  });
});
