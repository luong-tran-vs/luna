import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Goals, MyLessons as MyLessonsData } from '../../../core/models/study';
import { MyLessons } from './my-lessons';

describe('MyLessons', () => {
  let fixture: ComponentFixture<MyLessons>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  const setup = async (data: MyLessonsData, goals: Goals = { active: null, others: [] }) => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(MyLessons);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/lessons/mine').flush(data);
    http.expectOne('/api/goals').flush(goals);
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('lists the lessons studied, the one being studied and the locked upcoming ones', async () => {
    await setup({
      current: { id: 'l3', title: 'At the café' },
      completed: [
        { id: 'l2', title: 'My family', topicName: 'Gia đình', completedAt: '2026-09-29T10:00:00Z' },
        { id: 'l1', title: 'Shopping', topicName: 'Mua sắm', completedAt: '2026-09-28T10:00:00Z' },
      ],
      upcoming: [{ id: 'l4', title: 'At the bank' }],
    });

    const today = el.querySelector('.today')!;
    expect(text(today)).toContain('At the café');
    expect(text(today)).toContain('Đang học');
    // No separate "today" page: the row opens the lesson itself.
    expect(today.querySelectorAll('a')).toHaveLength(1);

    const done = Array.from(el.querySelectorAll('.completed li'));
    // Oldest first, numbered in the order they were studied.
    expect(done.map((li) => text(li.querySelector('.title')))).toEqual(['Shopping', 'My family']);
    expect(done.map((li) => text(li.querySelector('.num')))).toEqual(['1', '2']);
    // Each row opens the lesson's detail page.
    expect(done.map((li) => li.querySelector('a')?.getAttribute('href'))).toEqual(['/lessons/l1', '/lessons/l2']);
    expect(text(done[1])).toContain('Đã học 29/09/2026');
    expect(today.querySelector('a[href="/lessons/l3"]')).toBeTruthy();

    const upcoming = Array.from(el.querySelectorAll('.upcoming li'));
    expect(text(upcoming[0])).toContain('At the bank');
    expect(text(today.querySelector('.num'))).toBe('3');
    expect(text(upcoming[0].querySelector('.num'))).toBe('4');
    expect(upcoming[0].querySelector('lu-icon[name="lock"]')).toBeTruthy();
    expect(text(upcoming[0].querySelector('.visually-hidden'))).toBe('chưa mở');
    expect(upcoming[0].querySelector('a')).toBeNull();
  });

  it('names the topic being studied with its progress', async () => {
    await setup(
      { current: null, completed: [], upcoming: [] },
      {
        active: {
          topicId: 't1',
          topicName: 'Gia đình',
          level: 'A1',
          completedLessons: 3,
          totalLessons: 12,
          status: 'active',
          effectiveFrom: '2026-09-30',
        },
        others: [],
      },
    );
    expect(text(el.querySelector('h1'))).toBe('A1 · Gia đình');
    expect(text(el.querySelector('.progress-text'))).toBe('3 / 12 bài');
    expect(el.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('3');
  });

  it('without a goal: no empty boxes, a way to choose a topic', async () => {
    await setup({ current: null, completed: [], upcoming: [] });
    expect(el.querySelector('.today')).toBeNull();
    expect(el.querySelector('.completed')).toBeNull();
    expect(el.querySelector('.upcoming')).toBeNull();
    expect(text(el.querySelector('[role="status"]'))).toContain('Bạn chưa chọn chủ đề nào');
    expect(el.querySelector('a[href="/goal"].btn-primary')).toBeTruthy();
  });

  it('after the last lesson: only the lessons studied, and a note', async () => {
    await setup(
      {
        current: null,
        completed: [{ id: 'l1', title: 'Shopping', topicName: 'Mua sắm', completedAt: '2026-09-28T10:00:00Z' }],
        upcoming: [],
      },
      {
        active: {
          topicId: 't1',
          topicName: 'Mua sắm',
          level: 'A1',
          completedLessons: 1,
          totalLessons: 1,
          status: 'active',
          effectiveFrom: '2026-09-30',
        },
        others: [],
      },
    );
    expect(el.querySelector('.today')).toBeNull();
    expect(el.querySelectorAll('.completed li')).toHaveLength(1);
    expect(text(el.querySelector('p[role="status"]'))).toBe('Bạn đã học hết các bài hiện có của chủ đề này.');
  });
});
