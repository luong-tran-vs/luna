import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import {
  CurriculumBoard,
  CurriculumPlan,
  CurriculumSlot,
  CurriculumTopic,
} from '../../../../core/models/curriculum';
import { CurriculumDraft } from './curriculum-draft';

const past = {
  id: 'a2-past-simple-regular',
  titleVi: 'Quá khứ đơn (có quy tắc)',
  published: false,
};
const will = { id: 'a2-will', titleVi: 'Will', published: true };

const slot = (n: number, title: string): CurriculumSlot => ({
  key: `a2-01-${n}`,
  kind: n <= 3 ? 'main' : 'extra',
  title,
  idea: `Idea ${title}`,
  grammar: [past],
  lesson: null,
  checks: null,
});

const topic = (number: number, name: string, slots: CurriculumSlot[]): CurriculumTopic => ({
  number,
  name,
  canDoVi: 'kể chuyện.',
  topicId: number === 1 ? 'travel' : '',
  position: 0,
  words: 0,
  grammar: [past],
  slots,
  others: [],
});

/** Topic 1 has 4 lessons, topics 2 and 3 none yet. */
const draft = (): CurriculumBoard => ({
  level: 'A2',
  status: 'draft',
  hasApproved: false,
  warnings: ['Mới có 3 chủ đề, nên có ít nhất 7.'],
  points: [past, will],
  applied: false,
  main: 3,
  extra: 1,
  published: 0,
  drafts: 0,
  topics: [
    topic(1, 'Du lịch', [
      slot(1, 'My Trip'),
      slot(2, 'At the Hotel'),
      slot(3, 'Lost Bag'),
      slot(4, 'Postcards'),
    ]),
    topic(2, 'Thể thao', []),
    topic(3, 'Sức khoẻ', []),
  ],
});

