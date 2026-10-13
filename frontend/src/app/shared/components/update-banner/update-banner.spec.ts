import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AppUpdateService } from '../../../core/services/app-update.service';
import { UpdateBanner } from './update-banner';

describe('UpdateBanner', () => {
  let fixture: ComponentFixture<UpdateBanner>;
  let el: HTMLElement;
  const ready = signal(false);
  const update = vi.fn();

  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === label);

  beforeEach(async () => {
    ready.set(false);
    update.mockClear();
    TestBed.configureTestingModule({
      providers: [{ provide: AppUpdateService, useValue: { ready, update } }],
    });
    fixture = TestBed.createComponent(UpdateBanner);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('shows nothing until a new version is ready', async () => {
    expect(el.querySelector('.bar')).toBeNull();
    ready.set(true);
    await fixture.whenStable();
    expect(el.querySelector('[role="status"]')?.textContent).toContain(
      'Đã có phiên bản mới của Luna.',
    );
  });

  it('updates on Cập nhật, and hides on Để sau', async () => {
    ready.set(true);
    await fixture.whenStable();
    button('Cập nhật')!.click();
    expect(update).toHaveBeenCalledOnce();
    button('Để sau')!.click();
    await fixture.whenStable();
    expect(el.querySelector('.bar')).toBeNull();
  });
});
