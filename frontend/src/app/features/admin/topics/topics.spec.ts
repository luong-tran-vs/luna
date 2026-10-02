import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Topic } from '../../../core/models/topic';
import { Topics } from './topics';

function topic(id: string, name: string, level: Topic['level'], over: Partial<Topic> = {}): Topic {
  return {
    id, name, level, description: '', lessonCount: 0, roadmapCount: 0, remaining: 0, warning: true,
    createdAt: '2026-09-30T00:00:00Z', wordCount: 0, usedWordCount: 0, ...over,
  };
}

const topics = [
  topic('t1', 'Gia đình', 'A1', { description: 'Người thân', lessonCount: 4, roadmapCount: 2, remaining: 2 }),
  topic('t2', 'Mua sắm', 'A1'),
  topic('t3', 'Công việc', 'B1', { lessonCount: 3, roadmapCount: 3, remaining: 3, warning: false }),
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
  const type = async (selector: string, value: string) => {
    const f = el.querySelector<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(selector)!;
    f.value = value;
    f.dispatchEvent(new Event(f instanceof HTMLSelectElement ? 'change' : 'input'));
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

  it('groups topics by level with counts', async () => {
    await setup();
    expect(Array.from(el.querySelectorAll('.level-heading')).map((h) => text(h))).toEqual(['A1', 'B1']);
    expect(text(row('Gia đình').querySelector('.description'))).toBe('Người thân');
    expect(text(row('Gia đình').querySelector('.counts'))).toBe('4 bài · 2 trong lộ trình');
  });

  it('shows the vocabulary coverage and links to the word list (F18)', async () => {
    await setup([
      topic('t1', 'Gia đình', 'A1', { wordCount: 37, usedWordCount: 12 }),
      topic('t2', 'Mua sắm', 'A1'),
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

  it('validates and adds a topic', async () => {
    await setup();
    button('Thêm chủ đề')!.click();
    await settle();
    await submit();
    expect(text(el.querySelector('#topic-name-error'))).toBe('Vui lòng nhập tên chủ đề');
    expect(text(el.querySelector('#topic-level-error'))).toBe('Vui lòng chọn trình độ');

    await type('#topic-name', ' Du lịch ');
    await type('#topic-level', 'A2');
    await type('#topic-description', 'Đi chơi xa');
    await submit();
    const req = http.expectOne('/api/admin/topics');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ name: 'Du lịch', level: 'A2', description: 'Đi chơi xa' });
    req.flush({ topic: topic('t9', 'Du lịch', 'A2') }, { status: 201, statusText: 'Created' });
    await settle();
    expectList([...topics, topic('t9', 'Du lịch', 'A2')]);
    await settle();
    expect(el.querySelector('form')).toBeNull();
    expect(row('Du lịch')).toBeTruthy();
  });

  it('shows the duplicate name error from the server', async () => {
    await setup();
    button('Thêm chủ đề')!.click();
    await settle();
    await type('#topic-name', 'gia đình');
    await type('#topic-level', 'A1');
    await submit();
    http.expectOne('/api/admin/topics').flush(
      { error: 'validation_failed', message: 'x', fields: { name: 'Chủ đề này đã có ở trình độ A1' } },
      { status: 400, statusText: 'Bad Request' },
    );
    await settle();
    expect(text(el.querySelector('#topic-name-error'))).toBe('Chủ đề này đã có ở trình độ A1');
  });

  it('edits a topic in place', async () => {
    await setup();
    button('Sửa Gia đình', row('Gia đình'))!.click();
    await settle();
    expect(el.querySelector<HTMLInputElement>('#topic-name')!.value).toBe('Gia đình');
    expect(el.querySelector<HTMLSelectElement>('#topic-level')!.value).toBe('A1');
    await type('#topic-description', 'Bố mẹ, anh chị em');
    await submit();
    const req = http.expectOne('/api/admin/topics/t1');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ name: 'Gia đình', level: 'A1', description: 'Bố mẹ, anh chị em' });
    req.flush({ topic: { ...topics[0], description: 'Bố mẹ, anh chị em' } });
    await settle();
    expectList([{ ...topics[0], description: 'Bố mẹ, anh chị em' }, topics[1], topics[2]]);
    await settle();
    expect(text(row('Gia đình').querySelector('.description'))).toBe('Bố mẹ, anh chị em');
  });

  it('deletes after confirmation and explains when lessons remain', async () => {
    await setup();
    button('Xoá Gia đình', row('Gia đình'))!.click();
    await settle();
    const dialog = el.querySelector('[role="alertdialog"]')!;
    expect(text(dialog)).toContain('A1 · Gia đình');
    button('Xoá', dialog)!.click();
    await settle();
    http.expectOne('/api/admin/topics/t1').flush(
      { error: 'topic_in_use', message: 'Chủ đề còn 4 bài, hãy chuyển hoặc xoá bài trước', count: 4 },
      { status: 409, statusText: 'Conflict' },
    );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Chủ đề còn 4 bài, hãy chuyển hoặc xoá bài trước');

    button('Xoá Mua sắm', row('Mua sắm'))!.click();
    await settle();
    button('Xoá', el.querySelector('[role="alertdialog"]')!)!.click();
    await settle();
    http.expectOne('/api/admin/topics/t2').flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    expectList([topics[0], topics[2]]);
    await settle();
    expect(Array.from(el.querySelectorAll('.topic-name')).map((n) => text(n))).toEqual(['Gia đình', 'Công việc']);
  });
});
