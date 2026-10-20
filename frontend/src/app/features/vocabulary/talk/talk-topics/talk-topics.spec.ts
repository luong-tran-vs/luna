import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { of, throwError } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { TalkApiService } from '../../../../core/services/talk-api.service';
import { TalkTopics } from './talk-topics';

describe('TalkTopics', () => {
  let fixture: ComponentFixture<TalkTopics>;
  let el: HTMLElement;
  const api = { topics: vi.fn(), sessions: vi.fn(), start: vi.fn() };

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  const setup = async (
    topics = [
      {
        id: 'music',
        name: 'Âm nhạc',
        level: 'A1',
        lessons: 2,
        sessions: 1,
        lastAt: '2026-10-19T09:00:00Z',
      },
      { id: 'family', name: 'Gia đình', level: 'A2', lessons: 1, sessions: 0 },
    ],
  ) => {
    Object.values(api).forEach((f) => f.mockReset());
    api.topics.mockReturnValue(of(topics));
    api.sessions.mockReturnValue(
      of([
        {
          id: 's1',
          topicId: 'music',
          topicName: 'Âm nhạc',
          scenarioVi: 'Ở buổi hoà nhạc',
          status: 'ended',
          learnerTurns: 4,
          fixes: 2,
          createdAt: '2026-10-19T09:00:00Z',
        },
      ]),
    );
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: TalkApiService, useValue: api }],
    });
    fixture = TestBed.createComponent(TalkTopics);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('lists the studied topics and the past conversations', async () => {
    await setup();
    const topics = Array.from(el.querySelectorAll('.topic')).map((t) => text(t));
    expect(topics[0]).toContain('A1 Âm nhạc');
    expect(topics[0]).toContain('Đã học 2 bài · Đã luyện 1 lần');
    expect(topics[1]).toContain('Đã học 1 bài');
    expect(text(el.querySelector('.menu-list'))).toContain('4 lượt · 2 lỗi cần sửa');
  });

  it('starts a conversation and opens it', async () => {
    await setup();
    const go = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    api.start.mockReturnValue(of({ id: 's9' }));
    el.querySelector<HTMLButtonElement>('.topic .start')!.click();
    expect(api.start).toHaveBeenCalledWith('music');
    expect(go).toHaveBeenCalledWith(['/vocabulary/talk', 's9']);
  });

  it('shows why a conversation could not start', async () => {
    await setup();
    api.start.mockReturnValue(
      throwError(
        () => new ApiError('http', 429, { message: 'Đã hết lượt AI, vui lòng thử lại sau.' }),
      ),
    );
    el.querySelectorAll<HTMLButtonElement>('.topic .start')[1].click();
    await fixture.whenStable();
    expect(text(el.querySelectorAll('.topic')[1].querySelector('[role="alert"]'))).toBe(
      'Đã hết lượt AI, vui lòng thử lại sau.',
    );
  });

  it('explains the page is empty until a lesson is learned', async () => {
    await setup([]);
    expect(el.textContent).toContain('Học xong một bài là chủ đề của bài đó hiện ở đây');
  });
});
