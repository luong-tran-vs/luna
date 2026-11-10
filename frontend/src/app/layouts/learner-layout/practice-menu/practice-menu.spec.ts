import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { PracticeMenu } from './practice-menu';

const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
const daysAgo = (n: number) => new Date(Date.now() - n * 86_400_000).toISOString();

describe('PracticeMenu (added 2026-11-10)', () => {
  let fixture: ComponentFixture<PracticeMenu>;
  let el: HTMLElement;
  let http: HttpTestingController;
  const toggle = () => el.querySelector<HTMLButtonElement>('.toggle')!;

  beforeEach(async () => {
    TestBed.configureTestingModule({
      imports: [PracticeMenu],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(PracticeMenu);
    el = fixture.nativeElement;
    await fixture.whenStable();
    // "let" typed wrong twice lately: one verb often typed wrong.
    http.expectOne('/verbs/progress').flush({
      verbs: [
        {
          base: 'let',
          level: 1,
          learnedAt: daysAgo(3),
          mistakes: [
            { form: 'past', typed: 'letted', at: daysAgo(2) },
            { form: 'past', typed: 'lat', at: daysAgo(1) },
          ],
        },
      ],
    });
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('shows a dot on the closed button when verbs are often typed wrong', () => {
    expect(toggle().getAttribute('aria-expanded')).toBe('false');
    expect(toggle().getAttribute('aria-label')).toBe('Luyện tập, 1 từ hay sai');
    expect(el.querySelector('.fab-dot')).not.toBeNull();
    expect(el.querySelector('.panel')).toBeNull();
  });

  it('opens the card with the lesson to go on with and the six sections, and closes on Esc', async () => {
    toggle().click();
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs: [] });
    http
      .expectOne('/dashboard')
      .flush({ lesson: { id: 'l3', title: 'Family members', topicName: 'Gia đình', level: 'A1' } });
    await fixture.whenStable();

    expect(toggle().getAttribute('aria-expanded')).toBe('true');
    expect(text(toggle().querySelector('.label'))).toBe('Đóng');
    expect(el.querySelector('.continue')?.getAttribute('href')).toBe('/lessons/l3');
    expect(text(el.querySelector('.continue'))).toContain('Family members');
    expect(
      Array.from(el.querySelectorAll('.tile')).map((t) => [
        t.querySelector('.tile-label')?.textContent,
        t.getAttribute('href'),
      ]),
    ).toEqual([
      ['Ngữ pháp', '/grammar'],
      ['Động từ', '/irregular-verbs'],
      ['Luyện nghe', '/irregular-verbs/listen'],
      ['Từ hay sai', '/irregular-verbs/mistakes'],
      ['Trò chuyện', '/vocabulary/talk'],
      ['Tiến độ', '/stats'],
    ]);
    // Nothing learned on the second load: lesson 1 next, no verb typed wrong.
    expect(text(el.querySelectorAll('.tile')[1])).toContain('Bài 1/42');
    expect(text(el.querySelectorAll('.tile')[3])).toContain('Chưa có');

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await fixture.whenStable();
    expect(el.querySelector('.panel')).toBeNull();
    expect(toggle().getAttribute('aria-expanded')).toBe('false');
  });

  it('closes when the button is tapped again or outside the card', async () => {
    toggle().click();
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs: [] });
    http.expectOne('/dashboard').flush({ lesson: null });
    await fixture.whenStable();
    expect(el.querySelector('.continue')).toBeNull();

    el.querySelector<HTMLElement>('.scrim')!.click();
    await fixture.whenStable();
    expect(el.querySelector('.panel')).toBeNull();

    toggle().click();
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs: [] });
    http.expectOne('/dashboard').flush({ lesson: null });
    toggle().click();
    await fixture.whenStable();
    expect(el.querySelector('.panel')).toBeNull();
  });
});
