import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { serverUrl } from '../../../../core/api-url';
import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import { IMAGE_STYLE_PLACEHOLDER } from '../../../../core/models/generate';
import { SceneBatch } from '../../../../core/models/lesson';
import { SceneBatchService } from '../../curriculum/scene-batch.service';
import { LessonImages } from './lesson-images';

/** A scene picture batch; by default none ran since the server started. */
const sceneBatch = (extra: Partial<SceneBatch> = {}): SceneBatch => ({
  running: false,
  stopping: false,
  stopped: false,
  engine: '',
  key: '',
  label: '',
  total: 0,
  done: 0,
  drawing: null,
  saved: [],
  failures: [],
  skipped: [],
  codexImages: false,
  ...extra,
});

describe('LessonImages', () => {
  let fixture: ComponentFixture<LessonImages>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const url = '/admin/lessons/l1/images';
  const text = (selector: string) =>
    el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const submit = () => el.querySelector<HTMLButtonElement>('button[type="submit"]')!;

  const open = async (state: object, batch: SceneBatch = sceneBatch()) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LessonImages);
    fixture.componentRef.setInput('lessonId', 'l1');
    fixture.componentRef.setInput('lessonTitle', 'Park');
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne(url).flush({ words: [], ...state });
    http.expectOne('/admin/lessons/scene-batch').flush(batch);
    await settle();
  };

  afterEach(() => http.verify());

  describe('F23: uploading a picture for each word', () => {
    const words = [
      { lemma: 'coffee', meaningVi: 'cà phê', imageUrl: '' },
      { lemma: 'give up', meaningVi: 'từ bỏ', imageUrl: '/api/admin/lessons/l1/images/give%20up' },
    ];
    const base = { style: '', sceneDescription: '', count: 1, words };
    const rows = () => Array.from(el.querySelectorAll('.word:not(.scene)'));
    const choose = async (row: Element, file: File) => {
      const input = row.querySelector<HTMLInputElement>('input[type="file"]')!;
      Object.defineProperty(input, 'files', { value: [file], configurable: true });
      input.dispatchEvent(new Event('change'));
      await fixture.whenStable();
    };

    it('lists the words with their picture and the actions', async () => {
      await open(base);
      expect(rows().map((r) => r.querySelector('.lemma')!.textContent)).toEqual([
        'coffee',
        'give up',
      ]);
      expect(rows()[0].querySelector('img')).toBeNull();
      expect(rows()[0].textContent).toContain('Chưa có ảnh');
      const src = rows()[1].querySelector('img')!.getAttribute('src')!;
      expect(src.startsWith(serverUrl('/api/admin/lessons/l1/images/give%20up?v='))).toBe(true);
      expect(src).toMatch(/\?v=\d+$/);
      expect(rows()[0].querySelector('.upload-btn')!.textContent).toContain('Tải ảnh lên');
      expect(rows()[1].querySelector('.upload-btn')!.textContent).toContain('Đổi ảnh');
      expect(rows()[0].querySelector('.remove-btn')).toBeNull();
      expect(rows()[1].querySelector('.remove-btn')!.getAttribute('aria-label')).toBe(
        'Xoá ảnh của give up',
      );
    });

    it('says when the lesson has no words yet', async () => {
      await open({ ...base, words: [], count: 0 });
      expect(el.textContent).toContain('Bài chưa chú thích xong');
      expect(el.querySelector('.word:not(.scene)')).toBeNull();
    });

    it('sends the chosen file as the body and shows the new picture', async () => {
      await open(base);
      const file = new File(['png'], 'cup.png', { type: 'image/png' });
      await choose(rows()[0], file);
      const req = http.expectOne('/admin/lessons/l1/images/coffee');
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toBe(file);
      expect(req.request.headers.get('Content-Type')).toBe('image/png');
      expect(rows()[0].getAttribute('aria-busy')).toBe('true');
      req.flush({
        ...base,
        count: 2,
        words: [{ ...words[0], imageUrl: '/api/admin/lessons/l1/images/coffee' }, words[1]],
      });
      await settle();
      expect(rows()[0].querySelector('img')).not.toBeNull();
      expect(text('.word [role="status"]')).toBe('Đã tải ảnh lên.');
    });

    it('refuses other file types and files over 5 MB before sending', async () => {
      await open(base);
      await choose(rows()[0], new File(['x'], 'a.webp', { type: 'image/webp' }));
      expect(text('.word [role="alert"]')).toBe('Chỉ nhận ảnh JPEG, PNG hoặc GIF.');
      const big = new File(['x'], 'big.jpg', { type: 'image/jpeg' });
      Object.defineProperty(big, 'size', { value: 5 * 1024 * 1024 + 1 });
      await choose(rows()[0], big);
      expect(text('.word [role="alert"]')).toBe('Ảnh tối đa 5 MB.');
    });

    it('shows the server message when the picture cannot be read', async () => {
      await open(base);
      await choose(rows()[0], new File(['x'], 'a.png', { type: 'image/png' }));
      http
        .expectOne('/admin/lessons/l1/images/coffee')
        .flush(
          { fields: { image: 'Không đọc được ảnh. Hãy dùng ảnh JPEG, PNG hoặc GIF.' } },
          { status: 400, statusText: 'Bad Request' },
        );
      await settle();
      expect(text('.word [role="alert"]')).toBe(
        'Không đọc được ảnh. Hãy dùng ảnh JPEG, PNG hoặc GIF.',
      );
    });

    describe('from a link', () => {
      const openLink = async (row: Element) => {
        row.querySelector<HTMLButtonElement>('.link-btn')!.click();
        await fixture.whenStable();
      };
      const submitLink = async (row: Element, value: string) => {
        const input = row.querySelector<HTMLInputElement>('.link-form input')!;
        input.value = value;
        row.querySelector<HTMLButtonElement>('.link-form button[type="submit"]')!.click();
        await fixture.whenStable();
      };

      it('opens one link field at a time', async () => {
        await open(base);
        expect(el.querySelector('.link-form')).toBeNull();
        await openLink(rows()[0]);
        expect(rows()[0].querySelector('.link-form')).not.toBeNull();
        expect(rows()[0].querySelector('.link-btn')!.getAttribute('aria-expanded')).toBe('true');
        await openLink(rows()[1]);
        expect(rows()[0].querySelector('.link-form')).toBeNull();
        expect(rows()[1].querySelector('.link-form')).not.toBeNull();
      });

      it('sends the link to the server, which fetches the picture', async () => {
        await open(base);
        await openLink(rows()[0]);
        await submitLink(rows()[0], '  https://example.com/cup.jpg ');
        const req = http.expectOne('/admin/lessons/l1/images/coffee/import');
        expect(req.request.method).toBe('POST');
        expect(req.request.body).toEqual({ url: 'https://example.com/cup.jpg' });
        req.flush({
          ...base,
          count: 2,
          words: [{ ...words[0], imageUrl: '/api/admin/lessons/l1/images/coffee' }, words[1]],
        });
        await settle();
        expect(rows()[0].querySelector('img')).not.toBeNull();
        expect(rows()[0].querySelector('.link-form')).toBeNull();
        expect(text('.word [role="status"]')).toBe('Đã lấy ảnh từ link.');
      });

      it('refuses what is not a link, and shows why the server could not use one', async () => {
        await open(base);
        await openLink(rows()[0]);
        await submitLink(rows()[0], 'cup.jpg');
        expect(text('.word [role="alert"]')).toBe(
          'Vui lòng dán link ảnh bắt đầu bằng http:// hoặc https://',
        );

        await submitLink(rows()[0], 'https://example.com/page');
        http.expectOne('/admin/lessons/l1/images/coffee/import').flush(
          {
            fields: {
              url: 'Link này không phải ảnh JPEG, PNG hoặc GIF. Hãy dùng link trỏ thẳng tới file ảnh.',
            },
          },
          { status: 400, statusText: 'Bad Request' },
        );
        await settle();
        expect(text('.word [role="alert"]')).toBe(
          'Link này không phải ảnh JPEG, PNG hoặc GIF. Hãy dùng link trỏ thẳng tới file ảnh.',
        );
        // The field stays open to try another link.
        expect(rows()[0].querySelector('.link-form')).not.toBeNull();
      });
    });

    it('removes a picture', async () => {
      await open(base);
      rows()[1].querySelector<HTMLButtonElement>('.remove-btn')!.click();
      await fixture.whenStable();
      const req = http.expectOne('/admin/lessons/l1/images/give%20up');
      expect(req.request.method).toBe('DELETE');
      req.flush({ ...base, count: 0, words: [words[0], { ...words[1], imageUrl: '' }] });
      await settle();
      expect(rows()[1].querySelector('img')).toBeNull();
      expect(text('.word:not(.scene) [role="status"]')).toBe('Đã xoá ảnh.');
    });
  });

  describe('the scene picture, drawn on request with the chosen engine (2026-11-06)', () => {
    const sceneRow = () => el.querySelector('.word.scene')!;
    const area = (id: string) => el.querySelector<HTMLTextAreaElement>(`#${id}`)!;
    const save = () =>
      Array.from(el.querySelectorAll<HTMLButtonElement>('form button[type="button"]')).find((b) =>
        b.textContent?.includes('Chỉ lưu mô tả'),
      )!;
    const radio = (value: string) =>
      el.querySelector<HTMLInputElement>(`input[name="images-engine"][value="${value}"]`)!;
    const openField = async (label: string) => {
      Array.from(el.querySelectorAll<HTMLButtonElement>('.chip-toggle'))
        .find((b) => b.textContent?.replace(/\s+/g, ' ').trim() === label)!
        .click();
      await fixture.whenStable();
    };
    const base = {
      sceneDescription: 'A family at home',
      style: '',
      count: 0,
      sceneImageUrl: '/api/admin/lessons/l1/images/_scene',
    };

    beforeEach(() => localStorage.removeItem('luna.admin.imageEngine'));
    afterEach(() => localStorage.removeItem('luna.admin.imageEngine'));

    it('shows the scene picture, its description and the default style placeholder', async () => {
      await open(base);
      expect(sceneRow().textContent).toContain('Ảnh cảnh của bài');
      const src = sceneRow().querySelector('img')!.getAttribute('src')!;
      expect(src.startsWith(serverUrl('/api/admin/lessons/l1/images/_scene?v='))).toBe(true);
      expect(sceneRow().querySelector('.remove-btn')!.getAttribute('aria-label')).toBe(
        'Xoá ảnh cảnh',
      );
      expect(area('images-scene').value).toBe('A family at home');
      // The style is empty: its field stays closed until asked for.
      expect(el.querySelector('#images-style')).toBeNull();
      await openField('+ Phong cách ảnh');
      expect(area('images-style').placeholder).toBe(IMAGE_STYLE_PLACEHOLDER);
      expect(text('button[type="submit"]')).toBe('Tạo lại ảnh cảnh');
      // No switches any more; Codex is listed but cannot be chosen when the server has none.
      expect(el.querySelector('button[role="switch"]')).toBeNull();
      const codex = radio('codex');
      expect(codex.disabled).toBe(true);
      expect(codex.closest('label')!.textContent).toContain('Máy chủ chưa bật');
      // Nothing is chosen at first (asked 2026-11-10).
      expect(radio('gemini').checked).toBe(false);
      expect(codex.checked).toBe(false);
    });

    const batchUrl = '/admin/lessons/scene-batch';
    const running = (engine: 'gemini' | 'codex' = 'gemini') =>
      sceneBatch({
        running: true,
        engine,
        key: 'lesson:l1',
        label: 'bài “Park”',
        total: 1,
        drawing: { id: 'l1', title: 'Park' },
      });
    /** Saves the form (PUT) then starts the background batch of this lesson (POST). */
    const expectStart = (saved: object) => {
      const put = http.expectOne((r) => r.url === url && r.method === 'PUT');
      put.flush({ ...base, words: [], ...saved });
      return put;
    };

    it('saves the form, then has the server draw the scene in the background', async () => {
      await open({ ...base, sceneDescription: '', sceneImageUrl: '' });
      expect(text('button[type="submit"]')).toBe('Tạo ảnh cảnh');
      expect(el.querySelector('#images-scene')).toBeNull();
      await openField('+ Mô tả cảnh');
      area('images-scene').value = '  Two friends in a park  ';
      area('images-scene').dispatchEvent(new Event('input'));
      radio('gemini').click();
      await fixture.whenStable();
      submit().click();
      await fixture.whenStable();
      const put = expectStart({ sceneDescription: 'Two friends in a park', sceneImageUrl: '' });
      expect(put.request.body).toEqual({ sceneDescription: 'Two friends in a park', style: '' });
      await settle();
      const req = http.expectOne((r) => r.url === batchUrl && r.method === 'POST');
      expect(req.request.body).toEqual({
        lessonIds: ['l1'],
        replace: true,
        engine: 'gemini',
        key: 'lesson:l1',
        label: 'bài “Park”',
      });
      req.flush(running());
      await settle();
      expect(text('button[type="submit"]')).toBe('Gemini đang vẽ ngầm…');
      expect(text('.bg-note')).toContain('Máy chủ đang tạo ảnh cảnh ngầm');

      // The batch ends (the next status read): the new picture is read again.
      TestBed.inject(SceneBatchService).set(
        sceneBatch({
          ...running(),
          running: false,
          drawing: null,
          done: 1,
          saved: [{ id: 'l1', title: 'Park' }],
        }),
      );
      await settle();
      http.expectOne(url).flush({ ...base, words: [] });
      await settle();
      expect(sceneRow().querySelector('img')).not.toBeNull();
      expect(text('p[role="status"]')).toBe('Đã tạo ảnh cảnh bằng Gemini.');
      expect(text('button[type="submit"]')).toBe('Tạo lại ảnh cảnh');
    });

    it('offers Codex when the server runs it, and remembers no choice', async () => {
      await open({ ...base, codexImages: true });
      expect(el.querySelector('.engine.selected')).toBeNull();
      radio('codex').click();
      await fixture.whenStable();
      expect(radio('codex').checked).toBe(true);
      expect(el.querySelector('.engine.selected')!.textContent).toContain('tài khoản ChatGPT');
      expect(localStorage.getItem('luna.admin.imageEngine')).toBeNull();
      submit().click();
      await fixture.whenStable();
      expectStart({ codexImages: true });
      await settle();
      const req = http.expectOne((r) => r.url === batchUrl && r.method === 'POST');
      expect(req.request.body.engine).toBe('codex');
      req.flush(running('codex'));
      await settle();
      expect(text('button[type="submit"]')).toBe('Codex đang vẽ ngầm… (khoảng 1 phút)');
    });

    it('preselects nothing, even a choice made earlier, and draws nothing without a choice', async () => {
      localStorage.setItem('luna.admin.imageEngine', 'codex');
      await open({ ...base, codexImages: true });
      expect(radio('gemini').checked).toBe(false);
      expect(radio('codex').checked).toBe(false);
      submit().click();
      await fixture.whenStable();
      expect(text('.form-alert')).toBe('Hãy chọn tạo ảnh bằng Gemini hay Codex CLI trước.');
      http.expectNone((r) => r.url === url && r.method === 'PUT');
      http.expectNone((r) => r.url === batchUrl && r.method === 'POST');
    });

    it('counts Codex as no choice when this server has none', async () => {
      await open({ ...base, codexImages: true });
      radio('codex').click();
      await fixture.whenStable();
      // The server answers again, now without Codex.
      save().click();
      await fixture.whenStable();
      http.expectOne((r) => r.url === url && r.method === 'PUT').flush({ ...base, words: [] });
      await settle();
      expect(radio('codex').disabled).toBe(true);
      expect(radio('codex').checked).toBe(false);
      submit().click();
      await fixture.whenStable();
      expect(text('.form-alert')).toBe('Hãy chọn tạo ảnh bằng Gemini hay Codex CLI trước.');
      http.expectNone((r) => r.url === url && r.method === 'PUT');
    });

    it('shows why the drawing failed once the batch ends', async () => {
      await open(base, running());
      expect(text('button[type="submit"]')).toBe('Gemini đang vẽ ngầm…');
      TestBed.inject(SceneBatchService).set(
        sceneBatch({
          ...running(),
          running: false,
          drawing: null,
          done: 1,
          failures: [{ id: 'l1', title: 'Park', message: 'Đã hết lượt AI, vui lòng thử lại sau.' }],
          stopMessage: 'Đã hết lượt AI, vui lòng thử lại sau.',
        }),
      );
      await settle();
      http.expectOne(url).flush({ ...base, words: [] });
      await settle();
      expect(text('.form-alert')).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
    });

    it('waits while another scene batch runs', async () => {
      await open(
        base,
        sceneBatch({ running: true, key: 'A1-3', label: 'chủ đề “Gia đình”', total: 8, done: 2 }),
      );
      expect(submit().disabled).toBe(true);
      expect(el.textContent).toContain(
        'Máy chủ đang tạo ảnh cảnh ngầm cho chủ đề “Gia đình” (xong 2/8)',
      );
    });

    it('shows why the drawing could not start', async () => {
      await open(base);
      radio('gemini').click();
      await fixture.whenStable();
      submit().click();
      await fixture.whenStable();
      expectStart({});
      await settle();
      http
        .expectOne((r) => r.url === batchUrl && r.method === 'POST')
        .flush(
          { message: 'Đang có một lượt tạo ảnh cảnh chạy. Chờ xong hoặc bấm Dừng.' },
          { status: 409, statusText: 'Conflict' },
        );
      await settle();
      expect(text('.form-alert')).toBe(
        'Đang có một lượt tạo ảnh cảnh chạy. Chờ xong hoặc bấm Dừng.',
      );
    });

    it('saves the description and style without drawing', async () => {
      await open(base);
      await openField('+ Phong cách ảnh');
      area('images-style').value = ' watercolor ';
      area('images-style').dispatchEvent(new Event('input'));
      save().click();
      await fixture.whenStable();
      const req = http.expectOne(url);
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual({
        sceneDescription: 'A family at home',
        style: 'watercolor',
      });
      req.flush({ ...base, style: 'watercolor', words: [] });
      await settle();
      expect(text('p[role="status"]')).toBe('Đã lưu mô tả ảnh cảnh.');
      expect(area('images-style').value).toBe('watercolor');
    });

    it('refuses a description that is too long', async () => {
      await open(base);
      area('images-scene').value = 'x'.repeat(601);
      area('images-scene').dispatchEvent(new Event('input'));
      submit().click();
      await fixture.whenStable();
      expect(text('.form-alert')).toBe('Mô tả cảnh tối đa 600 ký tự');
    });
  });

  it('shows the word bank picture of a word without its own, and offers no AI drawing of words', async () => {
    await open({
      style: '',
      sceneDescription: '',
      count: 0,
      bankCount: 1,
      words: [
        {
          lemma: 'family',
          meaningVi: 'gia đình',
          imageUrl: '/api/admin/lessons/l1/images/family',
          fromBank: true,
        },
      ],
    });
    expect(el.querySelector('.count')).toBeNull();
    const row = el.querySelector('.word:not(.scene)')!;
    expect(row.querySelector('img')).not.toBeNull();
    expect(row.querySelector('.bank-tag')!.textContent).toContain('Ảnh từ Kho từ vựng');
    expect(row.querySelector('.remove-btn')).toBeNull();
    expect(row.querySelector('.upload-btn')!.textContent).toContain('Đổi ảnh');
    expect(row.textContent).not.toContain('Vẽ lại');
    expect(el.textContent).toContain('chỉ sinh trong Kho từ vựng');
  });
});
