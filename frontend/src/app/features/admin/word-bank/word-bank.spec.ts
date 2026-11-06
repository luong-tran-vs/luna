import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { serverUrl } from '../../../core/api-url';
import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { BankImageBatch, BankWord } from '../../../core/models/word-bank';
import { BANK_BATCH_POLL_MS } from './image-batch.service';
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

  const noBatch: BankImageBatch = {
    running: false, stopping: false, stopped: false, engine: '', ticked: false, topics: [], total: 0, done: 0, drawing: [], saved: [], failures: [],
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
  /** The searchable topic boxes: their choices, or pick one. */
  const topicChoices = async (selector: string) => {
    const box = el.querySelector<HTMLInputElement>(selector)!;
    box.click();
    await fixture.whenStable();
    const labels = Array.from(el.querySelectorAll(`${selector}-list [role="option"] .label`)).map((o) => o.textContent?.trim());
    box.dispatchEvent(new Event('blur'));
    await fixture.whenStable();
    return labels;
  };
  const pickTopic = async (selector: string, value: string) => {
    const box = el.querySelector<HTMLInputElement>(selector)!;
    box.click();
    await fixture.whenStable();
    el.querySelector<HTMLElement>(`${selector}-list [data-value="${value}"]`)!.click();
    box.dispatchEvent(new Event('blur'));
    await fixture.whenStable();
  };
  const expectList = (query: string) =>
    http.expectOne((r) => r.url === '/admin/words' && r.params.toString() === query);

  const open = async (words: BankWord[] = [house, table], hasMore = false, codexImages = false, batch: BankImageBatch = noBatch) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: BANK_BATCH_POLL_MS, useValue: 10 },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(WordBank);
    el = fixture.nativeElement as HTMLElement;
    http.expectOne('/admin/topics').flush({
      topics: [
        { id: 't2', name: 'Nhà cửa' },
        { id: 't1', name: 'Chào hỏi' },
      ],
    });
    // "Nhà cửa" lacks all three, "Chào hỏi" nothing.
    http.expectOne('/admin/words/missing-by-topic').flush({ topics: [{ topicId: 't2', image: 3, ipa: 1, meaning: 2 }] });
    http.expectOne('/admin/words/image-batch').flush(batch);
    await settle(350);
    expectList('page=1').flush({ words, total: words.length + (hasMore ? 1 : 0), hasMore, codexImages });
    await settle();
  };

  afterEach(() => {
    // Every change of the bank counts again what the topics lack.
    http.match('/admin/words/missing-by-topic').forEach((r) => r.flush({ topics: [] }));
    http.verify();
  });

  it('shows what each topic lacks in the colors of the legend, which also filters', async () => {
    await open();
    const box = el.querySelector<HTMLInputElement>('#bank-topic-filter')!;
    box.click();
    await settle();
    const option = (id: string) => el.querySelector(`#bank-topic-filter-list [data-value="${id}"]`)!;
    const badges = Array.from(option('t2').querySelectorAll<HTMLElement>('.badge'));
    expect(badges.map((b) => b.title)).toEqual(['thiếu ảnh', 'thiếu phiên âm', 'thiếu nghĩa']);
    expect(badges.map((b) => b.textContent?.replace(/\s+/g, ' ').trim())).toEqual([
      ', 3 thiếu ảnh', ', 1 thiếu phiên âm', ', 2 thiếu nghĩa',
    ]);
    expect(option('t1').querySelector('.badge')).toBeNull();
    box.dispatchEvent(new Event('blur'));

    const chips = Array.from(el.querySelectorAll<HTMLButtonElement>('.legend-chip'));
    expect(chips.map((c) => c.textContent?.trim())).toEqual(['Thiếu ảnh', 'Thiếu phiên âm', 'Thiếu nghĩa']);
    chips[1].click();
    await settle(350);
    expect(chips[1].getAttribute('aria-pressed')).toBe('true');
    expect(el.querySelector<HTMLSelectElement>('#bank-missing')!.value).toBe('ipa');
    expectList('page=1&missing=ipa').flush({ words: [], total: 0, hasMore: false });
    await settle();
  });

  it('lists the words with their picture, IPA and meaning', async () => {
    await open();
    expect(el.querySelector('.summary')!.textContent).toContain('2 từ');
    expect(rows().map((r) => r.querySelector('.lemma')!.textContent)).toEqual(['house', 'table']);
    expect(rows()[0].querySelector('img')!.getAttribute('src')).toBe(
      serverUrl('/api/admin/words/house/image?v=1'),
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
    const req = http.expectOne('/admin/words');
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
      .expectOne('/admin/words')
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
    // By name.
    expect(await topicChoices('#bank-topic-filter')).toEqual(['Tất cả chủ đề', 'Chào hỏi', 'Nhà cửa']);

    await pickTopic('#bank-topic-filter', 't2');
    await settle(350);
    expectList('page=1&topicId=t2').flush({ words: [], total: 0, hasMore: false });
    await settle();
    expect(el.textContent).toContain('Kho chưa có từ nào của chủ đề này.');
  });

  it('adds a word to a topic, or says the topic has it already', async () => {
    await open();
    const topic = el.querySelector<HTMLInputElement>('#bank-topic')!;
    await pickTopic('#bank-topic', 't2');
    type(el, '#bank-lemma', 'table');
    button(el, 'Thêm')!.click();
    const req = http.expectOne('/admin/words');
    expect(req.request.body).toEqual({ lemma: 'table', meaningVi: '', ipa: '', topicId: 't2' });
    req.flush({ ...table, inBank: true, topic: { id: 't2', name: 'Nhà cửa' } });
    await settle(350);
    expectList('page=1').flush({ words: [house, table], total: 2, hasMore: false });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain(
      '"table" đã có trong kho, đã thêm vào chủ đề "Nhà cửa".',
    );
    // The topic stays chosen for the next word.
    expect(topic.value).toBe('Nhà cửa');

    type(el, '#bank-lemma', 'house');
    button(el, 'Thêm')!.click();
    http
      .expectOne('/admin/words')
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
    await pickTopic('#bank-topic-filter', 't2');
    choose('#bank-missing', 'ipa');
    await settle(350);
    expectList('page=1&missing=ipa&topicId=t2').flush({ words: [table], total: 1, hasMore: false });
    await settle();
    expect(el.querySelector('.fill-bar')!.textContent).toContain('chủ đề "Nhà cửa"');

    button(el, 'Điền nghĩa và phiên âm bằng AI')!.click();
    const req = http.expectOne('/admin/words/fill-missing');
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
      .expectOne('/admin/words/fill-missing')
      .flush({ message: 'Đã hết lượt AI, vui lòng thử lại sau.' }, { status: 429, statusText: 'Too Many Requests' });
    await settle();
    expect(el.querySelector('.note')!.textContent).toContain('Đã hết lượt AI');
  });

  it('offers the standard IPA for a stored one that needs a look, one by one or all at once', async () => {
    const odd: BankWord = { ...house, ipa: '/ˈhɑʊs/', ipaSuggestion: '/haʊs/' };
    await open([odd, table]);
    expect(rows()[0].textContent?.replace(/\s+/g, ' ')).toContain('Gợi ý chuẩn: /haʊs/');
    expect(rows()[1].querySelector('.ipa-suggestion')).toBeNull();
    // A listed word has a suggestion: the button for all of them shows without the filter.
    expect(button(el, 'Nhận tất cả gợi ý')).toBeDefined();
    button(rows()[0], 'Nhận')!.click();
    const req = http.expectOne('/admin/words/house');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ ipa: '/haʊs/' });
    req.flush(house);
    await settle();
    expect(rows()[0].querySelector('.ipa-suggestion')).toBeNull();
    expect(button(el, 'Nhận tất cả gợi ý')).toBeUndefined();

    // The filter lists only those, with a button for all of them.
    const select = el.querySelector<HTMLSelectElement>('#bank-missing')!;
    select.value = 'ipaCheck';
    select.dispatchEvent(new Event('change'));
    await settle(350);
    expectList('page=1&missing=ipaCheck').flush({ words: [odd], total: 1, hasMore: false });
    await settle();
    button(el, 'Nhận tất cả gợi ý')!.click();
    http.expectOne((r) => r.url === '/admin/words/ipa-suggestions/accept' && r.method === 'POST').flush({ updated: 1 });
    await settle();
    expect(el.querySelector('.note')?.textContent).toContain('Đã thay phiên âm chuẩn cho 1 từ.');
    await settle(350);
    expectList('page=1&missing=ipaCheck').flush({ words: [], total: 0, hasMore: false });
    await settle();
    expect(button(el, 'Nhận tất cả gợi ý')).toBeUndefined();
  });

  it('takes every suggested IPA at once from the full list', async () => {
    const odd: BankWord = { ...house, ipa: '/ˈhɑʊs/', ipaSuggestion: '/haʊs/' };
    await open([odd, table]);
    button(el, 'Xem các từ cần xem')!.click();
    await settle(350);
    expectList('page=1&missing=ipaCheck').flush({ words: [odd], total: 1, hasMore: false });
    await settle();
    expect(button(el, 'Xem các từ cần xem')).toBeUndefined();
    const select = el.querySelector<HTMLSelectElement>('#bank-missing')!;
    select.value = '';
    select.dispatchEvent(new Event('change'));
    await settle(350);
    expectList('page=1').flush({ words: [odd, table], total: 2, hasMore: false });
    await settle();

    button(el, 'Nhận tất cả gợi ý')!.click();
    http.expectOne((r) => r.url === '/admin/words/ipa-suggestions/accept' && r.method === 'POST').flush({ updated: 1 });
    await settle();
    expect(el.querySelector('.note')?.textContent).toContain('Đã thay phiên âm chuẩn cho 1 từ.');
    await settle(350);
    expectList('page=1').flush({ words: [house, table], total: 2, hasMore: false });
    await settle();
    expect(button(el, 'Nhận tất cả gợi ý')).toBeUndefined();
  });

  it('imports the topic words', async () => {
    await open();
    button(el, 'Nhập từ các chủ đề')!.click();
    http.expectOne('/admin/words/import').flush({ added: 3 });
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
    const req = http.expectOne('/admin/words/table');
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
    const up = http.expectOne('/admin/words/table/image');
    expect(up.request.method).toBe('PUT');
    expect(up.request.body).toBe(file);
    await settle();
    expect(rows()[1].getAttribute('aria-busy')).toBe('true');
    up.flush({ ...table, imageUrl: '/api/admin/words/table/image?v=2' });
    await settle();
    expect(rows()[1].querySelector('img')!.getAttribute('src')).toBe(
      serverUrl('/api/admin/words/table/image?v=2'),
    );

    button(rows()[0], 'Sinh ảnh AI')!.click();
    await settle();
    expect(button(rows()[0], 'Đang sinh ảnh…')).toBeDefined();
    const draw = http.expectOne('/admin/words/house/image/generate');
    expect(draw.request.method).toBe('POST');
    draw.flush(
      { error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' },
      { status: 429, statusText: 'Too Many Requests' },
    );
    await settle();
    expect(rows()[0].querySelector('.row-note')!.textContent).toContain('Đã hết lượt AI');

    button(rows()[0], 'Xoá ảnh')!.click();
    const del = http.expectOne('/admin/words/house/image');
    expect(del.request.method).toBe('DELETE');
    del.flush({ ...house, imageUrl: '' });
    await settle();
    expect(rows()[0].querySelector('img')).toBeNull();
  });

  it('draws with the Codex CLI when the server offers it, remembering the choice', async () => {
    localStorage.removeItem('luna.admin.imageEngine');
    await open([house], false, true);
    const engine = el.querySelector<HTMLSelectElement>('#bank-engine')!;
    expect(engine.value).toBe('gemini');
    engine.value = 'codex';
    engine.dispatchEvent(new Event('change'));
    await settle();
    expect(localStorage.getItem('luna.admin.imageEngine')).toBe('codex');

    button(rows()[0], 'Sinh ảnh Codex')!.click();
    await settle();
    expect(button(rows()[0], 'Codex đang vẽ')).toBeDefined();
    const draw = http.expectOne('/admin/words/house/image/generate');
    expect(draw.request.body).toEqual({ style: '', engine: 'codex' });
    draw.flush({ ...house, imageUrl: '/api/admin/words/house/image?v=3' });
    await settle();
    expect(rows()[0].querySelector('.row-note')!.textContent).toContain('Codex CLI');
    localStorage.removeItem('luna.admin.imageEngine');
  });

  it('hides the engine choice and draws with Gemini when the server has no Codex', async () => {
    localStorage.setItem('luna.admin.imageEngine', 'codex');
    await open([house]);
    expect(el.querySelector('#bank-engine')).toBeNull();
    button(rows()[0], 'Sinh ảnh AI')!.click();
    await settle();
    const draw = http.expectOne('/admin/words/house/image/generate');
    expect(draw.request.body).toEqual({ style: '', engine: 'gemini' });
    draw.flush(house);
    await settle();
    localStorage.removeItem('luna.admin.imageEngine');
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
    const req = http.expectOne('/admin/words/table');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    expect(rows().map((r) => r.querySelector('.lemma')!.textContent)).toEqual(['house']);
    expect(el.querySelector('.summary')!.textContent).toContain('1 từ');
  });

  it('draws the ticked words in the background and follows the batch', async () => {
    await open();
    const bar = el.querySelector('.batch-bar')!;
    expect(button(bar, 'Sinh ảnh cho 0 từ đã chọn')!.disabled).toBe(true);
    const pick = (r: Element) => r.querySelector<HTMLInputElement>('.pick input')!;
    pick(rows()[1]).click();
    await settle();
    button(bar, 'Sinh ảnh cho 1 từ đã chọn')!.click();
    const start = http.expectOne((r) => r.method === 'POST' && r.url === '/admin/words/image-batch');
    expect(start.request.body).toEqual({ lemmas: ['table'], topicId: '', engine: 'gemini' });
    start.flush({ ...noBatch, running: true, ticked: true, engine: 'gemini', total: 1, drawing: ['table'], startedAt: '2026-11-04T10:00:00Z' });
    await settle();
    expect(bar.textContent).toContain('Đang sinh ảnh cho 1 từ đã chọn bằng Gemini: đã xử lý 0/1 từ, còn 1 từ');
    expect(rows()[1].querySelector('.thumb-empty')!.textContent).toContain('Đang sinh…');
    expect(pick(rows()[1]).checked).toBe(false);

    await settle(20);
    const done = { ...table, imageUrl: '/api/admin/words/table/image?v=2' };
    http.expectOne('/admin/words/image-batch').flush({
      ...noBatch, engine: 'gemini', total: 1, done: 1, saved: [done], startedAt: '2026-11-04T10:00:00Z', finishedAt: '2026-11-04T10:00:20Z',
    });
    await settle(20);
    expect(rows()[1].querySelector('img')!.getAttribute('src')).toBe(serverUrl(done.imageUrl));
    expect(bar.textContent).toContain('Lượt sinh ảnh đã xong cho mọi từ thiếu ảnh: sinh được 1/1 ảnh bằng Gemini');
    button(bar, 'Ẩn kết quả')!.click();
    await settle();
    expect(bar.textContent).not.toContain('Lượt sinh ảnh đã xong');
  });

  it('draws every word lacking a picture after asking, and shows why a batch stopped early', async () => {
    await open([house, table], false, true);
    el.querySelector<HTMLSelectElement>('#bank-engine')!.value = 'codex';
    el.querySelector<HTMLSelectElement>('#bank-engine')!.dispatchEvent(new Event('change'));
    await settle();
    button(el.querySelector('.batch-bar')!, 'Sinh ảnh mọi từ thiếu ảnh')!.click();
    await settle();
    expect(el.textContent).toContain('Mỗi ảnh Codex mất khoảng 1 phút');
    button(el, 'Bắt đầu')!.click();
    const start = http.expectOne((r) => r.method === 'POST' && r.url === '/admin/words/image-batch');
    expect(start.request.body).toEqual({ q: '', topicId: '', engine: 'codex' });
    start.flush({
      ...noBatch, engine: 'codex', total: 2, done: 1, startedAt: '2026-11-04T10:00:00Z', finishedAt: '2026-11-04T10:00:20Z',
      failures: [{ lemma: 'table', message: 'Codex CLI không có công cụ tạo ảnh' }], stopMessage: 'Codex CLI không có công cụ tạo ảnh',
    });
    await settle();
    expect(el.querySelector('.batch-bar [role="alert"]')!.textContent).toContain('dừng sớm: Codex CLI không có công cụ tạo ảnh');
    expect(el.querySelector('.batch-failures')!.textContent).toContain('table');
    localStorage.removeItem('luna.admin.imageEngine');
  });

  it('follows a batch that was already running when the page opened, and stops it', async () => {
    await open([house, table], false, false, { ...noBatch, running: true, engine: 'gemini', total: 5, done: 2, drawing: ['table'] });
    const bar = el.querySelector('.batch-bar')!;
    expect(bar.textContent).toContain('đã xử lý 2/5 từ, còn 3 từ');
    button(bar, 'Dừng')!.click();
    http
      .expectOne((r) => r.url === '/admin/words/image-batch/stop')
      .flush({ ...noBatch, running: true, stopping: true, engine: 'gemini', total: 5, done: 2, drawing: ['table'] });
    await settle();
    expect(button(bar, 'Đang dừng')!.disabled).toBe(true);
    http.match('/admin/words/image-batch').forEach((r) => r.flush({ ...noBatch, stopped: true, engine: 'gemini', total: 5, done: 3, startedAt: 'x' }));
    await settle(20);
    http.match('/admin/words/image-batch').forEach((r) => r.flush({ ...noBatch }));
    expect(bar.textContent).toContain('Đã dừng lượt sinh ảnh');
  });
});
