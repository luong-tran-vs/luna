import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import {
  CurriculumBoard,
  CurriculumLesson,
  CurriculumSlot,
  CurriculumTopic,
  LessonChecks,
} from '../../../core/models/curriculum';
import { Curriculum } from './curriculum';

const toBe = { id: 'a1-to-be', titleVi: 'to be', published: false };
const pronouns = { id: 'a1-pronouns', titleVi: 'đại từ chủ ngữ', published: true };

const lesson = (
  id: string,
  draft: boolean,
  annotationStatus: CurriculumLesson['annotationStatus'] = 'done',
) => ({
  id,
  title: `Bài ${id}`,
  draft,
  annotationStatus,
});

const checks: LessonChecks = {
  items: [
    { key: 'grammar', ok: true, warn: false, textVi: 'Dùng "to be" 8 lần (cần ≥ 3)' },
    { key: 'level', ok: false, warn: true, textVi: '1 từ trên A1' },
    { key: 'question', ok: false, warn: false, textVi: 'Chưa có câu hỏi' },
  ],
  above: [{ text: 'festival', level: 'B1' }],
};

const slot = (key: string, title: string, extra: Partial<CurriculumSlot> = {}): CurriculumSlot => ({
  key,
  kind: 'main',
  title,
  idea: `Ý tưởng ${title}`,
  grammar: [toBe],
  lesson: null,
  checks: null,
  ...extra,
});

const topic = (
  number: number,
  topicId: string,
  slots: CurriculumSlot[],
  extra: Partial<CurriculumTopic> = {},
): CurriculumTopic => ({
  number,
  name: `Chủ đề ${number}`,
  canDoVi: 'chào hỏi',
  topicId,
  position: number,
  grammar: [toBe, pronouns],
  slots,
  others: [],
  ...extra,
});

/**
 * Topic 1: an empty main slot, a draft and a published lesson, 2 empty extras and another (draft)
 * lesson. Topic 2: 2 empty slots.
 */
const board = (applied = true): CurriculumBoard => ({
  level: 'A1',
  applied,
  main: 4,
  extra: 3,
  published: 1,
  drafts: 1,
  topics: [
    topic(
      1,
      applied ? 't1' : '',
      [
        slot('a1-01-1', 'Hello! I am Minh'),
        slot('a1-01-2', 'Where Are You From?', { lesson: lesson('d1', true, 'running'), checks }),
        slot('a1-01-3', 'This Is My Friend Anna', { lesson: lesson('p1', false) }),
        slot('a1-01-4', 'At the Language Club', { kind: 'extra' }),
        slot('a1-01-5', 'A New Classmate', { kind: 'extra' }),
      ],
      { others: [lesson('o1', true)] },
    ),
    topic(
      2,
      applied ? 't2' : '',
      [slot('a1-02-1', 'My Family'), slot('a1-02-2', 'My Brother', { kind: 'extra' })],
      {
        grammar: [pronouns],
      },
    ),
  ],
});

