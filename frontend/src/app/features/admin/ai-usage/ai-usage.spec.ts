import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { AiTotals, AiUsage as Usage } from '../../../core/models/ai-usage';
import { AiUsage } from './ai-usage';

describe('AiUsage', () => {
  let fixture: ComponentFixture<AiUsage>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const zero: AiTotals = { requests: 0, errors: 0, quota: 0, promptTokens: 0, outputTokens: 0, totalTokens: 0 };
  const usage = (): Usage => ({
    now: '2026-10-08T05:00:00Z',
    lastMinute: [{ ...zero, model: 'flash', requests: 2, totalTokens: 1500 }],
    peakMinute: { ...zero, requests: 4, totalTokens: 6000 },
    minutes: Array.from({ length: 60 }, (_, i) => ({
      ...zero,
      start: new Date(Date.UTC(2026, 9, 8, 4, i)).toISOString(),
      requests: i === 59 ? 2 : i === 30 ? 4 : 0,
      totalTokens: i === 59 ? 1500 : i === 30 ? 6000 : 0,
    })),
    dayStart: '2026-10-07T07:00:00Z',
    today: [
      { ...zero, model: 'flash', requests: 12, quota: 1, totalTokens: 20000 },
      { ...zero, model: 'image', requests: 3, totalTokens: 4000 },
    ],
    week: [{ ...zero, op: 'annotate', requests: 10, errors: 1, promptTokens: 8000, outputTokens: 9000, totalTokens: 18000 }],
    recent: [
      {
        at: '2026-10-08T04:59:50Z',
        model: 'flash',
        op: 'word_meanings',
        outcome: 'quota',
        status: 429,
        promptTokens: 0,
        outputTokens: 0,
        thoughtTokens: 0,
        totalTokens: 0,
        durationMs: 300,
      },
    ],
  });

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(AiUsage);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('shows the last minute, today, the chart and the tables', async () => {
    http.expectOne('/admin/ai-usage').flush(usage());
    await fixture.whenStable();

    const stats = Array.from(el.querySelectorAll('.stat')).map((s) => s.textContent?.replace(/\s+/g, ' ').trim());
    expect(stats[0]).toContain('2 request');
    expect(stats[0]).toContain('1.500 token');
    expect(stats[1]).toContain('4 request');
    expect(stats[2]).toContain('15 request');
    expect(stats[3]).toContain('1 lần hết lượt');

    const bars = el.querySelectorAll<HTMLElement>('.bar-slot');
    expect(bars.length).toBe(60);
    expect(bars[30].querySelector<HTMLElement>('.bar')!.style.height).toBe('100%');
    bars[59].dispatchEvent(new Event('mouseenter'));
    await fixture.whenStable();
    expect(el.querySelector('.tooltip')!.textContent).toContain('2 request · 1.500 token');

    expect(el.textContent).toContain('Chú thích bài');
    expect(el.textContent).toContain('Điền nghĩa, phiên âm (kho từ)');
    expect(el.textContent).toContain('Hết lượt');
    expect(el.textContent).toContain('Mỗi chức năng tốn bao nhiêu request');
  });

  it('says when the numbers cannot be loaded and retries', async () => {
    http.expectOne('/admin/ai-usage').flush(null, { status: 500, statusText: 'Server Error' });
    await fixture.whenStable();
    expect(el.querySelector('[role="alert"]')!.textContent).toContain('Không tải được');

    Array.from(el.querySelectorAll('button'))
      .find((b) => b.textContent?.includes('Thử lại'))!
      .click();
    http.expectOne('/admin/ai-usage').flush(usage());
    await fixture.whenStable();
    expect(el.querySelector('[role="alert"]')).toBeNull();
  });
});
