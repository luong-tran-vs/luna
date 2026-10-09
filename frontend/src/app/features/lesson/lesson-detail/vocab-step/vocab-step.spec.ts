import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { serverUrl } from '../../../../core/api-url';
import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { highlight, placeholderImage, VocabStep } from './vocab-step';

const words: VocabItem[] = [
  {
    lemma: 'hello',
    text: 'Hello',
    meaningVi: 'xin chào',
    ipa: '/həˈləʊ/',
    sentenceIndex: 0,
    sentence: '',
    pos: 'interjection',
  },
  { lemma: 'name', text: 'name', meaningVi: 'tên', ipa: '', sentenceIndex: 0, sentence: '', pos: 'noun' },
  {
    lemma: 'meet',
    text: 'meet',
    meaningVi: 'gặp',
    ipa: '/miːt/',
    sentenceIndex: 1,
    sentence: 'Nice to meet you.',
  },
];

const examples: PracticeExample[] = [
  { lemma: 'hello', sentence: 'Hello, how are you?', meaningVi: 'Xin chào, bạn khoẻ không?' },
  { lemma: 'name', sentence: "What's your name?" },
];

describe('highlight', () => {
  it('cuts the sentence around the word, case ignored', () => {
    expect(highlight('I drink a cup of Coffee every morning.', ['coffee'])).toEqual({
      sentence: 'I drink a cup of Coffee every morning.',
      before: 'I drink a cup of ',
      hit: 'Coffee',
      after: ' every morning.',
    });
  });

  it('matches whole words only, the longest form first', () => {
    expect(highlight('She runs and ran.', ['run', 'runs']).hit).toBe('runs');
    expect(highlight('A cat in a category.', ['cat']).before).toBe('A ');
  });

  it('keeps the whole sentence when the word is not in it', () => {
    expect(highlight('Nothing here.', ['coffee'])).toEqual({
      sentence: 'Nothing here.',
      before: 'Nothing here.',
      hit: '',
      after: '',
    });
  });
});

describe('placeholderImage', () => {
  it('is the same picture for the same word', () => {
    expect(placeholderImage('coffee')).toBe(placeholderImage('Coffee'));
    expect(placeholderImage('get up')).toMatch(
      /^https:\/\/loremflickr\.com\/480\/360\/get%2Cup\?lock=\d+$/,
    );
  });
});

