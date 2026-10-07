import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Topic, TopicLevel } from '../../../core/models/topic';
import { Topics } from './topics';

const lv = (level: TopicLevel['level'], lessonCount: number, roadmapCount = lessonCount): TopicLevel => ({
  level, lessonCount, roadmapCount, remaining: roadmapCount, warning: roadmapCount < 3,
});

function topic(id: string, name: string, levels: TopicLevel[] = [], over: Partial<Topic> = {}): Topic {
  return {
    id, name, description: '', levels, lessonCount: levels.reduce((n, l) => n + l.lessonCount, 0),
    createdAt: '2026-09-30T00:00:00Z', wordCount: 0, usedWordCount: 0, ...over,
  };
}

const topics = [
  topic('t1', 'Gia đình', [lv('A1', 4, 2), lv('B1', 1)], { description: 'Người thân' }),
  topic('t2', 'Mua sắm', [lv('A1', 2)]),
  topic('t3', 'Công việc', [lv('B1', 3)]),
  topic('t4', 'Màu sắc'),
];

describe('Topics', () => {
  let fixture: ComponentFixture<Topics>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string, root: ParentNode = el) =>
    Array.from(root.querySelectorAll('button')).find((b) => text(b) === label);
  const row = (name: string) =>
    Array.from(el.querySelectorAll('.topic')).find((r) => text(r.querySelector('.topic-name')) === name)!;
  const names = () => Array.from(el.querySelectorAll('.topic-name')).map((n) => text(n));
  const picker = () => Array.from(el.querySelectorAll<HTMLButtonElement>('.level-btn'));
  const type = async (selector: string, value: string) => {
    const f = el.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!;
    f.value = value;
    f.dispatchEvent(new Event('input'));
    f.dispatchEvent(new Event('blur'));
    await fixture.whenStable();
  };
  const submit = async () => {
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
  };
  const expectList = (list: Topic[] = topics) => http.expectOne('/api/admin/topics').flush({ topics: list });

  const setup = async (list: Topic[] = topics) => {
    TestBed.configureTestingModule({
      imports: [Topics],
      providers: [provideRouter([]), provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Topics);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    expectList(list);
    await settle();
  };

  afterEach(() => http.verify());

  it('lists the topics with lessons at one level, the first level by default, or every topic', async () => {
    await setup();
    expect(picker().map((b) => text(b))).toEqual(['A1 2', 'B1 2', 'Tất cả 4']);
    expect(picker().map((b) => b.getAttribute('aria-pressed'))).toEqual(['true', 'false', 'false']);
    expect(text(el.querySelector('.level-heading'))).toBe('Có bài A1 · 2 chủ đề');
    expect(names()).toEqual(['Gia đình', 'Mua sắm']);

    picker()[1].click();
    await settle();
    expect(names()).toEqual(['Công việc', 'Gia đình']);

    picker()[2].click();
    await settle();
    expect(text(el.querySelector('.level-heading'))).toBe('Tất cả · 4 chủ đề');
    expect(names()).toEqual(['Công việc', 'Gia đình', 'Màu sắc', 'Mua sắm']);
  });

  it('counts the lessons of each level of a topic', async () => {
    await setup();
    expect(text(row('Gia đình').querySelector('.description'))).toBe('Người thân');
    const item = (i: Element) => `${text(i.querySelector('.tag'))} ${text(i.querySelector('.muted'))}`;
    expect(Array.from(row('Gia đình').querySelectorAll('.level-count-item')).map(item)).toEqual([
      'A1 4 bài · 2 trong lộ trình',
      'B1 1 bài · 1 trong lộ trình',
    ]);
    picker()[2].click();
    await settle();
    expect(text(row('Màu sắc').querySelector('.counts'))).toBe('Chưa có bài');
  });

  it('opens Sinh bài AI on the roadmap of the level being listed', async () => {
    await setup();
    picker()[1].click();
    await settle();
    const link = Array.from(row('Gia đình').querySelectorAll('a')).find((a) => text(a).startsWith('Sinh bài AI'))!;
    expect(link.getAttribute('href')).toBe('/admin/roadmap?topicId=t1&level=B1&generate=1');
  });

  it('shows the vocabulary coverage and links to the word list (F18)', async () => {
    await setup([
      topic('t1', 'Gia đình', [lv('A1', 1)], { wordCount: 37, usedWordCount: 12 }),
      topic('t2', 'Mua sắm', [lv('A1', 1)]),
    ]);
    expect(text(row('Gia đình').querySelector('.word-counts'))).toBe('Từ vựng: đã dùng 12/37');
    expect(text(row('Mua sắm').querySelector('.word-counts'))).toBe('Chưa có từ vựng');
    const link = row('Gia đình').querySelector<HTMLAnchorElement>('a')!;
    expect(text(link)).toBe('Từ vựng Gia đình');
    expect(link.getAttribute('href')).toBe('/admin/topics/t1/words');
  });

  it('shows an empty state', async () => {
    await setup([]);
    expect(el.textContent).toContain('Chưa có chủ đề nào');
  });

  it('validates and adds a topic without a level, then lists every topic', async () => {
    await setup();
    button('Thêm chủ đề')!.click();
    await settle();
    expect(el.querySelector('#topic-level')).toBeNull();
    await submit();
    expect(text(el.querySelector('#topic-name-error'))).toBe('Vui lòng nhập tên chủ đề');

    await type('#topic-name', ' Du lịch ');
    await type('#topic-description', 'Đi chơi xa');
    await submit();
    const req = http.expectOne('/api/admin/topics');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ name: 'Du lịch', description: 'Đi chơi xa' });
    req.flush({ topic: topic('t9', 'Du lịch') }, { status: 201, statusText: 'Created' });
    await settle();
    expectList([...topics, topic('t9', 'Du lịch')]);
    await settle();
    expect(el.querySelector('form')).toBeNull();
    // A new topic has no lesson yet: the list switches to "Tất cả".
    expect(el.querySelector('.level-btn[aria-pressed="true"]')?.textContent).toContain('Tất cả');
    expect(row('Du lịch')).toBeTruthy();
  });

  it('shows the duplicate name error from the server', async () => {
    await setup();
    button('Thêm chủ đề')!.click();
    await settle();
    await type('#topic-name', 'gia đình');
    await submit();
    http.expectOne('/api/admin/topics').flush(
      { error: 'validation_failed', message: 'x', fields: { name: 'Chủ đề này đã có' } },
      { status: 400, statusText: 'Bad Request' },
    );
    await settle();
    expect(text(el.querySelector('#topic-name-error'))).toBe('Chủ đề này đã có');
  });

  it('edits a topic in place', async () => {
    await setup();
    button('Sửa Gia đình', row('Gia đình'))!.click();
    await settle();
    expect(el.querySelector<HTMLInputElement>('#topic-name')!.value).toBe('Gia đình');
    await type('#topic-description', 'Bố mẹ, anh chị em');
    await submit();
    const req = http.expectOne('/api/admin/topics/t1');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ name: 'Gia đình', description: 'Bố mẹ, anh chị em' });
    req.flush({ topic: { ...topics[0], description: 'Bố mẹ, anh chị em' } });
    await settle();
    expectList([{ ...topics[0], description: 'Bố mẹ, anh chị em' }, ...topics.slice(1)]);
    await settle();
    expect(text(row('Gia đình').querySelector('.description'))).toBe('Bố mẹ, anh chị em');
  });

  it('deletes after confirmation and explains when lessons remain', async () => {
    await setup();
    button('Xoá Gia đình', row('Gia đình'))!.click();
    await settle();
    const dialog = el.querySelector('[role="alertdialog"]')!;
    expect(text(dialog)).toContain('“Gia đình”');
    button('Xoá', dialog)!.click();
    await settle();
    http.expectOne('/api/admin/topics/t1').flush(
      { error: 'topic_in_use', message: 'Chủ đề còn 5 bài, hãy chuyển hoặc xoá bài trước', count: 5 },
      { status: 409, statusText: 'Conflict' },
    );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Chủ đề còn 5 bài, hãy chuyển hoặc xoá bài trước');

    picker()[2].click();
    await settle();
    button('Xoá Màu sắc', row('Màu sắc'))!.click();
    await settle();
    button('Xoá', el.querySelector('[role="alertdialog"]')!)!.click();
    await settle();
    http.expectOne('/api/admin/topics/t4').flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    expectList(topics.slice(0, 3));
    await settle();
    expect(names()).toEqual(['Công việc', 'Gia đình', 'Mua sắm']);
  });
});
