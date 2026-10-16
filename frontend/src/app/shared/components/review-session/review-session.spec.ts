import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { DueCard, ReviewContext, ReviewMode, ReviewSummary } from '../../../core/models/vocab';
import { exampleParts, ReviewSession } from './review-session';

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
  template: `<lu-review-session [cards]="cards()" [mode]="mode()" [context]="context()" [pool]="pool()" (finished)="summaries.push($event)" (closed)="closed = closed + 1" />`,
})
class Host {
  readonly cards = signal<DueCard[]>([card('c1', 'went'), card('c2', 'gave up')]);
  readonly mode = signal<ReviewMode>('flip');
  readonly context = signal<ReviewContext>('free');
  readonly pool = signal<string[]>([]);
  readonly summaries: ReviewSummary[] = [];
  closed = 0;
}

describe('ReviewSession', () => {
  let fixture: ComponentFixture<Host>;
  let host: Host;
  let el: HTMLElement;
  let http: HttpTestingController;
  let speech: FakeSpeech;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (selector: string) => el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  // By its text or, for an icon or a rating button, its aria-label.
  const button = (label: string) =>
    Array.from(el.querySelectorAll('button')).find(
      (b) => b.textContent?.replace(/\s+/g, ' ').trim() === label || b.getAttribute('aria-label') === label,
    );
  const buttonStarting = (label: string) =>
    Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.replace(/\s+/g, ' ').trim().startsWith(label));
  const expectReview = (id: string): TestRequest => http.expectOne(`/vocab/cards/${id}/review`);
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
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        provideFakeSpeech(speech),
      ],
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
    speech = new FakeSpeech();
    Element.prototype.scrollIntoView = vi.fn();
  });

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  describe('flip mode', () => {
    // Pick a meaning, "Nộp", then "Tiếp" (design/on-tap-theo-bai.png).
    const choose = async (meaning: string) => {
      Array.from(el.querySelectorAll<HTMLButtonElement>('.choice'))
        .find((c) => c.querySelector('.text')?.textContent?.trim() === meaning)!
        .click();
      await settle();
    };
    const submit = async (meaning: string) => {
      await choose(meaning);
      button('Nộp')!.click();
      await settle();
    };
    const goOn = async () => {
      el.querySelector<HTMLButtonElement>('.btn-next')!.click();
      await settle();
    };
    const options = () => Array.from(el.querySelectorAll('.choice .text')).map((t) => t.textContent?.trim());

    it('asks for the meaning among the others', async () => {
      await setup();
      expect(text('.progress-text')).toBe('1/2');
      expect(text('.word')).toBe('went');
      expect(button('Nghe từ')).toBeTruthy();
      expect(text('.counter')).toBe('Từ này nghĩa là gì?');
      expect([...options()].sort()).toEqual(['nghĩa của gave up', 'nghĩa của went']);
      expect(el.querySelector('.meaning')).toBeNull();
      // The example sentence is on the front, the word marked, to guess the meaning from context.
      expect(text('.example')).toBe('We went home.');
      expect(text('.example mark')).toBe('went');
      expect(el.querySelector('.example-vi')).toBeNull();
      expect(button('Quên')).toBeTruthy();
      expect(buttonStarting('Nhớ')).toBeUndefined();

      // Nothing picked: "Nộp" does nothing.
      expect(button('Nộp')!.getAttribute('aria-disabled')).toBe('true');
      button('Nộp')!.click();
      await settle();
      expect(el.querySelector('.meaning')).toBeNull();
    });

    it('rates a right answer Good on "Tiếp"', async () => {
      await setup();
      await choose('nghĩa của went');
      expect(text('.choice.picked .text')).toBe('nghĩa của went');
      button('Nộp')!.click();
      await settle();
      expect(text('.counter')).toBe('✓ Đúng rồi');
      expect(text('.choice.right .text')).toBe('nghĩa của went');
      expect(el.querySelector('.choice.wrong')).toBeNull();
      expect(text('.ipa')).toBe('/ɡəʊ/');
      expect(text('.meaning')).toBe('nghĩa của went');
      expect(text('.btn-next')).toBe('Tiếp ôn lại sau 10 phút');

      await goOn();
      const req = expectReview('c1');
      expect(req.request.body).toEqual({ rating: 3, mode: 'flip', reps: 0, context: 'free' });
      req.flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();
      expect(text('.word')).toBe('gave up');
      expect(el.querySelector('.meaning')).toBeNull();
      expect(el.querySelector('.choice.picked')).toBeNull();
    });

    it('marks a wrong pick and the right one, and rates it Again', async () => {
      await setup();
      await submit('nghĩa của gave up');
      expect(text('.counter')).toBe('✗ Chưa đúng');
      expect(el.querySelector('.bubble.wrong')).toBeTruthy();
      expect(text('.choice.wrong .text')).toBe('nghĩa của gave up');
      expect(text('.choice.right .text')).toBe('nghĩa của went');
      expect(text('.btn-next')).toBe('Tiếp ôn lại sau 1 phút');
      expect(el.querySelector('.btn-next.again')).toBeTruthy();

      await goOn();
      expect(expectReview('c1').request.body.rating).toBe(1);
    });

    it('shows the answer on "Quên" and rates it Again', async () => {
      await setup();
      button('Quên')!.click();
      await settle();
      expect(text('.counter')).toBe('Nghĩa của từ');
      expect(text('.choice.right .text')).toBe('nghĩa của went');
      expect(el.querySelector('.choice.wrong')).toBeNull();
      await goOn();
      expect(expectReview('c1').request.body.rating).toBe(1);
    });

    it('picks with number keys and goes on with Enter', async () => {
      await setup();
      const right = options().indexOf('nghĩa của went');
      await keydown(String(right + 1));
      await keydown('Enter');
      expect(text('.counter')).toBe('✓ Đúng rồi');
      await keydown('Enter');
      expect(expectReview('c1').request.body.rating).toBe(3);
    });

    it('sends the review context', async () => {
      await setup();
      host.context.set('daily');
      await settle();
      await submit('nghĩa của went');
      await goOn();
      const req = expectReview('c1');
      expect(req.request.body).toEqual({ rating: 3, mode: 'flip', reps: 0, context: 'daily' });
      req.flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();
    });

    it('takes the wrong meanings from the pool too', async () => {
      await setup('flip', [card('c1', 'went')]);
      host.pool.set(['ăn', 'ngủ', 'chạy', 'đọc']);
      await settle();
      expect(options()).toHaveLength(4);
      expect(options()).toContain('nghĩa của went');
    });

    it('plays the word', async () => {
      await setup();
      button('Nghe từ')!.click();
      await settle();
      expect(speech.texts()).toContain('went');
    });

    it('shows the translation of the example once answered', async () => {
      await setup('flip', [{ ...card('c1', 'went'), contextSentenceVi: 'Chúng tôi về nhà.' }, card('c2', 'gave up')]);
      expect(el.querySelector('.example-vi')).toBeNull();
      await submit('nghĩa của went');
      expect(text('.example-vi')).toBe('Chúng tôi về nhà.');
    });

    it('plays the example sentence', async () => {
      await setup();
      button('Nghe câu ví dụ')!.click();
      await settle();
      expect(speech.texts()).toContain('We went home.');
    });

    it('shows a card rated Again once more at the end and reports the summary', async () => {
      await setup();
      await submit('nghĩa của gave up');
      await goOn();
      expectReview('c1').flush({ card: { ...card('c1', 'went', 1), intervals: { ...intervals, again: 120 } } });
      await settle();

      await submit('nghĩa của gave up');
      await goOn();
      expectReview('c2').flush({ card: { ...card('c2', 'gave up', 1), intervals } });
      await settle();

      // c1 again, with the intervals of its new schedule.
      expect(text('.word')).toBe('went');
      expect(text('.progress-text')).toBe('3/3');
      button('Quên')!.click();
      await settle();
      expect(text('.btn-next')).toBe('Tiếp ôn lại sau 2 phút');
      await goOn();
      const again = expectReview('c1');
      expect(again.request.body.reps).toBe(1);
      again.flush({ card: { ...card('c1', 'went', 2), intervals } });
      await settle();

      // Not re-queued a second time.
      expect(text('.summary')).toContain('Đã ôn 2 thẻ');
      expect(text('.summary')).toContain('Quên 2');
      expect(text('.summary')).toContain('Nhớ 1');
      expect(host.summaries).toEqual([{ reviewed: 2, counts: { 1: 2, 2: 0, 3: 1, 4: 0 } }]);
    });

    it('moves on when the card was already reviewed (409)', async () => {
      await setup();
      await submit('nghĩa của went');
      await goOn();
      expectReview('c1').flush(
        { error: 'review_conflict', message: 'x', card: card('c1', 'went', 1) },
        { status: 409, statusText: 'Conflict' },
      );
      await settle();
      expect(text('.word')).toBe('gave up');
    });

    it('keeps the card and offers a retry when saving fails', async () => {
      await setup();
      await submit('nghĩa của gave up');
      await goOn();
      expectReview('c1').flush({ error: 'internal_error', message: 'x' }, { status: 500, statusText: 'Error' });
      await settle();
      expect(text('.save-error')).toContain('Chưa lưu được kết quả');
      expect(text('.word')).toBe('went');

      button('Thử lại')!.click();
      await settle();
      const retry = expectReview('c1');
      expect(retry.request.body).toEqual({ rating: 1, mode: 'flip', reps: 0, context: 'free' });
      retry.flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();
      expect(text('.word')).toBe('gave up');
      expect(el.querySelector('.save-error')).toBeNull();
    });

    describe('with nothing to choose between', () => {
      it('flips the card and rates it by hand', async () => {
        await setup('flip', [card('c1', 'went')]);
        expect(el.querySelector('.choice')).toBeNull();
        expect(el.querySelector('.meaning')).toBeNull();
        button('Lật thẻ')!.click();
        await settle();
        expect(text('.meaning')).toBe('nghĩa của went');
        expect(['Quên · 1 phút', 'Khó · 5 phút', 'Nhớ · 10 phút', 'Dễ · 8 ngày'].map((l) => !!button(l))).toEqual([
          true,
          true,
          true,
          true,
        ]);
      });

      it('flips with Space and rates with number keys', async () => {
        await setup('flip', [card('c1', 'went')]);
        await keydown(' ');
        expect(el.querySelector('.meaning')).toBeTruthy();
        await keydown('3');
        const req = expectReview('c1');
        expect(req.request.body).toEqual({ rating: 3, mode: 'flip', reps: 0, context: 'free' });
        req.flush({ card: { ...card('c1', 'went', 1), intervals } });
        await settle();
        expect(text('.summary')).toContain('Đã ôn 1 thẻ');
      });

      it('flips back to the front before rating, with the button or Space', async () => {
        await setup('flip', [card('c1', 'went')]);
        button('Lật thẻ')!.click();
        await settle();
        button('↺ Lật lại mặt trước')!.click();
        await settle();
        expect(el.querySelector('.meaning')).toBeNull();
        expect(button('Lật thẻ')).toBeTruthy();
        await keydown(' ');
        expect(el.querySelector('.meaning')).toBeTruthy();
        await keydown(' ');
        expect(el.querySelector('.meaning')).toBeNull();
      });
    });

    it('cannot flip back a typed answer in listen mode', async () => {
      await setup('listen');
      await typeAndCheck('went');
      expect(button('↺ Lật lại mặt trước')).toBeUndefined();
    });

    it('views a rated card again without rating it', async () => {
      await setup();
      expect(button('Thẻ trước')!.disabled).toBe(true);
      await submit('nghĩa của went');
      await goOn();
      expectReview('c1').flush({ card: { ...card('c1', 'went', 1), intervals } });
      await settle();

      button('Thẻ trước')!.click();
      await settle();
      expect(text('.progress-text')).toBe('1/2');
      expect(text('.word')).toBe('went');
      expect(text('.meaning')).toBe('nghĩa của went');
      expect(text('.choice.right .text')).toBe('nghĩa của went');
      expect(text('.hint')).toBe('Bạn đã chấm: Nhớ');
      expect(button('Nộp')).toBeUndefined();
      await keydown('Enter'); // no rating while viewing

      await keydown('ArrowRight');
      expect(text('.word')).toBe('gave up');
      expect(el.querySelector('.hint')).toBeNull();
      expect(button('Nộp')).toBeTruthy();
    });

    it('skips the current card to the end', async () => {
      await setup('flip', [card('c1', 'went'), card('c2', 'gave up'), card('c3', 'ran')]);
      button('Để sau, ôn lại ở cuối')!.click();
      await settle();
      expect(text('.word')).toBe('gave up');
      expect(text('.progress-text')).toBe('1/3');

      for (const [id, word] of [
        ['c2', 'gave up'],
        ['c3', 'ran'],
      ]) {
        await submit(`nghĩa của ${word}`);
        await goOn();
        expectReview(id).flush({ card: { ...card(id, word, 1), intervals } });
        await settle();
      }
      expect(text('.word')).toBe('went');
      expect(text('.progress-text')).toBe('3/3');
      // The last card left cannot be skipped.
      expect(button('Để sau, ôn lại ở cuối')).toBeUndefined();
    });
  });

  it('stops the session with ✕', async () => {
    await setup();
    button('Dừng ôn')!.click();
    await settle();
    expect(host.closed).toBe(1);
  });

  describe('listen mode', () => {
    it('plays the word automatically and hides it', async () => {
      await setup('listen');
      expect(speech.texts()).toContain('went');
      expect(el.querySelector('.word')).toBeNull();
      // The example would give the answer away.
      expect(el.querySelector('.example')).toBeNull();
      expect(button('Nghe lại từ')).toBeTruthy();
      expect(button('🐢 Nghe chậm')).toBeTruthy();
      expect(input().getAttribute('autocapitalize')).toBe('off');
      expect(input().getAttribute('spellcheck')).toBe('false');
    });

    it('asks for an answer when empty', async () => {
      await setup('listen');
      await typeAndCheck('  ');
      expect(text('.answer-error')).toBe('Hãy gõ từ bạn nghe được');
      expect(buttonStarting('Nhớ')).toBeUndefined();
    });

    it('accepts any case and shows the card', async () => {
      await setup('listen');
      await typeAndCheck('WENT');
      expect(text('.counter')).toBe('✓ Đúng rồi');
      expect(el.querySelector('.bubble.right')).toBeTruthy();
      expect(text('.word')).toBe('went');
      expect(text('.meaning')).toBe('nghĩa của went');
      expect(text('.example')).toBe('We went home.');
      expect(buttonStarting('Nhớ')).toBeTruthy();
      button('Dễ · 8 ngày')!.click();
      await settle();
      expect(expectReview('c1').request.body).toEqual({ rating: 4, mode: 'listen', reps: 0, context: 'free' });
    });

    it('says wrong and shows the right word', async () => {
      await setup('listen');
      await typeAndCheck('want');
      expect(text('.counter')).toBe('✗ Chưa đúng');
      expect(el.querySelector('.bubble.wrong')).toBeTruthy();
      expect(text('.typed')).toBe('Bạn gõ: want');
      expect(text('.word')).toBe('went');
    });

    it('plays the word slower', async () => {
      await setup('listen');
      button('🐢 Nghe chậm')!.click();
      await settle();
      expect(speech.last().rate).toBeLessThan(1);
    });

    it('lets the learner see the word when the browser cannot read it', async () => {
      await setup('listen');
      speech.last().handlers.failed!('not-allowed');
      await settle();
      expect(text('.play-error')).toContain('Chưa đọc được từ này');
      button('Hiện từ')!.click();
      await settle();
      expect(text('.word')).toBe('went');
      expect(buttonStarting('Nhớ')).toBeTruthy();
    });
  });
});

describe('exampleParts', () => {
  const card = (text: string, lemma: string, contextSentence: string) => ({ text, lemma, contextSentence });

  it('cuts the sentence around the word, ignoring case', () => {
    expect(exampleParts(card('go home', 'go home', 'Then I Go Home by bus.'))).toEqual(['Then I ', 'Go Home', ' by bus.']);
  });

  it('falls back to the lemma, then to the whole sentence', () => {
    expect(exampleParts(card('went', 'go', 'We go there.'))).toEqual(['We ', 'go', ' there.']);
    expect(exampleParts(card('went', 'go', 'Nothing here.'))).toEqual(['Nothing here.', '', '']);
  });

  it('gives null without a sentence', () => {
    expect(exampleParts(card('went', 'go', '  '))).toBeNull();
    expect(exampleParts(null)).toBeNull();
  });
});
