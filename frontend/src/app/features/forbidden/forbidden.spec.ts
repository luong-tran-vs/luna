import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Forbidden } from './forbidden';

describe('Forbidden', () => {
  it('explains the missing permission and links home', async () => {
    await TestBed.configureTestingModule({
      imports: [Forbidden],
      providers: [provideRouter([])],
    }).compileComponents();

    const fixture = TestBed.createComponent(Forbidden);
    await fixture.whenStable();
    const el = fixture.nativeElement as HTMLElement;

    expect(el.querySelector('h1')?.textContent?.trim()).toBe('Bạn không có quyền truy cập trang này');
    expect(el.querySelector('a[href="/"]')).not.toBeNull();
  });
});
