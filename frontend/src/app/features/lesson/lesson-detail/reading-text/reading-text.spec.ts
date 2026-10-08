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
});