describe('VocabStep', () => {
  let fixture: ComponentFixture<VocabStep>;
  let el: HTMLElement;
  let played: string[];
  let done: number;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const nextWord = () => el.querySelector<HTMLButtonElement>('.next-word')!;
  const prev = () => el.querySelector<HTMLButtonElement>('.nav-btn.prev')!;
  const forward = () => el.querySelector<HTMLButtonElement>('.nav-btn.forward')!;
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const click = async (b: HTMLButtonElement) => {
    b.click();
    await fixture.whenStable();
  };

  const render = async (w: VocabItem[] = words, e: PracticeExample[] = examples) => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeSpeech(new FakeSpeech()),
      ],
    });
    fixture = TestBed.createComponent(VocabStep);
    fixture.componentRef.setInput('words', w);
    fixture.componentRef.setInput('examples', e);
    fixture.componentRef.setInput('topic', 'Daily Life');
    fixture.componentRef.setInput('lessonId', 'l1');
    http = TestBed.inject(HttpTestingController);
    played = [];
    done = 0;
    fixture.componentInstance.readAloud.subscribe((t) => played.push(t));
    fixture.componentInstance.done.subscribe(() => done++);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows the title with the topic and the word count', async () => {
    await render();
    expect(text(el.querySelector('#vocab-heading'))).toBe('1. Từ vựng – Chủ đề: Daily Life');
    expect(text(el.querySelector('.count'))).toBe('1/3');
    expect(el.querySelector('[role="progressbar"]')!.getAttribute('aria-valuetext')).toBe('1/3');
  });

  it('shows one word with its picture, IPA, meaning and example', async () => {
    await render();
    expect(el.querySelectorAll('.word').length).toBe(1);
    expect(text(el.querySelector('.lemma'))).toBe('hello');
    expect(text(el.querySelector('.ipa'))).toBe('/həˈləʊ/');
    expect(text(el.querySelector('.pos'))).toBe('thán từ');
    expect(text(el.querySelector('.meaning'))).toBe('xin chào');
    expect(el.querySelector('img')!.getAttribute('src')).toBe(placeholderImage('hello'));
    expect(text(el.querySelector('.example-label'))).toBe('Ví dụ:');
    expect(text(el.querySelector('.example-btn'))).toBe('Hello, how are you? (nghe câu)');
    expect(text(el.querySelector('.example strong'))).toBe('Hello');
    expect(text(el.querySelector('.example-vi'))).toBe('Xin chào, bạn khoẻ không?');
  });

  it('leaves out the Vietnamese line when the example has none', async () => {
    await render();
    el.querySelector<HTMLButtonElement>('.next-word')!.click();
    await fixture.whenStable();
    expect(text(el.querySelector('.example-btn'))).toContain("What's your name?");
    expect(el.querySelector('.example-vi')).toBeNull();
  });

  it("hides the previous word's picture until the next word's has loaded", async () => {
    await render();
    const img = () => el.querySelector('img')!;
    expect(img().classList.contains('loaded')).toBe(false);
    img().dispatchEvent(new Event('load'));
    await fixture.whenStable();
    expect(img().classList.contains('loaded')).toBe(true);

    el.querySelector<HTMLButtonElement>('.next-word')!.click();
    await fixture.whenStable();
    expect(img().getAttribute('src')).toBe(placeholderImage(words[1].lemma));
    expect(img().classList.contains('loaded')).toBe(false);
    img().dispatchEvent(new Event('load'));
    await fixture.whenStable();
    expect(img().classList.contains('loaded')).toBe(true);
  });

  it('draws the first letter when the picture does not load', async () => {
    await render();
    el.querySelector('img')!.dispatchEvent(new Event('error'));
    await fixture.whenStable();
    expect(el.querySelector('img')).toBeNull();
    expect(text(el.querySelector('.picture-fallback'))).toBe('H');
  });

  it('moves between the words with the arrows and Tiếp theo', async () => {
    await render();
    expect(prev().disabled).toBe(true);
    await click(nextWord());
    expect(text(el.querySelector('.lemma'))).toBe('name');
    expect(el.querySelector('.ipa')).toBeNull();
    // The part of speech shows even without an IPA.
    expect(text(el.querySelector('.word-sub'))).toBe('danh từ');
    // Part of speech without IPA.
    expect(text(el.querySelector('.word-sub'))).toBe('danh từ');
    await click(forward());
    expect(text(el.querySelector('.lemma'))).toBe('meet');
    expect(text(el.querySelector('.count'))).toBe('3/3');
    expect(el.querySelector('.pos')).toBeNull();
    expect(forward().disabled).toBe(true);
    await click(prev());
    expect(text(el.querySelector('.lemma'))).toBe('name');
    expect(done).toBe(0);
  });

  it('falls back to the sentence from the text when there is no example', async () => {
    await render();
    await click(forward());
    await click(forward());
    expect(text(el.querySelector('.example .sentence'))).toBe('Nice to meet you.');
  });

  it('Tiếp theo on the last word leaves the step, with the given label', async () => {
    await render();
    fixture.componentRef.setInput('finishLabel', 'Hoàn thành');
    await click(forward());
    await click(forward());
    expect(text(nextWord())).toBe('Hoàn thành');
    await click(nextWord());
    expect(done).toBe(1);
  });

  it('reads the word and the example aloud', async () => {
    await render();
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe từ hello"]')!);
    await click(el.querySelector<HTMLButtonElement>('button.example-btn')!);
    expect(played).toEqual(['hello', 'Hello, how are you?']);
  });

  it('uses the word picture when there is one', async () => {
    await render([{ ...words[0], imageUrl: '/api/lessons/l1/images/hello' }], examples);
    expect(el.querySelector('img')!.getAttribute('src')).toBe(serverUrl('/api/lessons/l1/images/hello'));
  });

  it('Đã học saves the word as a review card of the lesson', async () => {
    await render();
    const learn = () => el.querySelector<HTMLButtonElement>('.learn-btn');
    expect(text(learn())).toBe('Đã học – lưu vào ôn tập');
    learn()!.click();
    await fixture.whenStable();
    expect(learn()!.disabled).toBe(true);
    const req = http.expectOne('/vocab/cards/bulk');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ lessonId: 'l1', lemmas: ['hello'] });
    req.flush({ added: 1, cards: [] });
    await settle();
    expect(learn()).toBeNull();
    expect(text(el.querySelector('.learned'))).toBe('Đã lưu vào thẻ ôn tập');
    expect(text(el.querySelector('.learn [role="status"]'))).toBe('Đã lưu "hello" vào thẻ ôn tập.');

    // The next word is not saved yet; coming back keeps the first one saved.
    await click(forward());
    expect(text(learn())).toBe('Đã học – lưu vào ôn tập');
    await click(prev());
    expect(el.querySelector('.learned')).not.toBeNull();
    http.verify();
  });

  it('says when the word was already in the notebook, and when saving fails', async () => {
    await render();
    el.querySelector<HTMLButtonElement>('.learn-btn')!.click();
    http.expectOne('/vocab/cards/bulk').flush({ added: 0, cards: [] });
    await settle();
    expect(text(el.querySelector('.learn [role="status"]'))).toBe(
      '"hello" đã có trong thẻ ôn tập.',
    );

    await click(forward());
    el.querySelector<HTMLButtonElement>('.learn-btn')!.click();
    http.expectOne('/vocab/cards/bulk').flush('no', { status: 500, statusText: 'Error' });
    await settle();
    expect(text(el.querySelector('.learn [role="alert"]'))).toBe(
      'Không lưu được, vui lòng thử lại.',
    );
    expect(el.querySelector<HTMLButtonElement>('.learn-btn')!.disabled).toBe(false);
  });

  it('says when the lesson has no words', async () => {
    await render([], []);
    expect(text(el.querySelector('.empty'))).toBe('Bài này chưa có từ vựng.');
    expect(el.querySelector('.word-nav')).toBeNull();
  });
});
