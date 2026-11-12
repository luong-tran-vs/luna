import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import {
  BankImageBatch,
  CardWordsSummary,
  MAX_IMAGE_BATCH,
} from '../../../../core/models/word-bank';
import { batchScope, ImageBatchService } from '../image-batch.service';
import { CardWordPictures } from './card-word-pictures';

const summary = (extra: Partial<CardWordsSummary> = {}): CardWordsSummary => ({
  total: 5,
  withImage: 2,
  missingCount: 3,
  missing: [
    { lemma: 'run', meaningVi: 'chạy', learners: 4, inBank: false },
    { lemma: 'house', meaningVi: 'ngôi nhà', learners: 2, inBank: true },
    { lemma: 'swim', meaningVi: 'bơi', learners: 1, inBank: true },
  ],
  codexImages: false,
  ...extra,
});

const running: BankImageBatch = {
  running: true,
  stopping: false,
  stopped: false,
  engine: 'gemini',
  ticked: true,
  cardWords: true,
  topics: [],
  total: 3,
  done: 0,
  drawing: ['run'],
  saved: [],
  failures: [],
};

describe('CardWordPictures', () => {
  let fixture: ComponentFixture<CardWordPictures>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;

  const open = async (data: CardWordsSummary = summary()) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(CardWordPictures);
    el = fixture.nativeElement as HTMLElement;
    http.expectOne('/admin/words/card-words').flush(data);
    await settle();
  };

  afterEach(() => http.verify());

  it('counts the saved words with a picture and names the most saved without', async () => {
    await open();
    expect(text(el.querySelector('.count'))).toBe('Có ảnh 2/5 từ học viên đã lưu');
    expect(text(el)).toContain('Lưu nhiều nhất mà chưa có ảnh: run (4)*, house (2), swim (1)');
    expect(text(el)).toContain('* Chưa có trong Kho');
    expect(text(button('Sinh ảnh còn thiếu'))).toBe('Sinh ảnh còn thiếu (3)');
  });

  it('says when every word has its picture, or none was saved', async () => {
    await open(summary({ withImage: 5, missingCount: 0, missing: [] }));
    expect(text(el)).toContain('Mọi từ trong thẻ ôn đã có ảnh.');
    expect(el.querySelector('.btn-primary')).toBeNull();
  });

  it('starts the batch with the engine picked, the page following it', async () => {
    await open(summary({ missingCount: MAX_IMAGE_BATCH + 7 }));
    button('Sinh ảnh còn thiếu').click();
    await settle();
    const dialog = el.querySelector('lu-engine-dialog [role="dialog"]')!;
    expect(text(dialog)).toContain(`Máy chủ vẽ ${MAX_IMAGE_BATCH} từ được lưu nhiều nhất`);
    expect(text(dialog)).toContain('7 từ còn lại vẽ ở lượt sau');
    dialog.querySelector<HTMLInputElement>('input[value="gemini"]')!.click();
    await settle();
    button(`Sinh ${MAX_IMAGE_BATCH} ảnh`).click();
    await settle();

    const req = http.expectOne('/admin/words/card-words/image-batch');
    expect(req.request.body).toEqual({ engine: 'gemini' });
    req.flush(running);
    await settle();
    const batches = TestBed.inject(ImageBatchService);
    expect(batches.running()).toBe(true);
    expect(button('Sinh ảnh còn thiếu').disabled).toBe(true);
    expect(batchScope(running)).toBe('3 từ trong thẻ ôn của học viên');

    // Once the batch ends, the words are counted again.
    batches.set({ ...running, running: false, done: 3 });
    await settle();
    http
      .expectOne('/admin/words/card-words')
      .flush(summary({ withImage: 5, missingCount: 0, missing: [] }));
    await settle();
    expect(text(el)).toContain('Mọi từ trong thẻ ôn đã có ảnh.');
  });
});
