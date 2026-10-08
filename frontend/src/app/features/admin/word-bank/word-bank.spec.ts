import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { BankWord } from '../../../core/models/word-bank';
import { WordBank } from './word-bank';

describe('WordBank (F24)', () => {
  let fixture: ComponentFixture<WordBank>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const house: BankWord = {
    lemma: 'house',
    meaningVi: 'ngôi nhà',
    ipa: '/haʊs/',
    imageUrl: '/api/admin/words/house/image?v=1',
    updatedAt: '2026-10-07T09:00:00Z',
    topics: [{ id: 't2', name: 'Nhà cửa' }],
  };
  const table: BankWord = {
    lemma: 'table',
    meaningVi: '',
    ipa: '',
    imageUrl: '',
    updatedAt: '2026-10-07T09:00:00Z',
    topics: [],
  };

  /** Waits ms (past the search delay when needed), then for Angular. */
  const settle = async (ms = 0) => {
    await new Promise((resolve) => setTimeout(resolve, ms));
    await fixture.whenStable();
  };
  const rows = () => Array.from(el.querySelectorAll('lu-bank-word-row'));
  const button = (root: ParentNode, label: string) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      b.textContent?.replace(/\s+/g, ' ').trim().startsWith(label),
    );
  const type = (root: ParentNode, selector: string, value: string) => {
    const input = root.querySelector<HTMLInputElement>(selector)!;
    input.value = value;
    input.dispatchEvent(new Event('input'));
  };
  const chooseFile = (row: Element, file: File) => {
    const input = row.querySelector<HTMLInputElement>('input[type="file"]')!;
    Object.defineProperty(input, 'files', { value: [file], configurable: true });
    input.dispatchEvent(new Event('change'));
  };
  const expectList = (query: string) =>
    http.expectOne((r) => r.url === '/api/admin/words' && r.params.toString() === query);

  const open = async (words: BankWord[] = [house, table], hasMore = false) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(WordBank);
    el = fixture.nativeElement as HTMLElement;
    http.expectOne('/api/admin/topics').flush({
      topics: [
        { id: 't2', name: 'Nhà cửa' },
        { id: 't1', name: 'Chào hỏi' },
      ],
    });
    await settle(350);
    expectList('page=1').flush({ words, total: words.length + (hasMore ? 1 : 0), hasMore });
    await settle();
  };

  afterEach(() => http.verify());

  it('lists the words with their picture, IPA and meaning', async () => {
    await open();
    expect(el.querySelector('.summary')!.textContent).toContain('2 từ');
    expect(rows().map((r) => r.querySelector('.lemma')!.textContent)).toEqual(['house', 'table']);
    expect(rows()[0].querySelector('img')!.getAttribute('src')).toBe(
      '/api/admin/words/house/image?v=1',
    );
    expect(rows()[0].textContent).toContain('/haʊs/');
    expect(rows()[1].textContent).toContain('Chưa có ảnh');
    expect(rows()[1].textContent).toContain('Chưa có phiên âm');
    expect(rows()[1].textContent).toContain('Chưa có nghĩa');
    expect(button(rows()[1], 'Xoá ảnh')).toBeUndefined();
  });

  it('searches after typing stops and filters what is missing', async () => {
    await open();
    type(el, '#bank-search', ' nhà ');
    await settle(350);
    expectList('page=1&q=nh%C3%A0').flush({ words: [house], total: 1, hasMore: false });
    await settle();
    expect(rows().length).toBe(1);

    const select = el.querySelector<HTMLSelectElement>('#bank-missing')!;
    select.value = 'image';
    select.dispatchEvent(new Event('change'));
    await settle(350);
    expectList('page=1&q=nh%C3%A0&missing=image').flush({ words: [], total: 0, hasMore: false });
    await settle();
    expect(el.textContent).toContain('Không có từ nào khớp.');
  });

  it('loads the next page', async () => {
    await open([house], true);
    button(el, 'Xem thêm')!.click();
    expectList('page=2').flush({ words: [table], total: 2, hasMore: false });
    await settle();
    expect(rows().length).toBe(2);
    expect(button(el, 'Xem thêm')).toBeUndefined();
  });

  it('adds a word and shows the server error under the form', async () => {
    await open();
    button(el, 'Thêm')!.click();
    await settle();
    expect(el.querySelector('#bank-add-error')!.textContent).toContain('Vui lòng nhập từ');

    type(el, '#bank-lemma', ' Garden ');
    button(el, 'Thêm')!.click();
    const req = http.expectOne('/api/admin/words');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ lemma: 'Garden', meaningVi: '', ipa: '', topicId: '' });
    req.flush(
      { ...table, lemma: 'garden', inBank: false, topic: null },
      { status: 201, statusText: 'Created' },
    );
    await settle(350);
    expectList('page=1').flush({ words: [house, table], total: 2, hasMore: false });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain('Đã thêm "garden".');

    type(el, '#bank-lemma', 'house');
    button(el, 'Thêm')!.click();
    http
      .expectOne('/api/admin/words')
      .flush(
        { error: 'validation_failed', fields: { lemma: 'Từ này đã có trong kho' } },
        { status: 400, statusText: 'Bad Request' },
      );
    await settle();
    expect(el.querySelector('#bank-add-error')!.textContent).toContain('Từ này đã có trong kho');
  });

  it('shows the topics of each word and filters by topic', async () => {
    await open();
    expect(rows()[0].querySelector('.topics')!.textContent).toContain('Nhà cửa');
    expect(rows()[1].textContent).toContain('Chưa thuộc chủ đề nào');
    const filter = el.querySelector<HTMLSelectElement>('#bank-topic-filter')!;
    // By name.
    expect(Array.from(filter.options).map((o) => o.textContent?.trim())).toEqual([
      'Tất cả chủ đề',
      'Chào hỏi',
      'Nhà cửa',
    ]);

    filter.value = 't2';
    filter.dispatchEvent(new Event('change'));
    await settle(350);
    expectList('page=1&topicId=t2').flush({ words: [], total: 0, hasMore: false });
    await settle();
    expect(el.textContent).toContain('Kho chưa có từ nào của chủ đề này.');
  });

  it('adds a word to a topic, or says the topic has it already', async () => {
    await open();
    const topic = el.querySelector<HTMLSelectElement>('#bank-topic')!;
    topic.value = 't2';
    topic.dispatchEvent(new Event('change'));
    type(el, '#bank-lemma', 'table');
    button(el, 'Thêm')!.click();
    const req = http.expectOne('/api/admin/words');
    expect(req.request.body).toEqual({ lemma: 'table', meaningVi: '', ipa: '', topicId: 't2' });
    req.flush({ ...table, inBank: true, topic: { id: 't2', name: 'Nhà cửa' } });
    await settle(350);
    expectList('page=1').flush({ words: [house, table], total: 2, hasMore: false });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain(
      '"table" đã có trong kho, đã thêm vào chủ đề "Nhà cửa".',
    );
    // The topic stays chosen for the next word.
    expect(topic.value).toBe('t2');

    type(el, '#bank-lemma', 'house');
    button(el, 'Thêm')!.click();
    http
      .expectOne('/api/admin/words')
      .flush(
        { error: 'validation_failed', fields: { lemma: 'Từ này đã có trong chủ đề "Nhà cửa"' } },
        { status: 400, statusText: 'Bad Request' },
      );
    await settle();
    expect(el.querySelector('#bank-add-error')!.textContent).toContain(
      'Từ này đã có trong chủ đề "Nhà cửa"',
    );
  });

  it('fills the meanings and IPA of a topic with one AI request', async () => {
    await open();
    expect(button(el, 'Điền nghĩa và phiên âm bằng AI')).toBeUndefined();
    const choose = (selector: string, value: string) => {
      const s = el.querySelector<HTMLSelectElement>(selector)!;
      s.value = value;
      s.dispatchEvent(new Event('change'));
    };
    choose('#bank-topic-filter', 't2');
    choose('#bank-missing', 'ipa');
    await settle(350);
    expectList('page=1&missing=ipa&topicId=t2').flush({ words: [table], total: 1, hasMore: false });
    await settle();
    expect(el.querySelector('.fill-bar')!.textContent).toContain('chủ đề "Nhà cửa"');

    button(el, 'Điền nghĩa và phiên âm bằng AI')!.click();
    const req = http.expectOne('/api/admin/words/fill-missing');
    expect(req.request.body).toEqual({ topicId: 't2' });
    req.flush({ asked: 3, meanings: 2, ipas: 1 });
    await settle(350);
    expectList('page=1&missing=ipa&topicId=t2').flush({ words: [table], total: 1, hasMore: false });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain(
      'AI đã xử lý 3 từ còn thiếu: điền nghĩa cho 2 từ, phiên âm cho 1 từ.',
    );

    button(el, 'Điền nghĩa và phiên âm bằng AI')!.click();
    http
      .expectOne('/api/admin/words/fill-missing')
      .flush({ message: 'Đã hết lượt AI, vui lòng thử lại sau.' }, { status: 429, statusText: 'Too Many Requests' });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain('Đã hết lượt AI');
  });

  it('imports the topic words', async () => {
    await open();
    button(el, 'Nhập từ các chủ đề')!.click();
    http.expectOne('/api/admin/words/import').flush({ added: 3 });
    await settle(350);
    expectList('page=1').flush({ words: [house, table], total: 5, hasMore: false });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain('Đã thêm 3 từ từ các chủ đề.');
  });

  it('edits the meaning and IPA of a word', async () => {
    await open();
    button(rows()[1], 'Sửa')!.click();
    await settle();
    type(rows()[1], '#bank-meaning-table', 'cái bàn');
    button(rows()[1], 'Lưu')!.click();
    const req = http.expectOne('/api/admin/words/table');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ meaningVi: 'cái bàn', ipa: '' });
    req.flush({ ...table, meaningVi: 'cái bàn' });
    await settle();
    expect(rows()[1].textContent).toContain('cái bàn');
    expect(rows()[1].querySelector('#bank-meaning-table')).toBeNull();
  });

  it('uploads, draws and removes pictures', async () => {
    await open();
    const file = new File(['png'], 'table.png', { type: 'image/png' });
    chooseFile(rows()[1], file);
    const up = http.expectOne('/api/admin/words/table/image');
    expect(up.request.method).toBe('PUT');
    expect(up.request.body).toBe(file);
    await settle();
    expect(rows()[1].getAttribute('aria-busy')).toBe('true');
    up.flush({ ...table, imageUrl: '/api/admin/words/table/image?v=2' });
    await settle();
    expect(rows()[1].querySelector('img')!.getAttribute('src')).toBe(
      '/api/admin/words/table/image?v=2',
    );

    button(rows()[0], 'Sinh ảnh AI')!.click();
    await settle();
    expect(button(rows()[0], 'Đang sinh ảnh…')).toBeDefined();
    const draw = http.expectOne('/api/admin/words/house/image/generate');
    expect(draw.request.method).toBe('POST');
    draw.flush(
      { error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' },
      { status: 429, statusText: 'Too Many Requests' },
    );
    await settle();
    expect(rows()[0].querySelector('.row-note')!.textContent).toContain('Đã hết lượt AI');

    button(rows()[0], 'Xoá ảnh')!.click();
    const del = http.expectOne('/api/admin/words/house/image');
    expect(del.request.method).toBe('DELETE');
    del.flush({ ...house, imageUrl: '' });
    await settle();
    expect(rows()[0].querySelector('img')).toBeNull();
  });

  it('rejects files that are not pictures without calling the server', async () => {
    await open();
    chooseFile(rows()[0], new File(['x'], 'a.txt', { type: 'text/plain' }));
    await settle();
    expect(rows()[0].querySelector('.row-note')!.textContent).toContain(
      'Chỉ nhận ảnh JPEG, PNG hoặc GIF.',
    );
  });

  it('deletes a word after confirming', async () => {
    await open();
    button(rows()[1], 'Xoá từ')!.click();
    await settle();
    const dialog = el.querySelector('lu-confirm-dialog')!;
    expect(dialog.textContent).toContain('Từ "table" sẽ bị xoá khỏi kho');
    button(dialog, 'Xoá')!.click();
    const req = http.expectOne('/api/admin/words/table');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    expect(rows().map((r) => r.querySelector('.lemma')!.textContent)).toEqual(['house']);
    expect(el.querySelector('.summary')!.textContent).toContain('1 từ');
  });
});
