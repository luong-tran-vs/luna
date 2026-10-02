import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Loading } from './loading';

describe('Loading', () => {
  let fixture: ComponentFixture<Loading>;
  let el: HTMLElement;

  beforeEach(async () => {
    fixture = TestBed.createComponent(Loading);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('is a status with a decorative spinner and the default label for screen readers', () => {
    expect(el.getAttribute('role')).toBe('status');
    expect(el.querySelector('svg')!.getAttribute('aria-hidden')).toBe('true');
    expect(el.textContent!.trim()).toBe('Đang tải…');
  });

  it('reads the given label', async () => {
    fixture.componentRef.setInput('label', 'Đang tải bài…');
    await fixture.whenStable();
    expect(el.querySelector('.label')!.textContent).toBe('Đang tải bài…');
  });
});
