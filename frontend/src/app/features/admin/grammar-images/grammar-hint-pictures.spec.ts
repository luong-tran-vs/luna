import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarImageBatch } from '../../../core/models/grammar-admin';
import { GrammarHintPictures } from './grammar-hint-pictures';

const idle: GrammarImageBatch = {
  running: false,
  stopping: false,
  stopped: false,
  engine: '',
  key: '',
  label: '',
  total: 0,
  points: 0,
  done: 0,
  drawing: [],
  saved: [],
  failures: [],
  codexImages: false,
};

describe('GrammarHintPictures', () => {
  let fixture: ComponentFixture<GrammarHintPictures>;
  let el: HTMLElement;
  let http: HttpTestingController;
  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarHintPictures);
    fixture.componentRef.setInput('pointId', 'a1-to-be');
    fixture.componentRef.setInput('title', 'to be');
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/admin/grammar-image-batch').flush(idle);
    http.expectOne('/admin/grammar-lessons/a1-to-be/hint-pictures').flush({
      items: [
        { exerciseId: 'p1', kind: 'choice', sentence: 'She is a teacher.', key: 'k1' },
        {
          exerciseId: 'p2',
          kind: 'fill',
          sentence: 'They are happy.',
          key: 'k2',
          imageUrl: '/api/admin/grammar-lessons/a1-to-be/images/k2',
        },
      ],
    });
    await settle();
  });

  afterEach(() => http.verify());

  it('lists the choice sentences with their pictures and draws the missing ones as a batch', async () => {
    expect(text(el.querySelector('.head .muted'))).toBe('1/2 ảnh đã có');
    const rows = el.querySelectorAll('.row');
    expect(text(rows[0])).toContain('Chưa có ảnh');
    expect(rows[1].querySelector('img')).not.toBeNull();

    button('Tạo 1 ảnh câu còn thiếu').click();
    await settle();
    el.querySelector<HTMLInputElement>('input[value="gemini"]')!.click();
    await settle();
    button('Tạo ảnh').click();
    await settle();
    const req = http.expectOne(
      (r) => r.url === '/admin/grammar-image-batch' && r.method === 'POST',
    );
    expect(req.request.body).toEqual({
      pointIds: ['a1-to-be'],
      hints: true,
      keys: undefined,
      replace: undefined,
      engine: 'gemini',
      key: 'hints:point:a1-to-be',
      label: 'ảnh câu bài “to be”',
    });
    req.flush({ ...idle, running: true, key: 'hints:point:a1-to-be', total: 1 });
    await settle();
    expect(text(el.querySelector('.bg-note'))).toContain('xong 0/1');
  });
});
