import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Provider } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { DueCard, DueList, ReviewForecast } from '../../../core/models/vocab';
import { FocusMode } from '../../../core/services/focus-mode.service';
import { Review } from './review';

/** 2026-10-15 is a Thursday. */
const FORECAST: ReviewForecast = {
  new: 2,
  learning: 10,
  known: 3,
  days: [10, 4, 0, 2, 0, 6, 1].map((count, i) => ({ day: `2026-10-${15 + i}`, count })),
};

const intervals = { again: 60, hard: 300, good: 600, easy: 8 * 86400 };
const dueCard: DueCard = {
  id: 'c1', text: 'went', lemma: 'go', ipa: '', meaningVi: 'đi', contextSentence: '', lessonId: null, source: 'manual',
  createdAt: '2026-09-28T01:00:00Z', due: '2026-09-29T00:00:00Z', reps: 0, state: 'new', intervals,
};

/** The route of a review started from the done card of a lesson: the next lesson is in the query. */
const fromLesson = {
  provide: ActivatedRoute,
  useValue: { snapshot: { queryParamMap: convertToParamMap({ next: 'l2' }) } },
};

describe('Review', () => {
  let fixture: ComponentFixture<Review>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (selector: string) => el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  // By its text or, for an icon or a rating button, its aria-label.
  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find(
      (b) => b.textContent?.replace(/\s+/g, ' ').trim() === label || b.getAttribute('aria-label') === label,
    );
  const flushDue = (list: DueList, forecast: ReviewForecast | null = FORECAST) => {
    http.expectOne((r) => r.url === '/vocab/review/due').flush(list);
    const req = http.expectOne('/vocab/review/forecast');
    if (forecast) {
      req.flush(forecast);
    } else {
      req.flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'Error' });
    }
  };

  const setup = async (list: DueList, extra: Provider[] = [], forecast: ReviewForecast | null = FORECAST) => {
    TestBed.configureTestingModule({
      imports: [Review],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(new FakeSpeech()),
        ...extra,
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Review);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    flushDue(list, forecast);
    await settle();
  };

  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn();
  });

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  it('shows how many cards are due and starts in the chosen mode', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null });
    expect(text('.due-count')).toBe('1 thẻ cần ôn');
    expect(text('.start')).toContain('Khoảng 1 phút');
    const radios = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
    expect(radios.map((r) => r.textContent?.replace(/\s+/g, ' ').trim())).toEqual(['Xem từ đoán nghĩa', 'Nghe rồi gõ từ']);
    expect(radios[0].getAttribute('aria-checked')).toBe('true');

    radios[1].click();
    await settle();
    button('Bắt đầu ôn')!.click();
    await settle();
    expect(el.querySelector('lu-review-session')).toBeTruthy();
    expect(el.querySelector('#review-answer')).toBeTruthy(); // listen mode
  });

  it('shows the cards by stage and the next 7 days', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null });
    expect(Array.from(el.querySelectorAll('.stages li')).map((li) => li.textContent?.replace(/\s+/g, ' ').trim())).toEqual([
      '2 Mới',
      '10 Đang học',
      '3 Đã thuộc',
    ]);
    const days = Array.from(el.querySelectorAll('.bars li'));
    expect(days.map((d) => d.querySelector('.bar-label')?.textContent?.trim())).toEqual(['Nay', 'T6', 'T7', 'CN', 'T2', 'T3', 'T4']);
    expect(days[0].querySelector('.visually-hidden')?.textContent).toBe('Hôm nay 15/10: 10 thẻ');
    expect(days[1].querySelector('.visually-hidden')?.textContent).toBe('Thứ Sáu 16/10: 4 thẻ');
    expect(days[0].classList).toContain('today');
    expect(days[1].querySelector<HTMLElement>('.bar')!.style.height).toBe('40%');
    expect(text('.menu-hint')).toBe('15 từ · Xem, tìm và thêm thẻ');
  });

  it('leaves the stages and the days out when they cannot be loaded', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null }, [], null);
    expect(el.querySelector('.stages')).toBeNull();
    expect(el.querySelector('.week')).toBeNull();
    expect(text('.menu-hint')).toBe('Xem, tìm và thêm thẻ');
    expect(button('Bắt đầu ôn')).toBeTruthy();
  });

  it('hides the tab bar during the session and reloads when it is stopped', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null });
    const focus = TestBed.inject(FocusMode);
    expect(focus.on()).toBe(false);
    button('Bắt đầu ôn')!.click();
    await settle();
    expect(focus.on()).toBe(true);
    expect(el.querySelector('.screen-bar')).toBeNull();

    button('Dừng ôn')!.click();
    await settle();
    expect(focus.on()).toBe(false);
    flushDue({ cards: [dueCard], total: 1, nextDue: null });
    await settle();
    expect(button('Bắt đầu ôn')).toBeTruthy();

    button('Bắt đầu ôn')!.click();
    await settle();
    fixture.destroy();
    expect(focus.on()).toBe(false);
  });

  it('offers the next lesson after a review started from a finished lesson', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null }, [fromLesson]);
    expect(el.querySelector('a[href="/lessons/l2"]')).toBeNull(); // only when the review is done
    button('Bắt đầu ôn')!.click();
    await settle();
    button('Lật thẻ')!.click();
    await settle();
    button('Nhớ · 10 phút')!.click();
    await settle();
    http.expectOne('/vocab/cards/c1/review').flush({ card: { ...dueCard, reps: 1 } });
    await settle();
    expect(text('a.btn-primary')).toBe('Học bài tiếp theo');
    expect(el.querySelector('a.btn-primary')!.getAttribute('href')).toBe('/lessons/l2');
    expect(button('Ôn tiếp')).toBeDefined();
  });

  it('offers the next lesson at once when no card is due', async () => {
    await setup({ cards: [], total: 0, nextDue: null }, [fromLesson]);
    expect(el.querySelector('a.btn-primary')!.getAttribute('href')).toBe('/lessons/l2');
  });

  it('has no lesson link when the review was not started from a lesson', async () => {
    await setup({ cards: [], total: 0, nextDue: null });
    expect(text('.hub')).not.toContain('Học bài tiếp theo');
  });

  it('says when nothing is due and when the next card will be', async () => {
    await setup({ cards: [], total: 0, nextDue: '2026-09-30T17:00:00Z' });
    expect(el.textContent).toContain('Không có thẻ nào đến hạn');
    expect(text('.next-due')).toContain('Thẻ tiếp theo đến hạn lúc');
    expect(button('Bắt đầu ôn')).toBeUndefined();
  });

  it('offers to review again after the session', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null });
    button('Bắt đầu ôn')!.click();
    await settle();
    button('Lật thẻ')!.click();
    await settle();
    button('Nhớ · 10 phút')!.click();
    await settle();
    http.expectOne('/vocab/cards/c1/review').flush({ card: { ...dueCard, reps: 1 } });
    await settle();
    expect(el.textContent).toContain('Đã ôn 1 thẻ');

    button('Ôn tiếp')!.click();
    await settle();
    flushDue({ cards: [], total: 0, nextDue: null });
    await settle();
    expect(el.textContent).toContain('Không có thẻ nào đến hạn');
  });
});
