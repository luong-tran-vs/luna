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

const counted = {
  pictures: 7,
  points: 2,
  byPoint: [
    { pointId: 'a1-to-be', pictures: 4 },
    { pointId: 'a1-have', pictures: 3 },
  ],
};

// Added 2026-11-12: "Gỡ ảnh ví dụ" of a level, the lessons listed and all ticked at first.
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
  const dialog = () => el.querySelector('[role="alertdialog"]');
  const expectRemove = () => http.expectOne('/admin/grammar-images/remove');

  const open = async (initial = counted) => {
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
    fixture.componentRef.setInput('pointIds', ['a1-to-be', 'a1-have', 'a1-there-is']);
    fixture.componentRef.setInput('titles', { 'a1-to-be': 'Động từ to be', 'a1-have': 'Have got' });
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/admin/grammar-image-batch').flush(idle);
    // The pictures are counted at once, for the button.
    const count = expectRemove();
    expect(count.request.body.dryRun).toBe(true);
    count.flush(initial);
    await settle();
  };
  const ask = async () => {
    button(el, 'Gỡ ảnh ví dụ A1').click();
    await settle();
    const count = expectRemove();
    expect(count.request.body).toEqual({
      pointIds: ['a1-to-be', 'a1-have', 'a1-there-is'],
      dryRun: true,
    });
    count.flush(counted);
    await settle();
  };

  afterEach(() => http.verify());

  it('lists the lessons with pictures, all ticked, and removes them after asking', async () => {
    await open();
    await ask();
    const rows = Array.from(dialog()!.querySelectorAll('.remove-point'));
    expect(rows.map((r) => text(r))).toEqual(['Động từ to be, 4 ảnh', 'Have got, 3 ảnh']);
    expect(rows.every((r) => r.querySelector<HTMLInputElement>('input')!.checked)).toBe(true);
    expect(text(dialog())).toContain('7 ảnh của 2 bài đang tích sẽ bị gỡ');
    button(dialog()!, 'Gỡ 7 ảnh').click();
    await settle();
    const req = expectRemove();
    expect(req.request.body).toEqual({ pointIds: ['a1-to-be', 'a1-have'], dryRun: false });
    req.flush(counted);
    await settle();
    expect(text(el.querySelector('.field-hint[role="status"]'))).toBe('Đã gỡ 7 ảnh của 2 bài.');
    // Counted again: nothing left, the button turns off.
    expectRemove().flush({ pictures: 0, points: 0, byPoint: [] });
    await settle();
    expect(button(el, 'Gỡ ảnh ví dụ A1').disabled).toBe(true);
  });

  it('keeps the pictures of the lessons unticked', async () => {
    await open();
    await ask();
    const tick = (i: number) =>
      dialog()!.querySelectorAll<HTMLInputElement>('.remove-point input')[i];
    tick(0).click();
    await settle();
    expect(text(dialog())).toContain('3 ảnh của 1 bài đang tích');
    tick(1).click();
    await settle();
    expect(button(dialog()!, 'Gỡ 0 ảnh').disabled).toBe(true);
    tick(1).click();
    await settle();
    button(dialog()!, 'Gỡ 3 ảnh').click();
    await settle();
    const req = expectRemove();
    expect(req.request.body).toEqual({ pointIds: ['a1-have'], dryRun: false });
    req.flush({ pictures: 3, points: 1, byPoint: [{ pointId: 'a1-have', pictures: 3 }] });
    await settle();
    expectRemove().flush({
      pictures: 4,
      points: 1,
      byPoint: [{ pointId: 'a1-to-be', pictures: 4 }],
    });
    await settle();
    expect(text(button(el, 'Gỡ ảnh ví dụ A1'))).toBe('Gỡ ảnh ví dụ A1 (4 ảnh)');
  });

  it('turns the button off when the level has no picture', async () => {
    await open({ pictures: 0, points: 0, byPoint: [] });
    const b = button(el, 'Gỡ ảnh ví dụ A1');
    expect(text(b)).toBe('Gỡ ảnh ví dụ A1');
    expect(b.disabled).toBe(true);
  });

  it('shows the count on the button, and says when the server lists no lesson', async () => {
    await open();
    expect(text(button(el, 'Gỡ ảnh ví dụ A1'))).toBe('Gỡ ảnh ví dụ A1 (7 ảnh)');
    button(el, 'Gỡ ảnh ví dụ A1').click();
    await settle();
    // A server from before the list: no empty dialog, but what to do.
    expectRemove().flush({ pictures: 7, points: 2 });
    await settle();
    expect(dialog()).toBeNull();
    expect(text(el.querySelector('[role="alert"]'))).toContain('deploy lại backend');
  });
});
