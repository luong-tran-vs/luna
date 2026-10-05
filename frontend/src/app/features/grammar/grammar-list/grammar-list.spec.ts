import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { GrammarList as GrammarListData, GrammarListItem } from '../../../core/models/grammar-study';
import { GrammarList } from './grammar-list';

const item = (over: Partial<GrammarListItem>): GrammarListItem => ({
  id: 'a1-to-be', level: 'A1', titleVi: 'Động từ to be', titleEn: 'The verb to be', hintVi: 'am, is, are',
  available: true, status: 'new', bestMastery: 0, ...over,
});

const list: GrammarListData = {
  next: 'a1-articles',
  points: [
    item({ status: 'mastered', bestMastery: 90 }),
    item({ id: 'a1-articles', titleVi: 'Mạo từ', titleEn: 'Articles', status: 'learning' }),
    item({ id: 'a1-plural', titleVi: 'Danh từ số nhiều', titleEn: 'Plurals', available: false }),
  ],
};

describe('GrammarList', () => {
  let fixture: ComponentFixture<GrammarList>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const setup = async (goalLevel: string | null = 'B1') => {
    TestBed.configureTestingModule({
      imports: [GrammarList],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarList);
    el = fixture.nativeElement;
    await fixture.whenStable();
    if (goalLevel !== undefined) {
      http.expectOne('/api/goals').flush({
        active: goalLevel ? { level: goalLevel, topicId: 't', topicName: 'T' } : null,
        others: [],
      });
    }
    await settle();
  };
  const flushLevel = async (level: string, body: GrammarListData = list) => {
    http.expectOne((r) => r.url === '/api/grammar' && r.params.get('level') === level).flush(body);
    await settle();
  };

  beforeEach(() => sessionStorage.clear());
  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  it('starts at the level of the goal being studied', async () => {
    await setup('B1');
    await flushLevel('B1');
    expect(el.querySelector('[role="radio"][aria-checked="true"]')?.textContent?.trim()).toBe('B1');
  });

  it('starts at A1 without a goal, and remembers the chosen level in the session', async () => {
    await setup(null);
    await flushLevel('A1');
    (Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]')).find((b) => text(b) === 'A2')!).click();
    await settle();
    await flushLevel('A2');
    expect(sessionStorage.getItem('luna.grammar.level')).toBe('A2');

    fixture.destroy();
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      imports: [GrammarList],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarList);
    el = fixture.nativeElement;
    await settle();
    http.expectNone('/api/goals');
    await flushLevel('A2');
    expect(el.querySelector('[role="radio"][aria-checked="true"]')?.textContent?.trim()).toBe('A2');
  });

  it('lists the points with a status in words, and the next point to continue', async () => {
    await setup('A1');
    await flushLevel('A1');
    expect(text(el.querySelector('.next'))).toContain('Học tiếp');
    expect(text(el.querySelector('.next'))).toContain('Mạo từ');
    expect(el.querySelector('.next')?.getAttribute('href')).toBe('/grammar/a1-articles');
    const rows = Array.from(el.querySelectorAll('.point'));
    expect(rows.length).toBe(3);
    expect(text(rows[0].querySelector('.state'))).toBe('Đã nắm vững · 90%');
    expect(text(rows[1].querySelector('.state'))).toBe('Đang học');
    expect(text(rows[0].querySelector('.point-en'))).toBe('The verb to be');
    expect(text(rows[0].querySelector('.point-hint'))).toBe('am, is, are');
  });

  it('shows a point that is not published as faded "Sắp có" and not clickable', async () => {
    await setup('A1');
    await flushLevel('A1');
    const soon = el.querySelector('.point.is-soon')!;
    expect(soon.tagName).toBe('DIV');
    expect(text(soon.querySelector('.state'))).toBe('Sắp có');
    expect(el.querySelectorAll('a.point').length).toBe(2);
  });

  it('shows an empty state', async () => {
    await setup('A1');
    await flushLevel('A1', { points: [], next: '' });
    expect(text(el.querySelector('.empty'))).toContain('chưa có điểm ngữ pháp');
    expect(el.querySelector('.next')).toBeNull();
  });

  it('shows an error with a retry', async () => {
    await setup('A1');
    http.expectOne((r) => r.url === '/api/grammar').flush({}, { status: 500, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không tải được danh sách ngữ pháp');
    Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Thử lại')!.click();
    await settle();
    await flushLevel('A1');
    expect(el.querySelector('[role="alert"]')).toBeNull();
    expect(el.querySelectorAll('.point').length).toBe(3);
  });

  it('moves between levels with the arrow keys', async () => {
    await setup('A1');
    await flushLevel('A1');
    const first = el.querySelector<HTMLButtonElement>('[role="radio"]')!;
    first.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
    await settle();
    await flushLevel('A2');
    expect(el.querySelector('[role="radio"][aria-checked="true"]')?.textContent?.trim()).toBe('A2');
  });
});
