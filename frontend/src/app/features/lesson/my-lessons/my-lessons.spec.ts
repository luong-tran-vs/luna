import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { MyLessons as MyLessonsData } from '../../../core/models/study';
import { MyLessons } from './my-lessons';

describe('MyLessons', () => {
  let fixture: ComponentFixture<MyLessons>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  const setup = async (data: MyLessonsData) => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(MyLessons);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/lessons/mine').flush(data);
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('lists today, the lessons studied and the locked upcoming ones', async () => {
    await setup({
      today: { id: 'l3', title: 'At the café' },
      completed: [
        { id: 'l2', title: 'My family', topicName: 'Gia đình', completedAt: '2026-09-29T10:00:00Z' },
        { id: 'l1', title: 'Shopping', topicName: 'Mua sắm', completedAt: '2026-09-28T10:00:00Z' },
      ],
      upcoming: [{ id: 'l4', title: 'At the bank' }],
    });

    const today = el.querySelector('.today')!;
    expect(text(today)).toContain('At the café');
    expect(today.querySelector('a[href="/today"]')).toBeTruthy();

    const done = Array.from(el.querySelectorAll('.completed li'));
    expect(done.map((li) => text(li.querySelector('.title')))).toEqual(['My family', 'Shopping']);
    expect(text(done[0])).toContain('Gia đình');
    expect(done[0].querySelector('a[href="/lessons/l2/read?review=1"]')).toBeTruthy();
    expect(done[0].querySelector('a[href="/lessons/l2/listen?review=1"]')).toBeTruthy();

    const upcoming = Array.from(el.querySelectorAll('.upcoming li'));
    expect(text(upcoming[0])).toContain('At the bank');
    expect(text(upcoming[0])).toContain('🔒');
    expect(text(upcoming[0].querySelector('.visually-hidden'))).toBe('chưa mở');
    expect(upcoming[0].querySelector('a')).toBeNull();
  });

  it('shows empty states', async () => {
    await setup({ today: null, completed: [], upcoming: [] });
    expect(text(el.querySelector('.today'))).toContain('Chưa có bài hôm nay');
    expect(el.querySelector('.today a[href="/goal"]')).toBeTruthy();
    expect(text(el.querySelector('.completed'))).toContain('Chưa học bài nào');
    expect(el.querySelector('.upcoming')).toBeNull();
  });
});
