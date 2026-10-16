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
      http.expectOne('/goals').flush({
        active: goalLevel ? { level: goalLevel, topicId: 't', topicName: 'T' } : null,
        others: [],
      });
    }
    await settle();
  };
  const flushLevel = async (level: string, body: GrammarListData = list) => {
    http.expectOne((r) => r.url === '/grammar' && r.params.get('level') === level).flush(body);
    await settle();
  };

  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });
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
    http.expectNone('/goals');
    await flushLevel('A2');
    expect(el.querySelector('[role="radio"][aria-checked="true"]')?.textContent?.trim()).toBe('A2');
  });

  it('lists the points with a status in words, and the next point to continue', async () => {
    await setup('A1');
    await flushLevel('A1');
    expect(text(el.querySelector('.next'))).toContain('Làm bài kiểm tra');
    expect(text(el.querySelector('.next'))).toContain('Mạo từ');
    expect(el.querySelector('.next')?.getAttribute('href')).toBe('/grammar/a1-articles?step=test');
    // The open points only; the one without content is behind the collapsed "Sắp có" row.
    const rows = Array.from(el.querySelectorAll('.point'));
    expect(rows.length).toBe(2);
    expect(text(rows[0].querySelector('.state'))).toBe('Đã nắm vững · 90%');
    expect(text(rows[1].querySelector('.state'))).toBe('Đang học');
    expect(text(rows[0].querySelector('.point-en'))).toBe('The verb to be');
    expect(rows[0].querySelector('.point-hint')).toBeNull();
    expect(text(el.querySelector('.next-label'))).toBe('Học tiếp · Điểm 2/3');
    expect(text(el.querySelector('.mastery-label'))).toBe('Đã nắm vững trình độ A1 1/3');
    expect(el.querySelector('.track')?.getAttribute('aria-valuenow')).toBe('1');
  });

  it('gathers the points that are not published in one collapsed "Sắp có" row, after the open ones', async () => {
    await setup('A1');
    // In curriculum order the unpublished point comes first; it still goes after the open ones.
    await flushLevel('A1', { next: 'a1-articles', points: [list.points[2], list.points[0], list.points[1]] });
    expect(text(el.querySelector('.group-title'))).toBe('Đã mở · 2 điểm');
    expect(Array.from(el.querySelectorAll('a.point .point-title')).map((t) => text(t))).toEqual(['Động từ to be', 'Mạo từ']);
    expect(el.querySelector('.point.is-soon')).toBeNull();

    const toggle = el.querySelector<HTMLButtonElement>('.soon-toggle')!;
    expect(text(toggle.querySelector('.soon-title'))).toBe('Sắp có · 1 điểm');
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(toggle.compareDocumentPosition(el.querySelector('a.point')!) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy();
    toggle.click();
    await settle();
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    const soon = el.querySelector('#grammar-soon .point.is-soon')!;
    expect(soon.tagName).toBe('DIV');
    expect(text(soon.querySelector('.point-title'))).toBe('Danh từ số nhiều');
    expect(text(soon.querySelector('.state'))).toBe('Sắp có');
    expect(el.querySelectorAll('a.point').length).toBe(2);

    // Another level starts collapsed again.
    (Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]')).find((b) => text(b) === 'A2')!).click();
    await settle();
    await flushLevel('A2');
    expect(el.querySelector('.soon-toggle')?.getAttribute('aria-expanded')).toBe('false');
  });

  it('marks a point being learnt that has wrong answers to redo', async () => {
    await setup('A1');
    await flushLevel('A1', {
      next: 'a1-articles',
      points: [
        item({ status: 'mastered', bestMastery: 90, weakCount: 1 }),
        item({ id: 'a1-articles', titleVi: 'Mạo từ', status: 'learning', weakCount: 2 }),
      ],
    });
    const states = Array.from(el.querySelectorAll('.point .state'));
    expect(text(states[0])).toBe('Đã nắm vững · 90%');
    expect(states[0].classList).toContain('is-mastered');
    expect(text(states[1])).toBe('2 câu sai cần luyện');
    expect(states[1].classList).toContain('is-weak');
  });

  it('leads the card straight to what is left on the next point', async () => {
    const next = (over: Partial<GrammarListItem>): GrammarListData => ({
      next: 'a1-articles',
      points: [item({ id: 'a1-articles', titleVi: 'Mạo từ', ...over })],
    });
    const link = () => el.querySelector('.next')!;
    await setup('A1');
    await flushLevel('A1', next({ status: 'learning', weakCount: 2 }));
    expect(text(link())).toContain('Luyện lại 2 câu sai');
    expect(link().getAttribute('href')).toBe('/grammar/a1-articles?step=practice&redo=weak');

    // A point never practised: its start, or the step where it was left in this browser.
    (Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]')).find((b) => text(b) === 'A2')!).click();
    await settle();
    await flushLevel('A2', next({ status: 'new' }));
    expect(text(link())).toContain('Bắt đầu học');
    expect(link().getAttribute('href')).toBe('/grammar/a1-articles');

    localStorage.setItem('luna.grammar.step.a1-articles', '1');
    (Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]')).find((b) => text(b) === 'A1')!).click();
    await settle();
    await flushLevel('A1', next({ status: 'new' }));
    expect(text(link())).toContain('Tiếp tục: Bước 2');
    expect(link().getAttribute('href')).toBe('/grammar/a1-articles?step=structure');
  });

  it('says every open point is mastered when there is no next one', async () => {
    await setup('A1');
    await flushLevel('A1', { next: '', points: [item({ status: 'mastered', bestMastery: 90 })] });
    expect(el.querySelector('.next')).toBeNull();
    expect(text(el.querySelector('.hero-done'))).toContain('nắm vững mọi điểm đang mở');
    expect(text(el.querySelector('.mastery-count'))).toBe('1/1');
  });

  it('shows no "Sắp có" row when every point is open', async () => {
    await setup('A1');
    await flushLevel('A1', { next: 'a1-articles', points: [list.points[0], list.points[1]] });
    expect(el.querySelector('.soon-toggle')).toBeNull();
  });

  it('shows an empty state', async () => {
    await setup('A1');
    await flushLevel('A1', { points: [], next: '' });
    expect(text(el.querySelector('.empty'))).toContain('chưa có điểm ngữ pháp');
    expect(el.querySelector('.next')).toBeNull();
  });

  it('shows an error with a retry', async () => {
    await setup('A1');
    http.expectOne((r) => r.url === '/grammar').flush({}, { status: 500, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không tải được danh sách ngữ pháp');
    Array.from(el.querySelectorAll('button')).find((b) => text(b) === 'Thử lại')!.click();
    await settle();
    await flushLevel('A1');
    expect(el.querySelector('[role="alert"]')).toBeNull();
    expect(el.querySelectorAll('.point').length).toBe(2);
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
