import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Topic, TopicWord } from '../../../core/models/topic';
import { parseWords, TopicWords } from './topic-words';

const topic = (id: string, name: string): Topic => ({
  id, name, level: 'A1', description: '', lessonCount: 2, roadmapCount: 2, remaining: 2, warning: true, createdAt: '',
  wordCount: 3, usedWordCount: 1,
});

const word = (text: string, lessonCount = 0): TopicWord => ({ text, used: lessonCount > 0, lessonCount });

const words = [word('Family', 3), word('Parents'), word('take a shower')];
const url = '/api/admin/topics/t1/words';

describe('parseWords', () => {
  it('splits by lines and commas, trims and drops empty entries', () => {
    expect(parseWords(' cousin, nephew\n\n  take   a shower \n,niece,')).toEqual(['cousin', 'nephew', 'take a shower', 'niece']);
    expect(parseWords('  \n , ')).toEqual([]);
  });
});

describe('TopicWords', () => {
  let fixture: ComponentFixture<TopicWords>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label)!;
  const byLabel = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  const rows = () =>
    Array.from(el.querySelectorAll('.word')).map((r) => `${text(r.querySelector('.word-text'))} | ${text(r.querySelector('.word-status'))}`);
  const rowOf = (w: string) => Array.from(el.querySelectorAll('.word')).find((r) => text(r.querySelector('.word-text')) === w)!;
  const box = () => el.querySelector<HTMLTextAreaElement>('#topic-words-add')!;
  const typeWords = async (value: string) => {
    box().value = value;
    box().dispatchEvent(new Event('input'));
    await settle();
  };

  const setup = async (list: TopicWord[] = words) => {
    await TestBed.configureTestingModule({
      imports: [TopicWords],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: 't1' }) } } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TopicWords);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/admin/topics').flush({ topics: [topic('t1', 'Gia đình'), topic('t2', 'Mua sắm')] });
    http.expectOne(url).flush({ words: list });
    await settle();
  };

  afterEach(() => http.verify());

  it('shows the topic, the coverage summary and a text label for each word', async () => {
    await setup();
    expect(text(el.querySelector('h1'))).toBe('Từ vựng · A1 · Gia đình');
    expect(el.querySelector('a[href="/admin/topics"]')).toBeTruthy();
    expect(text(el.querySelector('.summary'))).toBe('Đã dùng 1/3 từ');
    expect(rows()).toEqual(['Family | Đã dùng · 3 bài', 'Parents | Chưa dùng', 'take a shower | Chưa dùng']);
    expect(byLabel('Xoá Parents')).toBeTruthy();
  });

  it('shows an empty state', async () => {
    await setup([]);
    expect(text(el.querySelector('.summary'))).toBe('Đã dùng 0/0 từ');
    expect(el.textContent).toContain('Chủ đề chưa có từ vựng');
  });

  it('shows not found for an unknown topic', async () => {
    await TestBed.configureTestingModule({
      imports: [TopicWords],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: 't1' }) } } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TopicWords);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/admin/topics').flush({ topics: [] });
    http.expectOne(url).flush({ error: 'not_found' }, { status: 404, statusText: 'Not Found' });
    await settle();
    expect(el.textContent).toContain('Không tìm thấy chủ đề');
  });

  it('adds many words separated by commas and lines, skipping ones already there', async () => {
    await setup();
    await typeWords('cousin, nephew\nfamily\n\n niece ,cousin');
    button('Thêm vào danh sách').click();
    await settle();
    expect(rows().slice(3)).toEqual(['cousin | Mới', 'nephew | Mới', 'niece | Mới']);
    expect(text(el.querySelector('[role="status"]'))).toBe(
      'Đã thêm 3 từ mới, bấm Lưu để lưu. Bỏ qua 2 từ đã có: family, cousin.',
    );
    expect(box().value).toBe('');
  });

  it('adds with Ctrl+Enter from the keyboard', async () => {
    await setup();
    await typeWords('cousin');
    box().dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', ctrlKey: true, cancelable: true }));
    await settle();
    expect(rows()).toContain('cousin | Mới');
  });

  it('removes and adds words, then saves the kept words in order followed by the new ones', async () => {
    await setup();
    byLabel('Xoá Parents').click();
    await settle();
    expect(rows()[1]).toBe('Parents | Sẽ xoá');
    // The same button now restores the word, so keyboard focus is not lost.
    expect(byLabel('Giữ lại Parents')).toBeTruthy();

    await typeWords('cousin\nnephew');
    button('Thêm vào danh sách').click();
    await settle();
    byLabel('Xoá nephew').click();
    await settle();
    expect(document.activeElement).toBe(box());
    await typeWords('uncle');

    button('Lưu').click();
    await settle();
    const req = http.expectOne(url);
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ words: ['Family', 'take a shower', 'cousin', 'uncle'] });
    expect(button('Đang lưu…').disabled).toBe(true);
    req.flush({ words: [word('Family', 3), word('take a shower'), word('cousin', 1), word('uncle')] });
    await settle();
    expect(rows()).toEqual([
      'Family | Đã dùng · 3 bài',
      'take a shower | Chưa dùng',
      'cousin | Đã dùng · 1 bài',
      'uncle | Chưa dùng',
    ]);
    expect(text(el.querySelector('.summary'))).toBe('Đã dùng 2/4 từ');
    expect(text(el.querySelector('[role="status"]'))).toBe('Đã lưu');
    expect(box().value).toBe('');
  });

  it('shows server errors under the right word, including new ones, and the list error on top', async () => {
    await setup();
    byLabel('Xoá Family').click();
    await typeWords('Parents2, nephew');
    button('Lưu').click();
    await settle();
    const req = http.expectOne(url);
    expect(req.request.body).toEqual({ words: ['Parents', 'take a shower', 'Parents2', 'nephew'] });
    req.flush(
      {
        error: 'validation_failed',
        message: 'Dữ liệu không hợp lệ',
        fields: { 'words.2': "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /", 'words.1': 'Tối đa 40 ký tự', words: 'Tối đa 100 từ' },
      },
      { status: 400, statusText: 'Bad Request' },
    );
    await settle();
    expect(text(rowOf('Parents2').querySelector('.field-error'))).toBe("Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /");
    expect(text(rowOf('take a shower').querySelector('.field-error'))).toBe('Tối đa 40 ký tự');
    expect(rowOf('Parents').querySelector('.field-error')).toBeNull();
    expect(byLabel('Xoá Parents2').getAttribute('aria-describedby')).toBe(rowOf('Parents2').querySelector('.field-error')!.id);
    expect(text(el.querySelector('[role="alert"]'))).toBe('Tối đa 100 từ');
    // Nothing was saved: the pending changes stay.
    expect(rows()[0]).toBe('Family | Sẽ xoá');
  });

  it('summarises word errors on top when there is no list error', async () => {
    await setup();
    await typeWords('family2');
    button('Lưu').click();
    await settle();
    http.expectOne(url).flush(
      { error: 'validation_failed', message: 'x', fields: { 'words.3': 'Từ bị trùng' } },
      { status: 400, statusText: 'Bad Request' },
    );
    await settle();
    expect(text(rowOf('family2').querySelector('.field-error'))).toBe('Từ bị trùng');
    expect(text(el.querySelector('[role="alert"]'))).toBe('Chưa lưu: 1 từ cần sửa (xem bên dưới).');
  });

  it('Huỷ restores the list as loaded', async () => {
    await setup();
    byLabel('Xoá Family').click();
    await typeWords('cousin');
    button('Thêm vào danh sách').click();
    await typeWords('half typed');
    await settle();
    button('Huỷ').click();
    await settle();
    expect(rows()).toEqual(['Family | Đã dùng · 3 bài', 'Parents | Chưa dùng', 'take a shower | Chưa dùng']);
    expect(box().value).toBe('');
  });
});
