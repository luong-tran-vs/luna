import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Provider } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { DueCard, DueList } from '../../../core/models/vocab';
import { Review } from './review';

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
  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.replace(/\s+/g, ' ').trim() === label);
  const flushDue = (list: DueList) => {
    const req = http.expectOne((r) => r.url === '/api/vocab/review/due');
    req.flush(list);
  };

  const setup = async (list: DueList, extra: Provider[] = []) => {
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
    flushDue(list);
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
    expect(text('.due-count')).toBe('1 thẻ đến hạn');
    const radios = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
    expect(radios.map((r) => r.textContent?.trim())).toEqual(['Xem từ đoán nghĩa', 'Nghe rồi gõ']);
    expect(radios[0].getAttribute('aria-checked')).toBe('true');

    radios[1].click();
    await settle();
    button('Bắt đầu')!.click();
    await settle();
    expect(el.querySelector('lu-review-session')).toBeTruthy();
    expect(el.querySelector('#review-answer')).toBeTruthy(); // listen mode
  });

  it('offers the next lesson after a review started from a finished lesson', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null }, [fromLesson]);
    expect(el.querySelector('a[href="/lessons/l2"]')).toBeNull(); // only when the review is done
    button('Bắt đầu')!.click();
    await settle();
    button('Lật thẻ')!.click();
    await settle();
    button('Nhớ · 10 phút')!.click();
    await settle();
    http.expectOne('/api/vocab/cards/c1/review').flush({ card: { ...dueCard, reps: 1 } });
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
    expect(button('Bắt đầu')).toBeUndefined();
  });

  it('offers to review again after the session', async () => {
    await setup({ cards: [dueCard], total: 1, nextDue: null });
    button('Bắt đầu')!.click();
    await settle();
    button('Lật thẻ')!.click();
    await settle();
    button('Nhớ · 10 phút')!.click();
    await settle();
    http.expectOne('/api/vocab/cards/c1/review').flush({ card: { ...dueCard, reps: 1 } });
    await settle();
    expect(el.textContent).toContain('Đã ôn 1 thẻ');

    button('Ôn tiếp')!.click();
    await settle();
    flushDue({ cards: [], total: 0, nextDue: null });
    await settle();
    expect(el.textContent).toContain('Không có thẻ nào đến hạn');
  });
});
