import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { WritingSummary } from '../../../core/models/writing';
import { WritingList } from './writing-list';

const row = (over: Partial<WritingSummary>): WritingSummary => ({
  id: 'w1',
  lessonId: 'l1',
  lessonTitle: 'My family',
  submittedAt: '2026-10-01T03:00:00Z',
  gradeStatus: 'done',
  average: 3.75,
  seen: true,
  ...over,
});

describe('WritingList', () => {
  let fixture: ComponentFixture<WritingList>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const open = async (writings: WritingSummary[]) => {
    await TestBed.configureTestingModule({
      imports: [WritingList],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(WritingList);
    el = fixture.nativeElement;
    http.expectOne('/writings').flush({ writings });
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('lists the writings with date, lesson and result', async () => {
    await open([
      row({ id: 'w3', lessonTitle: 'At work', gradeStatus: 'pending', average: null, seen: true, submittedAt: '2026-10-03T03:00:00Z' }),
      row({ id: 'w2', lessonTitle: 'Shopping', gradeStatus: 'failed', average: null, seen: false, submittedAt: '2026-10-02T03:00:00Z' }),
      row({ id: 'w1', seen: false }),
    ]);
    const rows = Array.from(el.querySelectorAll('.row'));
    expect(rows.map((r) => text(r.querySelector('.lesson')))).toEqual(['At work', 'Shopping', 'My family']);
    expect(rows.map((r) => text(r.querySelector('.result')))).toEqual(['Đang chấm', 'Chấm lỗi Mới', '3,8 / 5 Mới']);
    expect(text(rows[2].querySelector('.date'))).toBe('01/10/2026');
    expect(rows[1].getAttribute('href')).toBe('/writings/w2');
  });

  it('invites to write the first one when there is none', async () => {
    await open([]);
    expect(text(el.querySelector('.empty p'))).toBe('Chưa có bài viết. Học tới bước Viết của một bài để viết bài đầu tiên.');
    expect(el.querySelector('.empty a')?.getAttribute('href')).toBe('/lessons');
  });
});
