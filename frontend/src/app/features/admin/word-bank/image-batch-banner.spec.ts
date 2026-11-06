import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { BankImageBatch } from '../../../core/models/word-bank';
import { ImageBatchBanner } from './image-batch-banner';
import { BANK_BATCH_POLL_MS } from './image-batch.service';

describe('ImageBatchBanner', () => {
  const idle: BankImageBatch = {
    running: false,
    stopping: false,
    stopped: false,
    engine: '',
    ticked: false,
    topics: [],
    total: 0,
    done: 0,
    drawing: [],
    saved: [],
    failures: [],
  };

  const setup = () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        { provide: BANK_BATCH_POLL_MS, useValue: 10 },
      ],
    });
    const http = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(ImageBatchBanner);
    const el = fixture.nativeElement as HTMLElement;
    const text = () => el.textContent?.replace(/\s+/g, ' ').trim() ?? '';
    return { http, fixture, el, text };
  };

  it('shows nothing when no batch ran', async () => {
    const { http, fixture, text } = setup();
    http.expectOne('/admin/words/image-batch').flush(idle);
    await fixture.whenStable();
    expect(text()).toBe('');
    http.verify();
  });

  it('labels a running batch with its progress, then its result until hidden', async () => {
    const { http, fixture, el, text } = setup();
    http.expectOne('/admin/words/image-batch').flush({
      ...idle,
      running: true,
      engine: 'codex',
      topicName: 'Nhà cửa',
      topics: ['Nhà cửa', 'Chào hỏi'],
      total: 40,
      done: 12,
      failures: [{ lemma: 'x', message: 'lỗi' }],
    });
    await fixture.whenStable();
    expect(text()).toContain(
      'Máy chủ đang sinh ảnh ngầm cho các từ thiếu ảnh của chủ đề "Nhà cửa" bằng Codex CLI: xong 12/40 từ, còn 28 từ · 1 lỗi',
    );
    expect(el.querySelector('a')!.getAttribute('href')).toBe('/admin/words');

    await new Promise((r) => setTimeout(r, 20));
    http.expectOne('/admin/words/image-batch').flush({
      ...idle,
      engine: 'codex',
      total: 40,
      done: 40,
      saved: [],
      startedAt: 't',
      finishedAt: 'u',
    });
    await fixture.whenStable();
    expect(text()).toContain('Đã sinh ảnh ngầm xong cho mọi từ thiếu ảnh: 0/40 ảnh');
    el.querySelector<HTMLButtonElement>('button')!.click();
    await fixture.whenStable();
    expect(text()).toBe('');
    fixture.destroy();
    http.verify();
  });
});
