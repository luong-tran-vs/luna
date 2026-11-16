import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { SpotCount, SpotItem } from '../../../core/models/games';
import { GrammarSpots } from './grammar-spots';

const count = (over: Partial<SpotCount> = {}): SpotCount => ({
  pointId: 'a1-to-be',
  level: 'A1',
  titleVi: 'Động từ to be',
  choice: 3,
  fix: 2,
  spot: 1,
  draft: 1,
  total: 6,
  ...over,
});

const item = (id: string, over: Partial<SpotItem> = {}): SpotItem => ({
  id,
  pointId: 'a1-to-be',
  level: 'A1',
  sentence: 'She are a teacher.',
  wrong: 'are',
  right: 'is',
  distractors: ['am', 'be'],
  ruleVi: 'She đi với is.',
  status: 'draft',
  createdAt: 't',
  updatedAt: 't',
  ...over,
});

describe('GrammarSpots', () => {
  let fixture: ComponentFixture<GrammarSpots>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string, root: Element = el) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const flushCounts = (points: SpotCount[]) =>
    http.expectOne('/admin/games/spot-counts').flush({ points });

  const setup = async (
    points: SpotCount[] = [count()],
    items: SpotItem[] = [item('s1'), item('s2', { status: 'approved' })],
    published = true,
  ) => {
    TestBed.configureTestingModule({
      imports: [GrammarSpots],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarSpots);
    fixture.componentRef.setInput('pointId', 'a1-to-be');
    fixture.componentRef.setInput('published', published);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    flushCounts(points);
    const req = http.expectOne((r) => r.url === '/admin/games/spot-items');
    expect(req.request.params.get('pointId')).toBe('a1-to-be');
    req.flush({ items });
    await settle();
  };

  afterEach(() => http.verify());

  it('shows the counts by source and flags a point under 10 sentences in words', async () => {
    await setup();
    expect(text(el.querySelector('.spot-sum strong'))).toBe('Tổng: 6 câu');
    expect(text(el.querySelector('.spot-low'))).toBe('⚠ dưới 10 câu, nên soạn thêm');
    expect(text(el.querySelector('.spot-sources'))).toContain('Bài chọn đáp án: 3');
    expect(text(el.querySelector('.spot-sources'))).toContain('Câu nháp chờ duyệt: 1');
    const labels = Array.from(el.querySelectorAll('.spot-item .tag')).map((t) => text(t));
    expect(labels).toEqual(['Nháp', 'Đã duyệt']);
    // Only a draft can be approved.
    const second = el.querySelector('[data-spot="s2"]')!;
    expect(button('Duyệt', second)).toBeUndefined();
  });

  it('asks the AI for more sentences and tells how many were kept', async () => {
    await setup([count({ total: 12 })], []);
    expect(el.querySelector('.spot-low')).toBeNull();
    button('Nhờ AI soạn thêm').click();
    await settle();
    expect(text(button('Đang soạn…'))).toBe('Đang soạn…');
    const req = http.expectOne('/admin/games/spot-items/generate');
    expect(req.request.body).toEqual({ pointId: 'a1-to-be' });
    req.flush({ created: [item('n1'), item('n2')], dropped: 1 });
    await settle();
    flushCounts([count({ total: 12, draft: 3 })]);
    await settle();
    expect(text(el.querySelector('[role="status"]'))).toBe('Đã soạn 2 câu, bỏ 1 câu không hợp lệ.');
    expect(el.querySelectorAll('.spot-item').length).toBe(2);
  });

  it('shows the server message when the AI fails', async () => {
    await setup();
    button('Nhờ AI soạn thêm').click();
    await settle();
    http
      .expectOne('/admin/games/spot-items/generate')
      .flush(
        { error: 'ai_quota', message: 'Đã hết hạn mức AI hôm nay.' },
        { status: 429, statusText: 'x' },
      );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Đã hết hạn mức AI hôm nay.');
  });

  it('turns the AI off until the lesson is published', async () => {
    await setup([], [], false);
    expect(button('Nhờ AI soạn thêm').disabled).toBe(true);
    expect(text(el.querySelector('.spot-none'))).toContain('Chưa có bài đã đăng');
    expect(text(el)).toContain('Chưa có câu soạn thêm.');
  });

  it('approves a draft', async () => {
    await setup();
    button('Duyệt').click();
    await settle();
    http
      .expectOne('/admin/games/spot-items/s1/approve')
      .flush({ item: item('s1', { status: 'approved' }) });
    await settle();
    flushCounts([count()]);
    await settle();
    expect(text(el.querySelector('[data-spot="s1"] .tag'))).toBe('Đã duyệt');
  });

  it('edits a sentence and shows the field errors of a 400', async () => {
    await setup();
    const row = el.querySelector('[data-spot="s1"]')!;
    button('Sửa', row).click();
    await settle();
    const right = el.querySelector<HTMLInputElement>('#spot-right-s1')!;
    expect(right.value).toBe('is');
    right.value = 'are';
    right.dispatchEvent(new Event('input'));
    button('Lưu câu').click();
    await settle();
    const req = http.expectOne('/admin/games/spot-items/s1');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({
      sentence: 'She are a teacher.',
      wrong: 'are',
      right: 'are',
      distractors: ['am', 'be'],
      ruleVi: 'She đi với is.',
    });
    req.flush(
      { error: 'invalid', message: 'x', fields: { right: 'Chữ đúng phải khác chỗ sai' } },
      { status: 400, statusText: 'x' },
    );
    await settle();
    expect(right.getAttribute('aria-invalid')).toBe('true');
    expect(text(el.querySelector('#spot-right-err-s1'))).toBe('Chữ đúng phải khác chỗ sai');

    right.value = 'is';
    right.dispatchEvent(new Event('input'));
    button('Lưu câu').click();
    await settle();
    http
      .expectOne('/admin/games/spot-items/s1')
      .flush({ item: item('s1', { sentence: 'She are a teacher.' }) });
    await settle();
    flushCounts([count()]);
    await settle();
    expect(el.querySelector('#spot-right-s1')).toBeNull();
  });

  it('deletes a sentence after confirming', async () => {
    await setup();
    button('Xoá', el.querySelector('[data-spot="s2"]')!).click();
    await settle();
    button('Xoá câu', el.querySelector('lu-confirm-dialog')!).click();
    await settle();
    const req = http.expectOne('/admin/games/spot-items/s2');
    expect(req.request.method).toBe('DELETE');
    req.flush(null);
    await settle();
    flushCounts([count()]);
    await settle();
    expect(el.querySelector('[data-spot="s2"]')).toBeNull();
  });
});
