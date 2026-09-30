import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { DueCard, ReviewContext, ReviewMode, ReviewSummary } from '../../../core/models/vocab';
import { ReviewSession } from './review-session';

const intervals = { again: 60, hard: 300, good: 600, easy: 8 * 86400 };

function card(id: string, text: string, reps = 0): DueCard {
  return {
    id, text, lemma: text, ipa: '/ɡəʊ/', meaningVi: `nghĩa của ${text}`, contextSentence: `We ${text} home.`,
    lessonId: 'l1', source: 'ai', createdAt: '2026-09-28T01:00:00Z', due: '2026-09-29T00:00:00Z', reps, state: 'new',
    intervals,
  };
}

@Component({
  imports: [ReviewSession],
  template: `<lu-review-session [cards]="cards()" [mode]="mode()" [context]="context()" (finished)="summaries.push($event)" />`,
})
class Host {
  readonly cards = signal<DueCard[]>([card('c1', 'went'), card('c2', 'gave up')]);
  readonly mode = signal<ReviewMode>('flip');
  readonly context = signal<ReviewContext>('free');
  readonly summaries: ReviewSummary[] = [];
}

describe('ReviewSession', () => {
  let fixture: ComponentFixture<Host>;
  let host: Host;
  let el: HTMLElement;
  let http: HttpTestingController;
  let play: ReturnType<typeof vi.spyOn>;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (selector: string) => el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.replace(/\s+/g, ' ').trim() === label);
  const buttonStarting = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.replace(/\s+/g, ' ').trim().startsWith(label));
  const expectReview = (id: string): TestRequest => http.expectOne(`/api/vocab/cards/${id}/review`);
  const keydown = async (key: string) => {
    el.querySelector('lu-review-session')!.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
    await settle();
  };
  const input = () => el.querySelector<HTMLInputElement>('#review-answer')!;
  const typeAndCheck = async (value: string) => {
    input().value = value;
    input().dispatchEvent(new Event('input'));
    button('Kiểm tra')!.click();
    await settle();
  };

  const setup = async (mode: ReviewMode = 'flip', cards?: DueCard[]) => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Host);
    host = fixture.componentInstance;
    host.mode.set(mode);
    if (cards) {
      host.cards.set(cards);
    }
    el = fixture.nativeElement as HTMLElement;
    await settle();
  };

  beforeEach(() => {
    play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined);
    vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => undefined);
    Element.prototype.scrollIntoView = vi.fn();
  });

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  describe('flip mode', () => {
    it('shows only the word until the card is flipped', async () => {
      await setup();
      expect(text('.progress')).toBe('Thẻ 1/2');
      expect(text('.word')).toBe('went');
      expect(button('Nghe')).toBeTruthy();
      expect(el.querySelector('.meaning')).toBeNull();
      expect(buttonStarting('Good')).toBeUndefined();

      button('Lật thẻ')!.click();
      await settle();
      expect(text('.ipa')).toBe('/ɡəʊ/');
      expect(text('.meaning')).toBe('nghĩa của went');
      expect(text('.example')).toBe('We went home.');
      expect(['Again · 1 phút', 'Hard · 5 phút', 'Good · 10 phút', 'Easy · 8 ngày'].map((l) => !!button(l))).toEqual([
        true,
        true,
        true,
        true,
      ]);
    });

    it('flips with Space and rates with number keys', async () => {
      await setup();
      await keydown(' ');
      expect(el.querySelector('.meaning')).toBeTruthy();
      await keydown('3');
      const req = expectReview('c1');
      expect(req.request.body).toEqual({ rating: 3, mode: 'flip', reps: 0, context: 'free' });
      req.flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();
      expect(text('.word')).toBe('gave up');
      expect(el.querySelector('.meaning')).toBeNull();
    });

    it('sends the review context', async () => {
      await setup();
      host.context.set('daily');
      await settle();
      button('Lật thẻ')!.click();
      await settle();
      button('Good · 10 phút')!.click();
      await settle();
      const req = expectReview('c1');
      expect(req.request.body).toEqual({ rating: 3, mode: 'flip', reps: 0, context: 'daily' });
      req.flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();
    });

    it('plays the word', async () => {
      await setup();
      button('Nghe')!.click();
      await settle();
      expect(el.querySelector('audio')!.getAttribute('src')).toBe('/api/tts/word?text=went');
      expect(play).toHaveBeenCalled();
    });

    it('shows a card rated Again once more at the end and reports the summary', async () => {
      await setup();
      button('Lật thẻ')!.click();
      await settle();
      button('Again · 1 phút')!.click();
      await settle();
      expectReview('c1').flush({ card: { ...card('c1', 'went', 1), intervals: { ...intervals, good: 3600 } } });
      await settle();

      button('Lật thẻ')!.click();
      await settle();
      button('Good · 10 phút')!.click();
      await settle();
      expectReview('c2').flush({ card: { ...card('c2', 'gave up', 1), intervals } });
      await settle();

      // c1 again, with the intervals of its new schedule.
      expect(text('.word')).toBe('went');
      expect(text('.progress')).toBe('Thẻ 3/3');
      button('Lật thẻ')!.click();
      await settle();
      expect(button('Good · 1 giờ')).toBeTruthy();
      button('Again · 1 phút')!.click();
      await settle();
      const again = expectReview('c1');
      expect(again.request.body.reps).toBe(1);
      again.flush({ card: { ...card('c1', 'went', 2), intervals } });
      await settle();

      // Not re-queued a second time.
      expect(text('.summary')).toContain('Đã ôn 2 thẻ');
      expect(text('.summary')).toContain('Again 2');
      expect(text('.summary')).toContain('Good 1');
      expect(host.summaries).toEqual([{ reviewed: 2, counts: { 1: 2, 2: 0, 3: 1, 4: 0 } }]);
    });

    it('moves on when the card was already reviewed (409)', async () => {
      await setup();
      button('Lật thẻ')!.click();
      await settle();
      button('Good · 10 phút')!.click();
      await settle();
      expectReview('c1').flush(
        { error: 'review_conflict', message: 'x', card: card('c1', 'went', 1) },
        { status: 409, statusText: 'Conflict' },
      );
      await settle();
      expect(text('.word')).toBe('gave up');
    });

    it('keeps the card and offers a retry when saving fails', async () => {
      await setup();
      button('Lật thẻ')!.click();
      await settle();
      button('Hard · 5 phút')!.click();
      await settle();
      expectReview('c1').flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'Error' });
      await settle();
      expect(text('.save-error')).toContain('Chưa lưu được đánh giá');
      expect(text('.word')).toBe('went');

      button('Thử lại')!.click();
      await settle();
      const retry = expectReview('c1');
      expect(retry.request.body).toEqual({ rating: 2, mode: 'flip', reps: 0, context: 'free' });
      retry.flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();
      expect(text('.word')).toBe('gave up');
      expect(el.querySelector('.save-error')).toBeNull();
    });
  });

  describe('listen mode', () => {
    it('plays the word automatically and hides it', async () => {
      await setup('listen');
      expect(el.querySelector('audio')!.getAttribute('src')).toBe('/api/tts/word?text=went');
      expect(play).toHaveBeenCalled();
      expect(el.querySelector('.word')).toBeNull();
      expect(button('Nghe lại')).toBeTruthy();
      expect(input().getAttribute('autocapitalize')).toBe('off');
      expect(input().getAttribute('spellcheck')).toBe('false');
    });

    it('asks for an answer when empty', async () => {
      await setup('listen');
      await typeAndCheck('  ');
      expect(text('.answer-error')).toBe('Hãy gõ từ bạn nghe được');
      expect(buttonStarting('Good')).toBeUndefined();
    });

    it('accepts any case and shows the card', async () => {
      await setup('listen');
      await typeAndCheck('WENT');
      expect(text('.verdict')).toBe('✓ Đúng');
      expect(text('.word')).toBe('went');
      expect(text('.meaning')).toBe('nghĩa của went');
      expect(buttonStarting('Good')).toBeTruthy();
      button('Easy · 8 ngày')!.click();
      await settle();
      expect(expectReview('c1').request.body).toEqual({ rating: 4, mode: 'listen', reps: 0, context: 'free' });
    });

    it('says wrong and shows the right word', async () => {
      await setup('listen');
      await typeAndCheck('want');
      expect(text('.verdict')).toBe('✗ Sai');
      expect(text('.word')).toBe('went');
    });

    it('lets the learner see the word when audio cannot play', async () => {
      play.mockRejectedValue(new Error('NotAllowedError'));
      await setup('listen');
      expect(text('.play-error')).toContain('Chưa phát được âm thanh');
      button('Hiện từ')!.click();
      await settle();
      expect(text('.word')).toBe('went');
      expect(buttonStarting('Good')).toBeTruthy();
    });
  });
});