describe('CurriculumDraft', () => {
  let fixture: ComponentFixture<CurriculumDraft>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let emitted: CurriculumBoard[];

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (root: Element, label: string) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const topicCard = (n: number) => el.querySelectorAll<HTMLElement>('.topics > .topic')[n - 1];
  const titles = () =>
    Array.from(el.querySelectorAll('.topic-title')).map((h) => {
      const shown = h.cloneNode(true) as Element;
      shown.querySelectorAll('.visually-hidden, .number').forEach((n) => n.remove());
      return `${text(h.querySelector('.number'))} ${text(shown)}`;
    });
  const change = async (
    input: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement,
    value: string,
  ) => {
    input.value = value;
    input.dispatchEvent(new Event('change'));
    await settle();
  };
  const dialog = () => el.querySelector('[role="alertdialog"]');
  const confirm = async () => {
    dialog()!.querySelector<HTMLButtonElement>('.btn-primary')!.click();
    await settle();
  };
  const show = async (b: CurriculumBoard) => {
    fixture.componentRef.setInput('board', b);
    await settle();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CurriculumDraft],
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(CurriculumDraft);
    el = fixture.nativeElement as HTMLElement;
    emitted = [];
    // The page shows what the server returns, as the course page does.
    fixture.componentInstance.changed.subscribe((b) => {
      emitted.push(b);
      fixture.componentRef.setInput('board', b);
    });
    fixture.componentRef.setInput('topicNames', ['Du lịch', 'Gia đình']);
    await show(draft());
  });

  afterEach(() => http.verify());

  it('shows the topics, their lessons and what the course lacks', () => {
    expect(titles()).toEqual(['1 Du lịch', '2 Thể thao', '3 Sức khoẻ']);
    expect(text(el.querySelector('.intro .muted'))).toContain('3 chủ đề · 4 bài');
    expect(text(el.querySelector('.banner-warning li'))).toBe('Mới có 3 chủ đề, nên có ít nhất 7.');
    const kinds = Array.from(topicCard(1).querySelectorAll('.slot-kind')).map(text);
    expect(kinds).toEqual(['Bài chính 1', 'Bài chính 2', 'Bài chính 3', 'Bài thêm 1']);
    expect(text(topicCard(1).querySelector('.counts'))).toBe('3 bài chính · 1 bài thêm (nên 4–5)');
    expect(text(topicCard(1).querySelector('.field-hint'))).toBe('Đã có trong danh mục chủ đề.');
    expect(text(topicCard(2).querySelector('.field-hint'))).toBe('Chủ đề mới: được tạo khi duyệt.');
    expect(button(el, 'Soạn bài cho các chủ đề chưa có bài')).toBeDefined();
    expect(button(el, 'Lưu bản nháp').disabled).toBe(true);
    expect(el.querySelectorAll('#draft-topic-names option').length).toBe(2);
  });

  it('keeps edits until saved and sends the whole plan', async () => {
    await change(
      topicCard(1).querySelector<HTMLInputElement>('#draft-slot-title-0-1')!,
      'Hotel Night',
    );
    button(topicCard(2), '↑').click();
    await settle();
    expect(titles()).toEqual(['1 Thể thao', '2 Du lịch', '3 Sức khoẻ']);
    expect(text(el.querySelector('.unsaved'))).toBe('Có thay đổi chưa lưu.');

    button(el, 'Lưu bản nháp').click();
    await settle();
    const req = http.expectOne((r) => r.url === '/admin/curriculum/draft' && r.method === 'PUT');
    expect(req.request.params.get('level')).toBe('A2');
    const plan = req.request.body as CurriculumPlan;
    expect(plan.topics.map((t) => t.name)).toEqual(['Thể thao', 'Du lịch', 'Sức khoẻ']);
    expect(plan.topics[1].topicId).toBe('travel');
    expect(plan.topics[1].slots[1]).toEqual({
      title: 'Hotel Night',
      idea: 'Idea At the Hotel',
      grammar: ['a2-past-simple-regular'],
    });
    req.flush(draft());
    await settle();
    expect(emitted.length).toBe(1);
    expect(el.querySelector('.unsaved')).toBeNull();
    expect(text(el.querySelector('.note'))).toBe('Đã lưu bản nháp.');
  });

  it('renames a topic away from the catalogue and picks a lesson grammar point', async () => {
    await change(topicCard(1).querySelector<HTMLInputElement>('#draft-topic-name-0')!, 'Đi chơi');
    expect(text(topicCard(1).querySelector('.field-hint'))).toBe('Chủ đề mới: được tạo khi duyệt.');
    await change(topicCard(1).querySelector<HTMLSelectElement>('select')!, 'a2-will');
    button(el, 'Lưu bản nháp').click();
    await settle();
    const put = http.expectOne((r) => r.method === 'PUT');
    const plan = put.request.body as CurriculumPlan;
    expect(plan.topics[0]).toEqual(expect.objectContaining({ name: 'Đi chơi', topicId: '' }));
    expect(plan.topics[0].slots[0].grammar).toEqual(['a2-will']);
    put.flush(draft());
  });

  it('warns when a lesson would no longer fill a slot', async () => {
    const b = draft();
    b.topics[0].slots[0] = {
      ...b.topics[0].slots[0],
      lesson: { id: 'l1', title: 'My Trip', draft: false, annotationStatus: 'done' },
    };
    await show(b);
    expect(topicCard(1).querySelector('.unmatched')).toBeNull();
    await change(
      topicCard(1).querySelector<HTMLInputElement>('#draft-slot-title-0-0')!,
      'My  trip ',
    );
    expect(topicCard(1).querySelector('.unmatched')).toBeNull();
    await change(
      topicCard(1).querySelector<HTMLInputElement>('#draft-slot-title-0-0')!,
      'A Long Trip',
    );
    expect(text(topicCard(1).querySelector('.unmatched'))).toBe(
      'Bài đã có sẽ không khớp ô nào nữa (thành “bài khác” của chủ đề): My Trip.',
    );
  });

  it('adds and removes lessons and topics', async () => {
    button(topicCard(2), 'Thêm bài').click();
    await settle();
    expect(topicCard(2).querySelectorAll('.slot').length).toBe(1);
    button(topicCard(1), 'Bỏ').click();
    await settle();
    expect(titles()).toEqual(['1 Thể thao', '2 Sức khoẻ']);
    button(el, 'Thêm chủ đề').click();
    await settle();
    expect(titles()[2]).toBe('3 Chủ đề chưa có tên');
  });

  it('has the AI write the topics without lessons, one request each, saving edits first', async () => {
    await change(topicCard(1).querySelector<HTMLTextAreaElement>('textarea')!, 'đi du lịch.');
    button(el, 'Soạn bài cho các chủ đề chưa có bài').click();
    await settle();
    http.expectOne((r) => r.method === 'PUT').flush(draft());
    await settle();
    const first = http.expectOne((r) => r.url === '/admin/curriculum/draft/topics/2/write');
    expect(text(el.querySelector('.note'))).toContain('AI đang soạn bài cho chủ đề 2 (1/2)');
    expect(button(el, 'Duyệt giáo trình').disabled).toBe(true);
    const after2 = draft();
    after2.topics[1] = topic(2, 'Thể thao', [slot(1, 'Match Day')]);
    first.flush(after2);
    await settle();
    http.expectOne((r) => r.url === '/admin/curriculum/draft/topics/3/write').flush(draft());
    await settle();
    expect(text(el.querySelector('.note'))).toBe('AI đã soạn bài cho 2 chủ đề.');
  });

  it('adds the extra lessons a topic lacks, keeping its lessons', async () => {
    expect(button(topicCard(2), 'Soạn thêm bài thêm')).toBeUndefined();
    button(topicCard(1), 'Soạn thêm bài thêm (4)').click();
    await settle();
    const req = http.expectOne((r) => r.url === '/admin/curriculum/draft/topics/1/write');
    expect(req.request.params.get('mode')).toBe('extra');
    req.flush(draft());
    await settle();
    expect(text(el.querySelector('.note'))).toBe('AI đã soạn thêm bài thêm cho chủ đề.');
  });

  it('stops writing at the first error with the server message', async () => {
    button(el, 'Soạn bài cho các chủ đề chưa có bài').click();
    await settle();
    http
      .expectOne((r) => r.url === '/admin/curriculum/draft/topics/2/write')
      .flush(
        { error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' },
        { status: 429, statusText: 'Too Many Requests' },
      );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe(
      'Dừng ở chủ đề 2: Đã hết lượt AI, vui lòng thử lại sau.',
    );
    http.expectNone((r) => r.url.endsWith('/topics/3/write'));
  });

  it('approves after confirming, saving edits first', async () => {
    await change(topicCard(1).querySelector<HTMLTextAreaElement>('textarea')!, 'đi du lịch.');
    button(el, 'Duyệt giáo trình').click();
    await settle();
    expect(text(dialog()!.querySelector('h2'))).toBe('Duyệt giáo trình A2?');
    await confirm();
    http.expectOne((r) => r.method === 'PUT').flush(draft());
    await settle();
    http
      .expectOne((r) => r.url === '/admin/curriculum/approve' && r.method === 'POST')
      .flush({ ...draft(), status: 'approved', hasApproved: true });
    await settle();
    expect(emitted.at(-1)?.status).toBe('approved');
  });

  it('shows why approving was refused', async () => {
    button(el, 'Duyệt giáo trình').click();
    await settle();
    await confirm();
    http
      .expectOne((r) => r.url === '/admin/curriculum/approve')
      .flush(
        {
          error: 'invalid_curriculum',
          message: 'chủ đề “Thể thao” cần ít nhất 3 bài chính và 2 bài thêm',
        },
        { status: 422, statusText: 'Unprocessable Entity' },
      );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe(
      'chủ đề “Thể thao” cần ít nhất 3 bài chính và 2 bài thêm',
    );
  });

  it('discards the draft after confirming', async () => {
    button(el, 'Bỏ bản nháp').click();
    await settle();
    expect(text(dialog()!.querySelector('.message'))).toBe(
      'Bản nháp do AI soạn và mọi thay đổi sẽ mất.',
    );
    await confirm();
    http
      .expectOne((r) => r.url === '/admin/curriculum/draft' && r.method === 'DELETE')
      .flush({ ...draft(), status: 'none', topics: [] });
    await settle();
    expect(emitted.at(-1)?.status).toBe('none');
  });
});
