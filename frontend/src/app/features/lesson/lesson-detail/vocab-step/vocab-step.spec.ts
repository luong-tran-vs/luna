import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { serverUrl } from '../../../../core/api-url';
import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { highlight, VocabResult, VocabStep } from './vocab-step';

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

/** The first two words with a picture of their own. */
const pictured: VocabItem[] = [
  { ...words[0], imageUrl: '/api/lessons/l1/images/hello' },
  { ...words[1], imageUrl: '/api/lessons/l1/images/name' },
  words[2],
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

describe('VocabStep', () => {
  let fixture: ComponentFixture<VocabStep>;
  let el: HTMLElement;
  let played: string[];
  let done: number;
  let result: VocabResult | null;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const nextWord = () => el.querySelector<HTMLButtonElement>('.next-word')!;
  const prev = () => el.querySelector<HTMLButtonElement>('.nav-btn.prev')!;
  const forward = () => nextWord();
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
    result = null;
    fixture.componentInstance.readAloud.subscribe((t) => played.push(t));
    fixture.componentInstance.done.subscribe((r) => {
      done++;
      result = r;
    });
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
    await render(pictured);
    const img = () => el.querySelector('img')!;
    expect(img().classList.contains('loaded')).toBe(false);
    img().dispatchEvent(new Event('load'));
    await fixture.whenStable();
    expect(img().classList.contains('loaded')).toBe(true);

    el.querySelector<HTMLButtonElement>('.next-word')!.click();
    await fixture.whenStable();
    expect(img().getAttribute('src')).toBe(serverUrl('/api/lessons/l1/images/name'));
    expect(img().classList.contains('loaded')).toBe(false);
    img().dispatchEvent(new Event('load'));
    await fixture.whenStable();
    expect(img().classList.contains('loaded')).toBe(true);
  });

  it('drops the picture area when the picture does not load', async () => {
    await render(pictured);
    el.querySelector('img')!.dispatchEvent(new Event('error'));
    await fixture.whenStable();
    expect(el.querySelector('img')).toBeNull();
    expect(el.querySelector('.word-main')!.classList.contains('no-picture')).toBe(true);
  });

  it('moves between the words with ← and Tiếp theo, one button each way', async () => {
    await render();
    expect(prev().disabled).toBe(true);
    expect(el.querySelector('.nav-btn.forward')).toBeNull();
    await click(nextWord());
    expect(text(el.querySelector('.lemma'))).toBe('name');
    expect(el.querySelector('.ipa')).toBeNull();
    // The part of speech shows even without an IPA.
    expect(text(el.querySelector('.word-sub'))).toBe('danh từ');
    await click(nextWord());
    expect(text(el.querySelector('.lemma'))).toBe('meet');
    expect(text(el.querySelector('.count'))).toBe('3/3');
    expect(el.querySelector('.pos')).toBeNull();
    await click(prev());
    expect(text(el.querySelector('.lemma'))).toBe('name');
    expect(done).toBe(0);
  });

  it('← on the first word goes to the step before when there is one', async () => {
    await render();
    fixture.componentRef.setInput('canBack', true);
    let back = 0;
    fixture.componentInstance.back.subscribe(() => back++);
    await fixture.whenStable();
    expect(prev().disabled).toBe(false);
    await click(prev());
    expect(back).toBe(1);
  });

  it('falls back to the sentence from the text when there is no example', async () => {
    await render();
    await click(forward());
    await click(forward());
    expect(text(el.querySelector('.example .sentence'))).toBe('Nice to meet you.');
  });

  it('the last word leads to the quick recall, which ends the step', async () => {
    await render();
    fixture.componentRef.setInput('finishLabel', 'Sang bước Nghe');
    await click(forward());
    await click(forward());
    expect(text(nextWord())).toBe('Nhớ lại nhanh');
    await click(nextWord());
    expect(text(el.querySelector('#recall-heading'))).toBe('Nhớ lại nhanh');
    expect(done).toBe(0);

    // Answer every question right: the step ends with nothing missed.
    for (let i = 0; i < 3; i++) {
      const lemma = recallAnswer();
      await click(option(lemma));
      expect(text(el.querySelector('.recall-result'))).toContain('Đúng rồi!');
      if (i === 2) {
        expect(text(nextWord())).toBe('Sang bước Nghe');
      }
      await click(nextWord());
    }
    expect(done).toBe(1);
    expect(result).toEqual({ known: [], missed: [] });
  });

  it('asks a word got wrong again at the end and reports it as missed', async () => {
    await render();
    await click(forward());
    await click(forward());
    await click(nextWord());
    const lemma = recallAnswer();
    const wrong = words.map((w) => w.lemma).find((l) => l !== lemma)!;
    await click(option(wrong));
    expect(text(el.querySelector('.recall-result'))).toContain('Chưa đúng.');
    expect(text(el.querySelector('.count'))).toBe('1/4');
    await click(nextWord());
    for (let i = 0; i < 3; i++) {
      await click(option(recallAnswer()));
      await click(nextWord());
    }
    expect(result).toEqual({ known: [], missed: [lemma] });
  });

  it('Đã biết keeps a word out of the recall and reports it', async () => {
    await render();
    const known = () => el.querySelector<HTMLButtonElement>('.known-btn')!;
    expect(known().getAttribute('aria-pressed')).toBe('false');
    await click(known());
    expect(known().getAttribute('aria-pressed')).toBe('true');
    expect(text(el.querySelector('.known-note'))).toContain('không đưa vào ôn tập');
    await click(forward());
    await click(forward());
    await click(nextWord());
    expect(text(el.querySelector('.count'))).toBe('1/2');
    for (let i = 0; i < 2; i++) {
      await click(option(recallAnswer()));
      await click(nextWord());
    }
    expect(result).toEqual({ known: ['hello'], missed: [] });
  });

  it('ends at once when fewer than two words are left to recall', async () => {
    await render(words.slice(0, 1), examples);
    fixture.componentRef.setInput('finishLabel', 'Sang bước Nghe');
    await fixture.whenStable();
    expect(text(nextWord())).toBe('Sang bước Nghe');
    await click(nextWord());
    expect(done).toBe(1);
    expect(el.querySelector('#recall-heading')).toBeNull();
  });

  it('reads the word and the example aloud', async () => {
    await render();
    await click(el.querySelector<HTMLButtonElement>('button[aria-label="Nghe từ hello"]')!);
    await click(el.querySelector<HTMLButtonElement>('button.example-btn')!);
    expect(played).toEqual(['hello', 'Hello, how are you?']);
  });

  it('uses the word picture when there is one', async () => {
    await render(pictured, examples);
    expect(el.querySelector('img')!.getAttribute('src')).toBe(
      serverUrl('/api/lessons/l1/images/hello'),
    );
    expect(el.querySelector('.word-main')!.classList.contains('no-picture')).toBe(false);
  });

  it('has no picture area for a word without a picture', async () => {
    await render();
    expect(el.querySelector('img')).toBeNull();
    expect(el.querySelector('.picture')).toBeNull();
    expect(el.querySelector('.word-main')!.classList.contains('no-picture')).toBe(true);
  });

  it('saves nothing itself: the page puts the words in the review queue', async () => {
    await render();
    expect(el.querySelector('.learn-btn')).toBeNull();
    http.verify();
  });

  it('says when the lesson has no words and still goes on', async () => {
    await render([], []);
    expect(text(el.querySelector('.empty'))).toBe('Bài này chưa có từ vựng.');
    await click(nextWord());
    expect(result).toEqual({ known: [], missed: [] });
  });

  /** The lemma the question on screen asks for: by its meaning, or (listening) the word read aloud last. */
  function recallAnswer(): string {
    const meaning = text(el.querySelector('.recall-meaning'));
    if (meaning) {
      return words.find((w) => w.meaningVi === meaning)!.lemma;
    }
    return played[played.length - 1];
  }

  function option(lemma: string): HTMLButtonElement {
    return [...el.querySelectorAll<HTMLButtonElement>('.recall-option')].find((b) => text(b) === lemma)!;
  }
});
