import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Stats as StatsData } from '../../core/models/dashboard';
import { Stats } from './stats';

const sample: StatsData = {
  cards: 25,
  dictation: { sentences: 40, correctWords: 328, totalWords: 400, rate: 0.82 },
  lessons: { read: 6, listen: 5, completed: 5 },
  reading: { answered: 0, correct: 0, rate: null },
};

describe('Stats', () => {
  let fixture: ComponentFixture<Stats>;
  let controller: HttpTestingController;
  let el: HTMLElement;

  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  const open = async () => {
    fixture = TestBed.createComponent(Stats);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Stats],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
  });

  afterEach(() => controller.verify());

  it('shows words, dictation and lessons', async () => {
    await open();
    controller.expectOne('/api/stats').flush(sample);
    await fixture.whenStable();
    const page = text(el);
    expect(page).toContain('25 từ đã học');
    expect(page).toContain('40 câu đã chép chính tả');
    expect(text(el.querySelector('.rate'))).toBe('Tỷ lệ đúng 82%');
    expect(page).toContain('Đọc: 6 bài');
    expect(page).toContain('Nghe: 5 bài');
    expect(page).toContain('Hoàn thành: 5 bài');
    expect(el.querySelector('a[href="/"]')).not.toBeNull();
  });

  it('shows zeros and "—" for a new learner', async () => {
    await open();
    controller.expectOne('/api/stats').flush({
      cards: 0,
      dictation: { sentences: 0, correctWords: 0, totalWords: 0, rate: null },
      lessons: { read: 0, listen: 0, completed: 0 },
      reading: { answered: 0, correct: 0, rate: null },
    });
    await fixture.whenStable();
    expect(text(el)).toContain('0 từ đã học');
    expect(text(el.querySelector('.rate'))).toBe('Tỷ lệ đúng —');
    expect(text(el.querySelector('.reading .rate'))).toBe('Trả lời đúng —');
    expect(text(el.querySelector('.reading'))).toContain('Trả lời câu hỏi ở bước Đọc để xem tỷ lệ.');
  });

  it('shows the comprehension rate (F15)', async () => {
    await open();
    controller.expectOne('/api/stats').flush({ ...sample, reading: { answered: 12, correct: 9, rate: 0.75 } });
    await fixture.whenStable();
    expect(text(el.querySelector('.reading dt'))).toBe('Hiểu bài');
    expect(text(el.querySelector('.reading .rate'))).toBe('Trả lời đúng 75% (12 câu)');
  });

  it('rounds the rate', async () => {
    await open();
    controller.expectOne('/api/stats').flush({ ...sample, dictation: { ...sample.dictation, rate: 2 / 3 } });
    await fixture.whenStable();
    expect(text(el.querySelector('.rate'))).toBe('Tỷ lệ đúng 67%');
  });

  it('offers a retry when loading fails', async () => {
    await open();
    controller.expectOne('/api/stats').flush('down', { status: 500, statusText: 'Error' });
    await fixture.whenStable();
    expect(text(el.querySelector('[role="alert"] p'))).toBe('Không tải được thống kê.');
    (el.querySelector('[role="alert"] button') as HTMLButtonElement).click();
    controller.expectOne('/api/stats').flush(sample);
    await fixture.whenStable();
    expect(text(el)).toContain('25 từ đã học');
  });
});
