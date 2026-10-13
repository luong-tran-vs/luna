import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ReadingLesson } from '../../../../core/models/reading';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { ReadingText } from './reading-text';

const dialogue: ReadingLesson = {
  id: 'l1',
  title: 'Coffee',
  level: 'A1',
  topic: 'Ăn uống',
  sentences: [
    { index: 0, text: 'Minh: Hi, Anna.' },
    { index: 1, text: 'Coffee?' },
    { index: 2, text: 'Anna: Yes, please.' },
    { index: 3, text: 'Minh: Here you are.' },
    { index: 4, text: 'Tom: Me too!' },
  ],
  paragraphs: [[0, 1, 2, 3, 4]],
  lemmas: {},
  phrases: [],
  quiz: null,
  grammarNote: null,
  turns: [
    { speaker: 'Minh', text: 'Hi, Anna. Coffee?', sentences: [0, 1] },
    { speaker: 'Anna', text: 'Yes, please.', sentences: [2] },
    { speaker: 'Minh', text: 'Here you are.', sentences: [3] },
    { speaker: 'Tom', text: 'Me too!', sentences: [4] },
  ],
};

describe('ReadingText', () => {
  let fixture: ComponentFixture<ReadingText>;
  let el: HTMLElement;
  let speech: FakeSpeech;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const bubbles = () => Array.from(el.querySelectorAll<HTMLButtonElement>('button.bubble'));
  const allButton = () =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => /Nghe cả đoạn|Dừng/.test(text(b)))!;
  const render = async (lesson: ReadingLesson, supported = true) => {
    speech = new FakeSpeech();
    speech.supported = supported;
    await TestBed.configureTestingModule({ imports: [ReadingText], providers: [provideFakeSpeech(speech)] }).compileComponents();
    fixture = TestBed.createComponent(ReadingText);
    fixture.componentRef.setInput('lesson', lesson);
    fixture.componentRef.setInput('rate', 0.8);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows a dialogue as bubbles with the speaker names, the second speaker on the right', async () => {
    await render(dialogue);
    expect(text(el.querySelector('h2'))).toBe('Hội thoại');
    const turns = Array.from(el.querySelectorAll('.turn'));
    expect(turns.map((t) => text(t.querySelector('.turn-name')))).toEqual(['Minh', 'Anna', 'Minh', 'Tom']);
    expect(bubbles().map((b) => text(b))).toEqual(['Hi, Anna. Coffee?', 'Yes, please.', 'Here you are.', 'Me too!']);
    expect(turns.map((t) => t.classList.contains('turn-right'))).toEqual([false, true, false, false]);
    expect(turns[3].classList).toContain('turn-alt');
    expect(el.querySelector('.reading')).toBeNull();
  });

  it('reads a tapped line in its speaker\'s voice and stops it when tapped again', async () => {
    await render(dialogue);
    bubbles()[1].click();
    expect(speech.last()).toMatchObject({ text: 'Yes, please.', rate: 0.8, voice: 1 });
    speech.playing.set('Yes, please.');
    await fixture.whenStable();
    expect(bubbles()[1].classList).toContain('reading-now');
    bubbles()[1].click();
    expect(speech.stops).toBe(1);
    bubbles()[3].click();
    expect(speech.last()).toMatchObject({ text: 'Me too!', voice: 2 });
  });

  it('plays the whole conversation line after line, and stops on demand', async () => {
    await render(dialogue);
    allButton().click();
    expect(speech.last()).toMatchObject({ text: 'Hi, Anna. Coffee?', voice: 0 });
    speech.playing.set('Hi, Anna. Coffee?');
    await fixture.whenStable();
    expect(text(allButton())).toBe('Dừng');
    expect(allButton().getAttribute('aria-pressed')).toBe('true');

    speech.last().handlers.ended!();
    expect(speech.last()).toMatchObject({ text: 'Yes, please.', voice: 1 });
    speech.last().handlers.ended!();
    expect(speech.last()).toMatchObject({ text: 'Here you are.', voice: 0 });

    allButton().click();
    expect(speech.stops).toBe(1);
    await fixture.whenStable();
    expect(text(allButton())).toBe('Nghe cả đoạn');
    // A late "ended" from the stopped line does not go on.
    const count = speech.spoken.length;
    speech.spoken[count - 1].handlers.ended!();
    expect(speech.spoken.length).toBe(count);
  });

  it('shows plain bubbles and no play button without a voice', async () => {
    await render(dialogue, false);
    expect(bubbles()).toEqual([]);
    expect(el.querySelectorAll('p.bubble').length).toBe(4);
    expect(el.textContent).not.toContain('Nghe cả đoạn');
  });

  it('shows a reading text as paragraphs of sentences', async () => {
    await render({ ...dialogue, turns: null, sentences: [{ index: 0, text: 'We went home.' }], paragraphs: [[0]] });
    expect(text(el.querySelector('h2'))).toBe('Bài đọc');
    expect(text(el.querySelector('.read-sentence'))).toBe('We went home.');
    expect(el.querySelector('.talk')).toBeNull();
  });

  // --- hold a sentence or a line for its translation; no copying ---

  const translations = ['Chào Anna.', 'Cà phê nhé?', 'Vâng, cảm ơn.', 'Của bạn đây.', 'Tôi cũng vậy!'];
  const card = () => document.querySelector<HTMLElement>('lu-sentence-translation');
  const press = (target: Element, type: string, x = 10, y = 10) =>
    target.dispatchEvent(Object.assign(new Event(type, { bubbles: true }), { button: 0, clientX: x, clientY: y }));
  const hold = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

  it('shows the translation of a line held for 2 seconds, not reading it aloud', async () => {
    await render({ ...dialogue, sentenceTranslations: translations });
    const line = bubbles()[0];
    press(line, 'pointerdown');
    await fixture.whenStable();
    expect(line.classList).toContain('pressing');
    await hold(2100);
    await fixture.whenStable();
    // A line made of two sentences: both translations.
    expect(card()?.textContent).toContain('Chào Anna. Cà phê nhé?');
    expect(card()?.textContent).toContain('Hi, Anna. Coffee?');
    expect(line.classList).toContain('translated');
    // The release and its click do not read the line aloud.
    press(line, 'pointerup');
    line.click();
    expect(speech.spoken).toEqual([]);

    card()!.querySelector<HTMLButtonElement>('button[aria-label="Đóng"]')!.click();
    await fixture.whenStable();
    expect(card()).toBeNull();
    await hold(10);
    line.click();
    expect(speech.last()).toMatchObject({ text: 'Hi, Anna. Coffee?' });
  });

  it('cancels the press when the finger moves or lifts early', async () => {
    await render({ ...dialogue, sentenceTranslations: translations });
    press(bubbles()[1], 'pointerdown');
    press(bubbles()[1], 'pointermove', 10, 40);
    await fixture.whenStable();
    expect(bubbles()[1].classList).not.toContain('pressing');
    press(bubbles()[2], 'pointerdown');
    await hold(400);
    press(bubbles()[2], 'pointerup');
    await hold(1800);
    await fixture.whenStable();
    expect(card()).toBeNull();
  });

  it('shows the translation of a focused sentence with the T key; none yet says so', async () => {
    const reading = { ...dialogue, turns: null, sentences: [{ index: 0, text: 'We went home.' }], paragraphs: [[0]] };
    await render({ ...reading, sentenceTranslations: ['Chúng tôi đã về nhà.'] });
    const sentence = el.querySelector<HTMLElement>('.read-sentence')!;
    sentence.dispatchEvent(new KeyboardEvent('keydown', { key: 't', bubbles: true }));
    await fixture.whenStable();
    expect(card()?.textContent).toContain('Chúng tôi đã về nhà.');
    card()!.querySelector('.card')!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    await fixture.whenStable();
    expect(card()).toBeNull();

    fixture.componentRef.setInput('lesson', reading);
    await fixture.whenStable();
    el.querySelector('.read-sentence')!.dispatchEvent(new KeyboardEvent('keydown', { key: 'T', bubbles: true }));
    await fixture.whenStable();
    expect(card()?.textContent).toContain('Bài này chưa có bản dịch câu.');
  });

  it('blocks copying the text and the long-press menu', async () => {
    await render(dialogue);
    for (const type of ['copy', 'contextmenu']) {
      const e = new Event(type, { bubbles: true, cancelable: true });
      bubbles()[0].dispatchEvent(e);
      expect(e.defaultPrevented).toBe(true);
    }
  });
});
