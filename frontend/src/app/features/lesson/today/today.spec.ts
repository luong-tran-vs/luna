import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component, input, output, Type } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Today as TodayData } from '../../../core/models/study';
import { DueCard, ReviewContext, ReviewMode, ReviewSummary } from '../../../core/models/vocab';
import { ReviewSession } from '../../../shared/components/review-session/review-session';
import { Listening } from '../listening/listening';
import { Reading } from '../reading/reading';
import { Writing } from '../writing/writing';
import { Today } from './today';

@Component({ selector: 'lu-review-session', template: '' })
class ReviewStub {
  readonly cards = input<DueCard[]>([]);
  readonly mode = input<ReviewMode>('flip');
  readonly context = input<ReviewContext>('free');
  readonly finished = output<ReviewSummary>();
}

@Component({ selector: 'lu-reading', template: '' })
class ReadingStub {
  readonly lessonId = input('');
  readonly startSentence = input<number | null>(null);
  readonly mode = input<string | null>(null);
  readonly completed = output<void>();
  readonly position = output<number>();
}

@Component({ selector: 'lu-writing', template: '' })
class WritingStub {
  readonly lessonId = input('');
  readonly mode = input<string | null>(null);
  readonly completed = output<void>();
}

@Component({ selector: 'lu-listening', template: '' })
class ListeningStub {
  readonly lessonId = input('');
  readonly startSentence = input<number | null>(null);
  readonly mode = input<string | null>(null);
  readonly completed = output<void>();
  readonly position = output<number>();
}

const goal = { topicId: 't1', topicName: 'Gia đình', level: 'A1' as const, completedLessons: 2, totalLessons: 12, status: 'active' as const, effectiveFrom: '' };

function today(over: Partial<TodayData> = {}): TodayData {
  return {
    kind: 'studying', goal, lesson: { id: 'l3', title: 'At the café' },
    steps: { review: 'current', read: 'locked', listen: 'locked', write: 'locked' }, currentStep: 'review', sentenceIndex: 0,
    reviewCount: 12, streak: 4, goalCompleted: false, ...over,
  };
}

const card = { id: 'c1', text: 'went' } as DueCard;

