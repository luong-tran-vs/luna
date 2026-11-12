import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { GrammarImageBatch } from '../../../core/models/grammar-admin';
import { GrammarLevelImages } from './grammar-level-images';

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

// Added 2026-11-12: "Gỡ ảnh ví dụ" of a level.
describe('GrammarLevelImages', () => {
  let fixture: ComponentFixture<GrammarLevelImages>;
  let el: HTMLElement;
  let http: HttpTestingController;
  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const button = (root: Element, label: string) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      text(b).startsWith(label),
    )!;
  const expectRemove = () => http.expectOne('/admin/grammar-images/remove');

  const open = async () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideRouter([]),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GrammarLevelImages);
    fixture.componentRef.setInput('level', 'A1');
    fixture.componentRef.setInput('pointIds', ['a1-to-be', 'a1-have']);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/admin/grammar-image-batch').flush(idle);
    await settle();
  };

  afterEach(() => http.verify());

  it('counts the pictures first, then removes them after asking', async () => {
    await open();
    button(el, 'Gỡ ảnh ví dụ A1').click();
    await settle();
    const count = expectRemove();
    expect(count.request.body).toEqual({ pointIds: ['a1-to-be', 'a1-have'], dryRun: true });
    count.flush({ pictures: 7, points: 2 });
    await settle();
    const dialog = el.querySelector('[role="alertdialog"]')!;
    expect(text(dialog)).toContain('7 ảnh của 2 bài sẽ bị gỡ');
    button(dialog, 'Gỡ 7 ảnh').click();
    await settle();
    const req = expectRemove();
    expect(req.request.body).toEqual({ pointIds: ['a1-to-be', 'a1-have'], dryRun: false });
    req.flush({ pictures: 7, points: 2 });
    await settle();
    expect(text(el.querySelector('.field-hint[role="status"]'))).toBe('Đã gỡ 7 ảnh của 2 bài.');
  });

  it('says when the level has no picture, without asking', async () => {
    await open();
    button(el, 'Gỡ ảnh ví dụ A1').click();
    await settle();
    expectRemove().flush({ pictures: 0, points: 0 });
    await settle();
    expect(el.querySelector('[role="alertdialog"]')).toBeNull();
    expect(text(el.querySelector('.field-hint[role="status"]'))).toBe(
      'Các bài ngữ pháp A1 chưa có ảnh nào.',
    );
  });
});
