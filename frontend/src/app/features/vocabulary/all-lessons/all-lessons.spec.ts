import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { MyLessons } from '../../../core/models/study';
import { LessonStats } from '../../../core/models/vocab';
import { AllLessons } from './all-lessons';

describe('AllLessons', () => {
  let fixture: ComponentFixture<AllLessons>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const mine: MyLessons = {
    current: null,
    completed: [
      { id: 'l1', title: 'Hello', topicName: '', completedAt: '2026-10-01T01:00:00Z' },
      { id: 'l2', title: 'At the office', topicName: '', completedAt: '2026-10-05T01:00:00Z' },
      { id: 'l3', title: 'My family', topicName: '', completedAt: '2026-10-09T01:00:00Z' },
    ],
    upcoming: [],
  };
  const stats: LessonStats[] = [
    {
      lessonId: 'l2',
      cards: 4,
      due: 2,
      remembered: 1,
      lastReview: '2026-10-08T01:00:00Z',
      nextDue: null,
      result: null,
    },
  ];

  const titles = () =>
    Array.from(el.querySelectorAll('.lesson-title')).map((t) => t.textContent?.trim());
  const chip = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('.chip')).find((c) =>
      c.textContent?.replace(/\s+/g, ' ').trim().startsWith(label),
    )!;

  async function setup(fail = false): Promise<void> {
    TestBed.configureTestingModule({
      imports: [AllLessons],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(AllLessons);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const lessons = http.expectOne((r) => r.url === '/lessons/mine');
    if (fail) {
      lessons.flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'x' });
    } else {
      lessons.flush(mine);
      http.expectOne('/vocab/review/lessons').flush({ lessons: stats });
    }
    await fixture.whenStable();
  }

  afterEach(() => http.verify());

  it('lists every lesson by need, with a filter', async () => {
    await setup();
    expect(el.querySelector('h1')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Bài đã học · 3');
    expect(titles()).toEqual(['At the office', 'Hello', 'My family']);
    expect(chip('Tất cả').getAttribute('aria-pressed')).toBe('true');
    expect(chip('Cần ôn').textContent?.replace(/\s+/g, ' ').trim()).toBe('Cần ôn · 1');

    chip('Cần ôn').click();
    await fixture.whenStable();
    expect(titles()).toEqual(['At the office']);

    chip('Chưa ôn').click();
    await fixture.whenStable();
    expect(titles()).toEqual(['Hello', 'My family']);
  });

  it('offers to try again when the list fails', async () => {
    await setup(true);
    http.match('/vocab/review/lessons');
    expect(el.textContent).toContain('Không tải được danh sách bài.');
  });
});