describe('Curriculum', () => {
  let fixture: ComponentFixture<Curriculum>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const topicCard = (n: number) => el.querySelectorAll<HTMLElement>('.topic')[n - 1];
  const toggleOf = (n: number) => el.querySelector<HTMLButtonElement>(`#topic-toggle-${n}`)!;
  const bodyOf = (n: number) => el.querySelector<HTMLElement>(`#topic-body-${n}`)!;
  const row = (id: string) =>
    el.querySelector<HTMLElement>(`#${id}`)?.closest<HTMLElement>('lu-lesson-row') ?? null;
  const slotRow = (key: string) => row(`slot-${key}`)!;
  const button = (root: Element, label: string) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const actions = (r: Element) =>
    Array.from(r.querySelectorAll('.actions .act')).map((a) => {
      const shown = a.cloneNode(true) as Element;
      shown.querySelectorAll('.visually-hidden').forEach((n) => n.remove());
      return text(shown);
    });
  const statusOf = (n: number) => text(topicCard(n).querySelector('.topic-status'));
  const alertOf = (n: number) => text(topicCard(n).querySelector('[role="alert"]'));
  const chips = () => Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
  const dialog = () => el.querySelector('[role="alertdialog"]');

  const configure = async () => {
    await TestBed.configureTestingModule({
      imports: [Curriculum],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Curriculum);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };
  const expectBoard = () =>
    http.expectOne((r) => r.url === '/admin/curriculum' && r.method === 'GET');
  const setup = async (data: CurriculumBoard = board()) => {
    await configure();
    const req = expectBoard();
    expect(req.request.params.get('level')).toBe('A1');
    req.flush(data);
    await settle();
  };
  const expectGenerate = (key: string) =>
    http.expectOne(
      (r) => r.url === `/admin/curriculum/slots/${key}/generate` && r.method === 'POST',
    );
  const expectRemove = (id: string) =>
    http.expectOne(
      (r) => r.url === `/admin/curriculum/lessons/${id}/remove` && r.method === 'POST',
    );
  const draftOf = (s: CurriculumSlot): CurriculumSlot => ({
    ...s,
    lesson: lesson(`new-${s.key}`, true),
    checks,
  });

  afterEach(() => http.verify());

  describe('board', () => {
    beforeEach(() => setup());

    it('shows the totals and the progress in the header', () => {
      expect(text(el.querySelector('h1'))).toBe('Giáo trình A1');
      expect(text(el.querySelector('.summary'))).toBe('1/7 bài đã đăng · 1 nháp · 5 ô trống');
      expect(el.querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe('1');
      expect(el.querySelectorAll('.topic').length).toBe(2);
    });

    it('offers the apply button as a small secondary action once applied', () => {
      const apply = button(el, 'Áp dụng thứ tự');
      expect(apply.className).toBe('btn btn-sm');
      expect(apply.getAttribute('aria-describedby')).toBe('apply-help');
      expect(el.querySelector('.banner-warning')).toBeNull();
    });

    it('shows the filters with the number of lines of each state', () => {
      expect(el.querySelector('[role="radiogroup"]')?.getAttribute('aria-label')).toBe('Lọc bài');
      expect(chips().map(text)).toEqual(['Tất cả', 'Còn trống (5)', 'Nháp (2)', 'Đã đăng (1)']);
      expect(chips().map((c) => c.getAttribute('aria-checked'))).toEqual([
        'true',
        'false',
        'false',
        'false',
      ]);
      expect(chips().map((c) => c.tabIndex)).toEqual([0, -1, -1, -1]);
    });

    it('opens the topics with a draft or the first empty slot, others are collapsed', async () => {
      expect(toggleOf(1).getAttribute('aria-expanded')).toBe('true');
      expect(toggleOf(1).getAttribute('aria-controls')).toBe('topic-body-1');
      expect(bodyOf(1).hidden).toBe(false);
      expect(toggleOf(2).getAttribute('aria-expanded')).toBe('false');
      expect(bodyOf(2).hidden).toBe(true);

      toggleOf(2).click();
      await settle();
      expect(toggleOf(2).getAttribute('aria-expanded')).toBe('true');
      expect(bodyOf(2).hidden).toBe(false);
      toggleOf(1).click();
      await settle();
      expect(bodyOf(1).hidden).toBe(true);
    });

    it('shows the topic header in one line with a short grammar warning', () => {
      expect(text(toggleOf(1).querySelector('.name'))).toBe('Chủ đề 1: Chủ đề 1');
      expect(text(toggleOf(1).querySelector('.count'))).toBe('2/5 ô có bài');
      expect(text(toggleOf(1).querySelector('.warn'))).toBe('⚠ 1 bài ngữ pháp chưa đăng');
      expect(toggleOf(2).querySelector('.warn')).toBeNull();
      expect(text(button(topicCard(1), 'Sinh ô trống'))).toBe(
        'Sinh ô trống (3) của chủ đề Chủ đề 1',
      );
    });

    it('lists the grammar inside, linking an unpublished point to its admin page', () => {
      expect(text(bodyOf(1).querySelector('.can-do'))).toBe('Học xong: chào hỏi');
      const links = bodyOf(1).querySelectorAll<HTMLAnchorElement>('.missing-grammar');
      expect(links.length).toBe(1);
      expect(text(links[0])).toBe('to be (chưa đăng)');
      expect(links[0].getAttribute('href')).toBe('/admin/grammar/a1-to-be');
      expect(text(bodyOf(1).querySelector('.grammar'))).toContain('đại từ chủ ngữ');
      expect(bodyOf(2).querySelector('.missing-grammar')).toBeNull();
    });

    it('lists main lessons first, then extras and the other lessons of the topic', () => {
      expect(bodyOf(1).querySelectorAll('.main-rows lu-lesson-row').length).toBe(3);
      expect(bodyOf(1).querySelectorAll('.extra-rows lu-lesson-row').length).toBe(2);
      expect(bodyOf(1).querySelectorAll('.other-rows lu-lesson-row').length).toBe(1);
      expect(Array.from(bodyOf(1).querySelectorAll('.sub')).map(text)).toEqual([
        'Bài thêm',
        'Bài khác trong chủ đề',
      ]);
      expect(bodyOf(2).querySelector('.other-rows')).toBeNull();
    });

    it('shows an empty slot with its grammar, idea and generate button', () => {
      const r = slotRow('a1-01-1');
      expect(r.dataset['state']).toBe('empty');
      expect(text(r.querySelector('.pill'))).toBe('Trống');
      expect(text(r.querySelector('.title'))).toBe('Hello! I am Minh');
      expect(text(r.querySelector('.grammar'))).toBe('Ngữ pháp: to be');
      expect(text(r.querySelector('.idea-text'))).toBe('Ý tưởng: Ý tưởng Hello! I am Minh');
      expect(actions(r)).toEqual(['Sinh bài']);
      expect(button(r, 'Sinh bài').disabled).toBe(false);
    });

    it('shows a draft with a checks summary, publish disabled until annotated and remove', () => {
      const r = slotRow('a1-01-2');
      expect(r.dataset['state']).toBe('draft');
      expect(text(r.querySelector('.pill'))).toBe('Nháp');
      expect(text(r.querySelector('.title'))).toBe('Bài d1');
      expect(actions(r)).toEqual(['✓1!1✕1', 'Xem', 'Đăng', 'Gỡ']);
      expect(r.querySelector('.actions a')?.getAttribute('href')).toBe('/admin/lessons/d1');
      const publish = button(r, 'Đăng');
      expect(publish.disabled).toBe(true);
      expect(publish.getAttribute('aria-describedby')).toBe('slot-a1-01-2-hint');
      expect(text(r.querySelector('.hint'))).toBe('Đang chú thích, đăng được khi xong.');
    });

    it('opens and closes the full checks of a draft', async () => {
      const r = slotRow('a1-01-2');
      const toggle = r.querySelector<HTMLButtonElement>('.checks-toggle')!;
      expect(text(toggle)).toContain('Kết quả kiểm tra bài Bài d1: 1 đạt, 1 cảnh báo, 1 chưa đạt');
      expect(toggle.getAttribute('aria-expanded')).toBe('false');
      const panel = r.querySelector<HTMLElement>('#slot-a1-01-2-checks')!;
      expect(toggle.getAttribute('aria-controls')).toBe(panel.id);
      expect(panel.hidden).toBe(true);

      toggle.click();
      await settle();
      expect(toggle.getAttribute('aria-expanded')).toBe('true');
      expect(panel.hidden).toBe(false);
      const items = Array.from(panel.querySelectorAll('.checks li'));
      expect(items.map((li) => li.getAttribute('data-result'))).toEqual(['ok', 'warn', 'bad']);
      expect(items.map((li) => text(li.querySelector('.mark')))).toEqual(['✓', '!', '✕']);
      expect(text(items[2])).toContain('Chưa đạt: Chưa có câu hỏi');
      expect(text(panel.querySelector('.above-word'))).toBe('festival · B1');
      expect(button(panel, 'Kiểm tra lại')).toBeTruthy();

      toggle.click();
      await settle();
      expect(panel.hidden).toBe(true);
    });

    it('shows a published lesson with view and remove only', () => {
      const r = slotRow('a1-01-3');
      expect(r.dataset['state']).toBe('published');
      expect(text(r.querySelector('.pill'))).toBe('Đã đăng');
      expect(actions(r)).toEqual(['Xem', 'Gỡ']);
      expect(r.querySelector('.actions a')?.getAttribute('href')).toBe('/admin/lessons/p1');
    });

    it('shows another lesson of the topic with view and remove', () => {
      const r = row('other-o1')!;
      expect(text(r.querySelector('.pill'))).toBe('Nháp');
      expect(text(r.querySelector('.title'))).toBe('Bài o1');
      expect(actions(r)).toEqual(['Xem', 'Gỡ']);
      expect(r.querySelector('.actions a')?.getAttribute('href')).toBe('/admin/lessons/o1');
    });
  });

  it('opens the first topic with an empty slot when no topic has a draft', async () => {
    const data = board();
    data.topics[0].slots = [slot('a1-01-3', 'Anna', { lesson: lesson('p1', false) })];
    data.topics[0].others = [];
    await setup(data);
    expect(toggleOf(1).getAttribute('aria-expanded')).toBe('false');
    expect(toggleOf(2).getAttribute('aria-expanded')).toBe('true');
    expect(button(topicCard(1), 'Sinh ô trống')).toBeUndefined();
  });

  describe('filters', () => {
    beforeEach(() => setup());

    it('shows only the drafts, hiding topics without one', async () => {
      chips()[2].click();
      await settle();
      expect(chips()[2].getAttribute('aria-checked')).toBe('true');
      expect(topicCard(1).hidden).toBe(false);
      expect(topicCard(2).hidden).toBe(true);
      expect(
        Array.from(bodyOf(1).querySelectorAll<HTMLElement>('lu-lesson-row')).map(
          (r) => r.dataset['state'],
        ),
      ).toEqual(['draft', 'draft']);
      expect(bodyOf(1).querySelector('.extra-rows')).toBeNull();
    });

    it('shows the empty slots and opens every topic having one', async () => {
      chips()[1].click();
      await settle();
      expect(topicCard(2).hidden).toBe(false);
      expect(bodyOf(2).hidden).toBe(false);
      expect(bodyOf(1).querySelectorAll('lu-lesson-row').length).toBe(3);
      expect(row('slot-a1-01-2')).toBeNull();
    });

    it('shows the published lessons, then everything again', async () => {
      chips()[3].click();
      await settle();
      expect(topicCard(2).hidden).toBe(true);
      expect(bodyOf(1).querySelectorAll('lu-lesson-row').length).toBe(1);
      chips()[0].click();
      await settle();
      expect(topicCard(2).hidden).toBe(false);
      expect(bodyOf(1).querySelectorAll('lu-lesson-row').length).toBe(6);
      expect(bodyOf(2).hidden).toBe(true);
    });

    it('moves the choice with the arrow keys', async () => {
      chips()[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
      await settle();
      expect(chips()[1].getAttribute('aria-checked')).toBe('true');
      expect(document.activeElement).toBe(chips()[1]);
      chips()[1].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
      chips()[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
      await settle();
      expect(chips()[3].getAttribute('aria-checked')).toBe('true');
    });
  });

  describe('not applied', () => {
    beforeEach(() => setup(board(false)));

    it('highlights apply, explains it and disables generation', () => {
      expect(button(el, 'Áp dụng thứ tự').className).toBe('btn-primary');
      const help = el.querySelector('.banner-warning')!;
      expect(help.id).toBe('apply-help');
      expect(text(help)).toContain('Chưa áp dụng giáo trình');
      const gen = button(slotRow('a1-01-1'), 'Sinh bài');
      expect(gen.disabled).toBe(true);
      expect(gen.getAttribute('aria-describedby')).toBe('apply-help');
      expect(gen.title).toBe('Bấm “Áp dụng thứ tự” trước để tạo chủ đề.');
      expect(button(topicCard(1), 'Sinh ô trống').disabled).toBe(true);
    });

    it('applies after confirming and shows the new board', async () => {
      button(el, 'Áp dụng thứ tự').click();
      await settle();
      expect(text(dialog()!.querySelector('h2'))).toBe('Áp dụng thứ tự giáo trình?');
      dialog()!.querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await settle();
      expect(dialog()).toBeNull();
      const req = http.expectOne((r) => r.url === '/admin/curriculum/apply' && r.method === 'POST');
      expect(req.request.params.get('level')).toBe('A1');
      req.flush(board(true));
      await settle();
      expect(el.querySelector('.banner-warning')).toBeNull();
      expect(button(slotRow('a1-01-1'), 'Sinh bài').disabled).toBe(false);
    });

    it('does nothing when the confirmation is cancelled', async () => {
      button(el, 'Áp dụng thứ tự').click();
      await settle();
      dialog()!.querySelector<HTMLButtonElement>('.btn')!.click();
      await settle();
      expect(dialog()).toBeNull();
      http.expectNone('/admin/curriculum/apply');
    });
  });

  describe('generating one slot', () => {
    beforeEach(() => setup());

    it('posts the slot, then shows the draft and the new counts', async () => {
      toggleOf(2).click();
      await settle();
      button(slotRow('a1-01-1'), 'Sinh bài').click();
      await settle();
      expect(topicCard(1).getAttribute('aria-busy')).toBe('true');
      expect(slotRow('a1-01-1').getAttribute('aria-busy')).toBe('true');
      expect(statusOf(1)).toContain('Đang sinh bài…');
      // Another generation cannot start meanwhile.
      const other = button(slotRow('a1-02-1'), 'Sinh bài');
      expect(other.disabled).toBe(true);
      expect(other.title).toBe('Đang có bài khác được sinh, đợi xong đã.');
      expect(button(topicCard(2), 'Sinh ô trống').disabled).toBe(true);

      expectGenerate('a1-01-1').flush(draftOf(board().topics[0].slots[0]));
      await settle();
      expect(slotRow('a1-01-1').dataset['state']).toBe('draft');
      expect(text(el.querySelector('.summary'))).toBe('1/7 bài đã đăng · 2 nháp · 4 ô trống');
      expect(statusOf(1)).toBe('Đã sinh bản nháp “Hello! I am Minh”.');
      expect(topicCard(1).getAttribute('aria-busy')).toBe('false');
      expect(button(slotRow('a1-02-1'), 'Sinh bài').disabled).toBe(false);
    });

    it("shows the server's message when it fails", async () => {
      button(slotRow('a1-01-1'), 'Sinh bài').click();
      await settle();
      expectGenerate('a1-01-1').flush(
        { error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' },
        { status: 429, statusText: 'Too Many Requests' },
      );
      await settle();
      expect(alertOf(1)).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
      expect(slotRow('a1-01-1').dataset['state']).toBe('empty');
      expect(button(slotRow('a1-01-1'), 'Sinh bài').disabled).toBe(false);
    });
  });

  describe('generating the empty slots of a topic', () => {
    beforeEach(() => setup());

    it('generates them one after another with progress', async () => {
      const empty = board().topics[0].slots.filter((s) => !s.lesson);
      button(topicCard(1), 'Sinh ô trống').click();
      await settle();
      expect(text(button(topicCard(1), 'Sinh ô trống'))).toContain('(3)');
      for (const [i, s] of empty.entries()) {
        expect(statusOf(1)).toContain(`Đang sinh ${i + 1}/3…`);
        expectGenerate(s.key).flush(draftOf(s));
        await settle();
      }
      http.expectNone((r) => r.url.includes('/generate'));
      expect(statusOf(1)).toBe('Đã sinh 3 bản nháp.');
      expect(text(el.querySelector('.summary'))).toContain('4 nháp');
      expect(button(topicCard(1), 'Sinh ô trống')).toBeUndefined();
    });

    it('opens a collapsed topic while generating it', async () => {
      expect(bodyOf(2).hidden).toBe(true);
      button(topicCard(2), 'Sinh ô trống').click();
      await settle();
      expect(bodyOf(2).hidden).toBe(false);
      expect(toggleOf(2).getAttribute('aria-expanded')).toBe('true');
      const slots = board().topics[1].slots;
      expectGenerate(slots[0].key).flush(draftOf(slots[0]));
      await settle();
      expectGenerate(slots[1].key).flush(draftOf(slots[1]));
      await settle();
      expect(statusOf(2)).toBe('Đã sinh 2 bản nháp.');
    });

    it('stops at the first error and shows it', async () => {
      const empty = board().topics[0].slots.filter((s) => !s.lesson);
      button(topicCard(1), 'Sinh ô trống').click();
      await settle();
      expectGenerate(empty[0].key).flush(draftOf(empty[0]));
      await settle();
      expectGenerate(empty[1].key).flush(
        { error: 'ai_unusable', message: 'AI trả về nội dung không dùng được, vui lòng thử lại.' },
        { status: 502, statusText: 'Bad Gateway' },
      );
      await settle();
      http.expectNone(`/admin/curriculum/slots/${empty[2].key}/generate`);
      expect(alertOf(1)).toBe(
        'Dừng ở bài “At the Language Club”: AI trả về nội dung không dùng được, vui lòng thử lại.',
      );
      expect(slotRow(empty[0].key).dataset['state']).toBe('draft');
      expect(slotRow(empty[2].key).dataset['state']).toBe('empty');
      expect(text(button(topicCard(1), 'Sinh ô trống'))).toContain('(2)');
    });
  });

  describe('a draft', () => {
    it('is published, then the board is read again', async () => {
      const data = board();
      data.topics[0].slots[1].lesson = lesson('d1', true, 'done');
      await setup(data);
      const publish = button(slotRow('a1-01-2'), 'Đăng');
      expect(publish.disabled).toBe(false);
      expect(slotRow('a1-01-2').querySelector('.hint')).toBeNull();
      publish.click();
      await settle();
      const req = http.expectOne(
        (r) => r.url === '/admin/lessons/d1/published' && r.method === 'PUT',
      );
      expect(req.request.body).toEqual({ published: true });
      req.flush({ lesson: { id: 'd1' } });
      await settle();
      const after = board();
      after.topics[0].slots[1] = {
        ...after.topics[0].slots[1],
        lesson: lesson('d1', false),
        checks: null,
      };
      expectBoard().flush({ ...after, published: 2, drafts: 0 });
      await settle();
      expect(slotRow('a1-01-2').dataset['state']).toBe('published');
      expect(text(el.querySelector('.summary'))).toBe('2/7 bài đã đăng · 0 nháp · 5 ô trống');
      expect(statusOf(1)).toBe('Đã đăng bài.');
    });

    it('has its checks read again', async () => {
      await setup();
      const r = slotRow('a1-01-2');
      r.querySelector<HTMLButtonElement>('.checks-toggle')!.click();
      await settle();
      button(r, 'Kiểm tra lại').click();
      await settle();
      http.expectOne('/admin/curriculum/checks/d1').flush({
        items: [{ key: 'question', ok: true, warn: false, textVi: 'Có câu hỏi' }],
        above: [],
      });
      await settle();
      const items = slotRow('a1-01-2').querySelectorAll('.checks li');
      expect(items.length).toBe(1);
      expect(items[0].getAttribute('data-result')).toBe('ok');
      expect(slotRow('a1-01-2').querySelector('.above')).toBeNull();
      expect(text(slotRow('a1-01-2').querySelector('.checks-toggle [aria-hidden]'))).toBe('✓1');
      expect(statusOf(1)).toBe('Đã kiểm tra lại “Where Are You From?”.');
    });
  });

  describe('removing a lesson', () => {
    beforeEach(() => setup());

    const removeFrom = async (r: HTMLElement) => {
      button(r, 'Gỡ').click();
      await settle();
    };
    const confirm = async () => {
      dialog()!.querySelector<HTMLButtonElement>('.btn-primary')!.click();
      await settle();
    };

    it('removes and deletes a draft after confirming, the slot is empty again', async () => {
      await removeFrom(slotRow('a1-01-2'));
      expect(text(dialog()!.querySelector('h2'))).toBe('Gỡ bài khỏi giáo trình?');
      expect(text(dialog()!.querySelector('.message'))).toBe(
        'Gỡ và xoá bài nháp “Bài d1”? Ô sẽ trống để sinh lại.',
      );
      expect(text(dialog()!.querySelector('.btn-primary'))).toBe('Gỡ');
      await confirm();
      expect(dialog()).toBeNull();
      expectRemove('d1').flush({ deleted: true });
      await settle();
      const after = board();
      after.topics[0].slots[1] = { ...after.topics[0].slots[1], lesson: null, checks: null };
      expectBoard().flush({ ...after, drafts: 0 });
      await settle();
      expect(slotRow('a1-01-2').dataset['state']).toBe('empty');
      expect(statusOf(1)).toBe('Đã gỡ và xoá bài nháp “Bài d1”.');
      expect(text(el.querySelector('.summary'))).toBe('1/7 bài đã đăng · 0 nháp · 6 ô trống');
      expect(document.activeElement).toBe(toggleOf(1));
    });

    it('takes a published lesson out of the course and keeps it', async () => {
      await removeFrom(slotRow('a1-01-3'));
      expect(text(dialog()!.querySelector('.message'))).toBe(
        'Gỡ “Bài p1” khỏi giáo trình? Bài vẫn còn trong danh sách Bài học, người học đã học vẫn giữ tiến độ.',
      );
      await confirm();
      expectRemove('p1').flush({ deleted: false });
      await settle();
      const after = board();
      after.topics[0].slots[2] = { ...after.topics[0].slots[2], lesson: null };
      after.topics[0].others = [...after.topics[0].others, lesson('p1', false)];
      expectBoard().flush({ ...after, published: 0 });
      await settle();
      expect(statusOf(1)).toBe('Đã gỡ “Bài p1” khỏi giáo trình.');
      expect(slotRow('a1-01-3').dataset['state']).toBe('empty');
    });

    it('removes another lesson of the topic', async () => {
      await removeFrom(row('other-o1')!);
      await confirm();
      expectRemove('o1').flush({ deleted: true });
      await settle();
      const after = board();
      after.topics[0].others = [];
      expectBoard().flush(after);
      await settle();
      expect(row('other-o1')).toBeNull();
      expect(statusOf(1)).toBe('Đã gỡ và xoá bài nháp “Bài o1”.');
    });

    it('does nothing when cancelled and gives the focus back', async () => {
      const remove = button(slotRow('a1-01-3'), 'Gỡ');
      remove.focus();
      remove.click();
      await settle();
      dialog()!.querySelector<HTMLButtonElement>('.btn')!.click();
      await settle();
      expect(dialog()).toBeNull();
      http.expectNone((r) => r.url.includes('/remove'));
      expect(document.activeElement).toBe(remove);
      expect(statusOf(1)).toBe('');
    });

    it("shows the server's message when it fails", async () => {
      await removeFrom(slotRow('a1-01-3'));
      await confirm();
      expectRemove('p1').flush(
        { error: 'x', message: 'Không tìm thấy bài.' },
        { status: 404, statusText: 'Not Found' },
      );
      await settle();
      expect(alertOf(1)).toBe('Không tìm thấy bài.');
      expect(slotRow('a1-01-3').dataset['state']).toBe('published');
      http.expectNone((r) => r.url === '/admin/curriculum' && r.method === 'GET');
    });
  });

  it('shows a load error with a retry', async () => {
    await configure();
    http
      .expectOne('/admin/curriculum?level=A1')
      .flush({ error: 'x' }, { status: 500, statusText: 'Error' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe(
      'Không tải được giáo trình, vui lòng thử lại.',
    );
    button(el, 'Thử lại').click();
    await settle();
    http.expectOne('/admin/curriculum?level=A1').flush(board());
    await settle();
    expect(el.querySelectorAll('.topic').length).toBe(2);
  });
});
