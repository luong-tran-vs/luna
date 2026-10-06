import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import { DEFAULT_IMAGE_STYLE } from '../../../../core/models/generate';
import { LessonImages } from './lesson-images';

describe('LessonImages', () => {
  let fixture: ComponentFixture<LessonImages>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const url = '/api/admin/lessons/l1/images';
  const text = (selector: string) =>
    el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const toggle = () => el.querySelector<HTMLButtonElement>('button[role="switch"]')!;
  const submit = () => el.querySelector<HTMLButtonElement>('button[type="submit"]')!;

  const open = async (state: object) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LessonImages);
    fixture.componentRef.setInput('lessonId', 'l1');
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne(url).flush({ words: [], ...state });
    await settle();
  };

  afterEach(() => http.verify());

  describe('F23: uploading a picture for each word', () => {
    const words = [
      { lemma: 'coffee', meaningVi: 'cà phê', imageUrl: '' },
      { lemma: 'give up', meaningVi: 'từ bỏ', imageUrl: '/api/admin/lessons/l1/images/give%20up' },
    ];
    const base = { enabled: false, style: '', status: '', error: '', count: 1, words };
    const rows = () => Array.from(el.querySelectorAll('.word'));
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
      expect(rows()[1].querySelector('img')!.getAttribute('src')).toMatch(
        /^\/api\/admin\/lessons\/l1\/images\/give%20up\?v=\d+$/,
      );
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
      expect(el.querySelector('.word')).toBeNull();
    });

    it('sends the chosen file as the body and shows the new picture', async () => {
      await open(base);
      const file = new File(['png'], 'cup.png', { type: 'image/png' });
      await choose(rows()[0], file);
      const req = http.expectOne('/api/admin/lessons/l1/images/coffee');
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
      expect(text('.count')).toBe('2 từ đã có ảnh');
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
        .expectOne('/api/admin/lessons/l1/images/coffee')
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
        const req = http.expectOne('/api/admin/lessons/l1/images/coffee/import');
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
        http
          .expectOne('/api/admin/lessons/l1/images/coffee/import')
          .flush(
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
      const req = http.expectOne('/api/admin/lessons/l1/images/give%20up');
      expect(req.request.method).toBe('DELETE');
      req.flush({ ...base, count: 0, words: [words[0], { ...words[1], imageUrl: '' }] });
      await settle();
      expect(rows()[1].querySelector('img')).toBeNull();
      expect(text('.word [role="status"]')).toBe('Đã xoá ảnh.');
    });
  });

  it('shows pictures off, with the default style offered when turned on', async () => {
    await open({ enabled: false, style: '', status: '', error: '', count: 0 });
    expect(text('.none-chip')).toBe('Ảnh: Tắt');
    expect(text('.count')).toBe('0 từ đã có ảnh');
    expect(toggle().getAttribute('aria-checked')).toBe('false');
    expect(el.querySelector('#images-style')).toBeNull();
    toggle().click();
    await fixture.whenStable();
    expect(el.querySelector<HTMLTextAreaElement>('#images-style')!.value).toBe(DEFAULT_IMAGE_STYLE);
    expect(text('button[type="submit"]')).toBe('Lưu và sinh ảnh còn thiếu');
  });

  it('shows a failed drawing with its reason, and saving retries it', async () => {
    await open({
      enabled: true,
      style: 'watercolor',
      status: 'failed',
      error: 'AI hết hạn mức, thử lại sau',
      count: 3,
    });
    expect(text('lu-status-chip')).toContain('Ảnh: Lỗi');
    expect(el.textContent).toContain('Lý do: AI hết hạn mức, thử lại sau');
    expect(el.querySelector<HTMLTextAreaElement>('#images-style')!.value).toBe('watercolor');

    submit().click();
    await fixture.whenStable();
    const req = http.expectOne(url);
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ enabled: true, style: 'watercolor' });
    req.flush({
      enabled: true,
      style: 'watercolor',
      status: 'running',
      error: '',
      count: 3,
      words: [],
    });
    await settle();
    expect(text('lu-status-chip')).toContain('Ảnh: Đang chạy');
    expect(text('[role="status"]')).toBe('Đã lưu. Ảnh của các từ còn thiếu đang được sinh.');
  });

  it('turns pictures off and shows server errors', async () => {
    await open({ enabled: true, style: 'x', status: 'done', error: '', count: 5 });
    toggle().click();
    await fixture.whenStable();
    submit().click();
    await fixture.whenStable();
    http
      .expectOne(url)
      .flush(
        { message: 'Máy chủ chưa hỗ trợ ảnh từ vựng' },
        { status: 503, statusText: 'Unavailable' },
      );
    await settle();
    expect(text('[role="alert"]')).toBe('Máy chủ chưa hỗ trợ ảnh từ vựng');
  });
});
