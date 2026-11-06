import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarPictureBatch } from '../../../core/models/grammar-admin';
import { BATCH_IDLE_POLL_MS, BATCH_POLL_MS } from '../batch-watcher';
import { GrammarLevelPictures } from './grammar-level-pictures';

const BATCH = '/admin/grammar-picture-batch';

const batch = (over: Partial<GrammarPictureBatch> = {}): GrammarPictureBatch => ({
  running: false,
  stopping: false,
  stopped: false,
  key: '',
  label: '',
  total: 0,
  done: 0,
  current: null,
  saved: [],
  failures: [],
  ...over,
});

describe('GrammarLevelPictures', () => {
  let fixture: ComponentFixture<GrammarLevelPictures>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = () => el.querySelector<HTMLButtonElement>('.line button')!;

  const setup = async (status: GrammarPictureBatch = batch()) => {
    TestBed.configureTestingModule({
      imports: [GrammarLevelPictures],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: BATCH_POLL_MS, useValue: 60000 },
        { provide: BATCH_IDLE_POLL_MS, useValue: 60000 },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarLevelPictures);
    fixture.componentRef.setInput('level', 'A1');
    fixture.componentRef.setInput('pointIds', ['a1-to-be', 'a1-can']);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne(BATCH).flush(status);
    await settle();
  };

  afterEach(() => http.verify());

  it('starts the batch of the level in the background and shows how it goes', async () => {
    await setup();
    expect(text(el)).toContain('3 bài “Nhìn ảnh chọn từ” ở Luyện tập và 1 bài ở Kiểm tra');
    expect(text(button())).toBe('Thêm bài nhìn ảnh A1');
    button().click();
    await settle();
    const req = http.expectOne((r) => r.url === BATCH && r.method === 'POST');
    expect(req.request.body).toEqual({
      pointIds: ['a1-to-be', 'a1-can'],
      key: 'level:A1',
      label: 'các bài ngữ pháp A1',
    });
    req.flush(
      batch({
        running: true,
        key: 'level:A1',
        label: 'các bài ngữ pháp A1',
        total: 2,
        current: { pointId: 'a1-to-be', title: 'Động từ to be', added: 0 },
        startedAt: 't',
      }),
    );
    await settle();
    expect(text(el.querySelector('.status'))).toContain('xong 0/2 bài · đang soạn “Động từ to be”');
    expect(text(button())).toBe('Dừng');

    button().click();
    await settle();
    http
      .expectOne(`${BATCH}/stop`)
      .flush(batch({ running: true, stopping: true, key: 'level:A1', total: 2, startedAt: 't' }));
    await settle();
    expect(text(button())).toBe('Đang dừng…');
  });

  it('shows the result with the lessons that failed', async () => {
    await setup(
      batch({
        key: 'level:A1',
        total: 2,
        done: 2,
        saved: [{ pointId: 'a1-to-be', title: 'Động từ to be', added: 4 }],
        failures: [
          {
            pointId: 'a1-can',
            title: 'Can',
            added: 0,
            message: 'AI trả về nội dung không dùng được',
          },
        ],
        startedAt: 't',
      }),
    );
    const status = text(el.querySelector('.status'));
    expect(status).toContain('Đã thêm bài nhìn ảnh xong. 1/2 bài, 4 bài tập mới · 1 lỗi');
    expect(text(el.querySelector('.failures'))).toContain(
      'Can: AI trả về nội dung không dùng được',
    );
  });

  it('says why it could not start', async () => {
    await setup();
    button().click();
    await settle();
    http
      .expectOne((r) => r.url === BATCH && r.method === 'POST')
      .flush(
        { error: 'validation', fields: { pointIds: 'Mọi bài vẽ được đã có đủ bài nhìn ảnh' } },
        { status: 400, statusText: 'x' },
      );
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Mọi bài vẽ được đã có đủ bài nhìn ảnh');
  });

  it('waits while another level runs', async () => {
    await setup(
      batch({ running: true, key: 'level:A2', label: 'các bài ngữ pháp A2', total: 3, done: 1 }),
    );
    expect(button().disabled).toBe(true);
    expect(text(el)).toContain('cho các bài ngữ pháp A2 (xong 1/3)');
  });
});