describe('Today', () => {
  let fixture: ComponentFixture<Today>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (selector: string) => el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const child = <T>(type: Type<T>) => fixture.debugElement.query(By.directive(type))?.componentInstance as T | undefined;
  const expectComplete = (step: string) => {
    const req = http.expectOne(`/api/today/steps/${step}/complete`);
    expect(req.request.method).toBe('POST');
    return req;
  };

  const setup = async (data: TodayData) => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    TestBed.overrideComponent(Today, {
      remove: { imports: [ReviewSession, Reading, Listening, Writing] },
      add: { imports: [ReviewStub, ReadingStub, ListeningStub, WritingStub] },
    });
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    fixture = TestBed.createComponent(Today);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/today').flush(data);
    await settle();
  };

  afterEach(() => {
    http.verify();
    vi.useRealTimers();
  });

  it('sends a learner without a goal to choose one', async () => {
    await setup(today({ kind: 'noGoal', goal: null, lesson: null }));
    expect(el.textContent).toContain('Bạn chưa chọn mục tiêu');
    expect(el.querySelector('a[href="/goal"]')).toBeTruthy();
  });

  it('shows the goal, streak, steps and the review session of the day', async () => {
    await setup(today());
    expect(text('.goal-name')).toBe('A1 · Gia đình');
    expect(text('.goal-progress')).toBe('2/12 bài');
    expect(text('.streak')).toContain('4 ngày');
    expect(text('.lesson-title')).toBe('At the café');
    expect(el.querySelector('lu-step-indicator')).toBeTruthy();

    const req = http.expectOne((r) => r.url === '/api/vocab/review/due');
    expect(req.request.params.get('limit')).toBe('12');
    req.flush({ cards: [card], total: 12, nextDue: null });
    await settle();
    const review = child(ReviewStub)!;
    expect(review.cards()).toEqual([card]);
    expect(review.context()).toBe('daily');
  });

  it('moves through review, reading and listening', async () => {
    await setup(today());
    http.expectOne((r) => r.url === '/api/vocab/review/due').flush({ cards: [card], total: 1, nextDue: null });
    await settle();

    child(ReviewStub)!.finished.emit({ reviewed: 1, counts: { 1: 0, 2: 0, 3: 1, 4: 0 } });
    await settle();
    expectComplete('review').flush(today({ steps: { review: 'done', read: 'current', listen: 'locked', write: 'locked' }, currentStep: 'read', sentenceIndex: 5 }));
    await settle();
    const reading = child(ReadingStub)!;
    expect(reading.lessonId()).toBe('l3');
    expect(reading.startSentence()).toBe(5);
    expect(reading.mode()).toBe('study');

    reading.completed.emit();
    await settle();
    expectComplete('read').flush(today({ steps: { review: 'done', read: 'done', listen: 'current', write: 'locked' }, currentStep: 'listen', sentenceIndex: 2 }));
    await settle();
    expect(child(ReadingStub)).toBeUndefined();
    const listening = child(ListeningStub)!;
    expect(listening.startSentence()).toBe(2);

    listening.completed.emit();
    await settle();
    expectComplete('listen').flush(
      today({ kind: 'doneToday', steps: { review: 'done', read: 'done', listen: 'done', write: 'done' }, currentStep: 'done', streak: 5, goal: { ...goal, completedLessons: 3 } }),
    );
    await settle();
    expect(el.textContent).toContain('Đã xong bài hôm nay, hẹn bạn ngày mai');
    expect(text('.streak')).toContain('5 ngày');
    expect(text('.goal-progress')).toBe('3/12 bài');
  });

  it('saves the position one second after the last move', async () => {
    await setup(today({ steps: { review: 'done', read: 'current', listen: 'locked', write: 'locked' }, currentStep: 'read', reviewCount: 0 }));
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    const reading = child(ReadingStub)!;
    reading.position.emit(2);
    reading.position.emit(3);
    vi.advanceTimersByTime(999);
    http.expectNone('/api/today/position');
    vi.advanceTimersByTime(1);
    const req = http.expectOne('/api/today/position');
    expect(req.request.body).toEqual({ step: 'read', sentenceIndex: 3 });
    req.flush(null, { status: 204, statusText: 'No Content' });
  });

  it('shows why a step cannot be completed', async () => {
    await setup(today({ steps: { review: 'done', read: 'done', listen: 'current', write: 'locked' }, currentStep: 'listen', reviewCount: 0 }));
    child(ListeningStub)!.completed.emit();
    await settle();
    expectComplete('listen').flush(
      { error: 'listen_incomplete', message: 'Hãy kiểm tra hết các câu của bước Nghe' },
      { status: 409, statusText: 'Conflict' },
    );
    await settle();
    expect(text('[role="alert"]')).toBe('Hãy kiểm tra hết các câu của bước Nghe');
  });

  it('shows the Write step and completes it after submitting (F8)', async () => {
    await setup(
      today({ steps: { review: 'done', read: 'done', listen: 'done', write: 'current' }, currentStep: 'write', reviewCount: 0 }),
    );
    const writing = child(WritingStub)!;
    expect(writing.lessonId()).toBe('l3');
    expect(writing.mode()).toBe('study');
    writing.completed.emit();
    await settle();
    expectComplete('write').flush(
      { error: 'write_incomplete', message: 'Hãy nộp bài viết' },
      { status: 409, statusText: 'Conflict' },
    );
    await settle();
    expect(text('[role="alert"]')).toBe('Hãy nộp bài viết');
  });

  it('shows why the Reading step is not done yet (F15)', async () => {
    await setup(today({ steps: { review: 'done', read: 'current', listen: 'locked', write: 'locked' }, currentStep: 'read', reviewCount: 0 }));
    child(ReadingStub)!.completed.emit();
    await settle();
    expectComplete('read').flush(
      { error: 'read_incomplete', message: 'Hãy trả lời hết câu hỏi hiểu bài' },
      { status: 409, statusText: 'Conflict' },
    );
    await settle();
    expect(text('[role="alert"]')).toBe('Hãy trả lời hết câu hỏi hiểu bài');
  });

  it('congratulates when the roadmap is finished', async () => {
    await setup(today({ steps: { review: 'done', read: 'done', listen: 'current', write: 'locked' }, currentStep: 'listen', reviewCount: 0 }));
    child(ListeningStub)!.completed.emit();
    await settle();
    expectComplete('listen').flush(today({ kind: 'doneToday', currentStep: 'done', goalCompleted: true }));
    await settle();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/goal?completed=1');
  });

  it('says the lesson is done for today with ways to keep practising', async () => {
    await setup(today({ kind: 'doneToday', currentStep: 'done', steps: { review: 'done', read: 'done', listen: 'done', write: 'done' } }));
    expect(el.textContent).toContain('Đã xong bài hôm nay');
    expect(el.querySelector('a[href="/vocabulary/review"]')).toBeTruthy();
    expect(el.querySelector('a[href="/lessons"]')).toBeTruthy();
    expect(child(ReadingStub)).toBeUndefined();
  });

  it('says there is no new lesson', async () => {
    await setup(today({ kind: 'noNewLesson', lesson: null, currentStep: 'done' }));
    expect(el.textContent).toContain('Chưa có bài mới');
    expect(el.querySelector('a[href="/vocabulary/review"]')).toBeTruthy();
    expect(el.querySelector('a[href="/goal"]')).toBeTruthy();
  });
});
