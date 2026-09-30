import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { CardPage, DayCard } from '../../../core/models/vocab';
import { Notebook } from './notebook';

function card(id: string, text: string, day: string, lessonId: string | null = 'l1'): DayCard {
  return {
    id, text, lemma: text, ipa: '/x/', meaningVi: `nghĩa ${text}`, contextSentence: `We ${text}.`, lessonId,
    source: lessonId ? 'ai' : 'manual', createdAt: `${day}T01:00:00Z`, due: `${day}T17:00:00Z`, reps: 0, state: 'new', day,
  };
}

const page1: CardPage = {
  cards: [card('c1', 'went', '2026-09-30'), card('c2', 'gave up', '2026-09-29'), card('c3', 'run', '2026-09-20')],
  hasMore: true,
  today: '2026-09-30',
  yesterday: '2026-09-29',
};

describe('Notebook', () => {
  let fixture: ComponentFixture<Notebook>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async (ms = 0) => {
    await new Promise((resolve) => setTimeout(resolve, ms));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string, root: ParentNode = el) =>
    Array.from(root.querySelectorAll('button')).find((b) => text(b) === label);
  const expectList = (): TestRequest => http.expectOne((r) => r.url === '/api/vocab/cards');
  const headings = () => Array.from(el.querySelectorAll('.day-heading')).map((h) => text(h));
  const words = () => Array.from(el.querySelectorAll('.card-item .word')).map((w) => text(w));

  const setup = async (page: CardPage = page1, manualCount = 1) => {
    TestBed.configureTestingModule({
      imports: [Notebook],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Notebook);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    expectList().flush(page);
    http.expectOne('/api/vocab/lessons').flush({ lessons: [{ id: 'l1', title: 'A day at the park', count: 3 }], manualCount });
    http.expectOne((r) => r.url === '/api/vocab/review/due').flush({ cards: [], total: 4, nextDue: null });
    await settle();
  };

  beforeEach(() => {
    vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined);
    vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => undefined);
  });

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  it('groups cards by the day they were saved', async () => {
    await setup();
    expect(headings()).toEqual(['Hôm nay', 'Hôm qua', '20/09/2026']);
    const first = el.querySelector('.card-item')!;
    expect(text(first.querySelector('.word'))).toBe('went');
    expect(text(first.querySelector('.ipa'))).toBe('/x/');
    expect(text(first.querySelector('.meaning'))).toBe('nghĩa went');
    expect(text(first.querySelector('.example'))).toBe('We went.');
  });

  it('loads more and merges a day split across pages', async () => {
    await setup();
    button('Xem thêm')!.click();
    await settle();
    const req = expectList();
    expect(req.request.params.get('page')).toBe('2');
    req.flush({ ...page1, cards: [card('c4', 'ran', '2026-09-20'), card('c5', 'saw', '2026-09-01')], hasMore: false });
    await settle();
    expect(headings()).toEqual(['Hôm nay', 'Hôm qua', '20/09/2026', '01/09/2026']);
    expect(words()).toEqual(['went', 'gave up', 'run', 'ran', 'saw']);
    expect(button('Xem thêm')).toBeUndefined();
  });

  it('searches by word after the learner stops typing', async () => {
    await setup();
    const search = el.querySelector<HTMLInputElement>('#notebook-search')!;
    search.value = 'giv';
    search.dispatchEvent(new Event('input'));
    await settle(350);
    const req = expectList();
    expect(req.request.params.get('q')).toBe('giv');
    expect(req.request.params.get('page')).toBe('1');
    req.flush({ ...page1, cards: [], hasMore: false });
    await settle();
    expect(el.textContent).toContain('Không tìm thấy từ nào');
  });

  it('filters by lesson, including cards added by hand', async () => {
    await setup();
    const select = el.querySelector<HTMLSelectElement>('#notebook-lesson')!;
    expect(Array.from(select.options).map((o) => text(o))).toEqual([
      'Tất cả bài',
      'A day at the park (3)',
      'Thẻ tự thêm (1)',
    ]);
    select.value = 'manual';
    select.dispatchEvent(new Event('change'));
    await settle();
    const req = expectList();
    expect(req.request.params.get('lessonId')).toBe('manual');
    req.flush({ ...page1, cards: [card('c9', 'hi', '2026-09-30', null)], hasMore: false });
    await settle();
    expect(words()).toEqual(['hi']);
  });

  it('guides the learner when the notebook is empty', async () => {
    await setup({ ...page1, cards: [], hasMore: false }, 0);
    expect(el.textContent).toContain('Sổ từ còn trống');
    expect(Array.from(el.querySelectorAll<HTMLOptionElement>('#notebook-lesson option')).map((o) => o.value)).toEqual(['', 'l1']);
  });

  it('plays a word', async () => {
    await setup();
    button('Nghe went')!.click();
    await settle();
    expect(el.querySelector('audio')!.getAttribute('src')).toBe('/api/tts/word?text=went');
  });

  it('links to review with the number of due cards', async () => {
    await setup();
    const link = el.querySelector<HTMLAnchorElement>('a[href="/vocabulary/review"]')!;
    expect(text(link)).toBe('Ôn tập (4)');
  });

  it('edits a card in place', async () => {
    await setup();
    button('Sửa went')!.click();
    await settle();
    const meaning = el.querySelector<HTMLInputElement>('#card-meaning')!;
    meaning.value = 'đã đi';
    meaning.dispatchEvent(new Event('input'));
    el.querySelector('lu-card-form form')!.dispatchEvent(new Event('submit'));
    await settle();
    http.expectOne('/api/vocab/cards/c1').flush({ card: { ...page1.cards[0], meaningVi: 'đã đi' } });
    await settle();
    expect(el.querySelector('lu-card-form')).toBeNull();
    expect(text(el.querySelector('.card-item .meaning'))).toBe('đã đi');
  });

  it('adds a card and reloads the list', async () => {
    await setup();
    button('Thêm thẻ')!.click();
    await settle();
    const form = el.querySelector('lu-card-form')!;
    for (const [id, value] of [['card-text', 'hello'], ['card-meaning', 'xin chào']]) {
      const input = form.querySelector<HTMLInputElement>(`#${id}`)!;
      input.value = value;
      input.dispatchEvent(new Event('input'));
    }
    form.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
    http.expectOne('/api/vocab/cards').flush({ card: card('c9', 'hello', '2026-09-30', null) }, { status: 201, statusText: 'Created' });
    await settle();
    expectList().flush({ ...page1, cards: [card('c9', 'hello', '2026-09-30', null), ...page1.cards] });
    http.expectOne('/api/vocab/lessons').flush({ lessons: [], manualCount: 1 });
    await settle();
    expect(words()[0]).toBe('hello');
    expect(el.querySelector('lu-card-form')).toBeNull();
  });

  it('deletes a card after confirmation', async () => {
    await setup();
    button('Xoá went')!.click();
    await settle();
    const dialog = el.querySelector('[role="alertdialog"]')!;
    expect(text(dialog)).toContain('went');
    button('Xoá', dialog)!.click();
    await settle();
    const req = http.expectOne('/api/vocab/cards/c1');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    // Lesson counts are refreshed after a delete.
    http.expectOne('/api/vocab/lessons').flush({ lessons: [{ id: 'l1', title: 'A day at the park', count: 2 }], manualCount: 1 });
    await settle();
    expect(words()).toEqual(['gave up', 'run']);
    expect(el.querySelector('[role="alertdialog"]')).toBeNull();
  });
});
