import { provideHttpClient } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeExample } from '../../../../core/models/practice';
import { VocabItem } from '../../../../core/models/vocab';
import { VocabStep } from './vocab-step';

const words: VocabItem[] = [
  {
    lemma: 'hello',
    text: 'Hello',
    meaningVi: 'xin chào',
    ipa: '/həˈləʊ/',
    sentenceIndex: 0,
    sentence: '',
  },
  { lemma: 'name', text: 'name', meaningVi: 'tên', ipa: '', sentenceIndex: 0, sentence: '' },
  { lemma: 'meet', text: 'meet', meaningVi: 'gặp', ipa: '/miːt/', sentenceIndex: 1, sentence: '' },
];

const examples: PracticeExample[] = [
  { lemma: 'hello', sentence: 'Hello, how are you?', audioUrl: '/a/example/0' },
  { lemma: 'name', sentence: "What's your name?", audioUrl: null },
];

describe('VocabStep', () => {
  let fixture: ComponentFixture<VocabStep>;
  let el: HTMLElement;
  let played: string[];

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  const render = async (w: VocabItem[] = words, e: PracticeExample[] = examples) => {
    TestBed.configureTestingModule({ providers: [provideHttpClient()] });
    fixture = TestBed.createComponent(VocabStep);
    fixture.componentRef.setInput('words', w);
    fixture.componentRef.setInput('examples', e);
    played = [];
    fixture.componentInstance.playAudio.subscribe((url) => played.push(url));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows each word with its IPA, meaning and example', async () => {
    await render();
    const items = Array.from(el.querySelectorAll('.word'));
    expect(items.length).toBe(3);
    expect(text(items[0].querySelector('.word-text'))).toBe('hello /həˈləʊ/ — xin chào');
    expect(text(items[0].querySelector('.example'))).toBe(
      'Ví dụ: "Hello, how are you?" (nghe câu)',
    );
    expect(text(items[1].querySelector('.word-text'))).toBe('name — tên');
    expect(text(items[1].querySelector('.example'))).toBe(`Ví dụ: "What's your name?"`);
    expect(items[2].querySelector('.example')).toBeNull();
  });

  it('plays the word from the dictionary voice', async () => {
    await render();
    el.querySelector<HTMLButtonElement>('button[aria-label="Nghe từ hello"]')!.click();
    expect(played).toEqual(['/api/tts/word?text=hello']);
  });

  it('plays an example that has audio; one without audio is plain text', async () => {
    await render();
    const buttons = el.querySelectorAll<HTMLButtonElement>('button.example');
    expect(buttons.length).toBe(1);
    buttons[0].click();
    expect(played).toEqual(['/a/example/0']);
  });

  it('collapses and expands the list', async () => {
    await render();
    const toggle = el.querySelector<HTMLButtonElement>('#vocab-heading')!;
    const list = el.querySelector<HTMLElement>('#vocab-list')!;
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    expect(toggle.getAttribute('aria-controls')).toBe('vocab-list');
    toggle.click();
    await fixture.whenStable();
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(list.hidden).toBe(true);
    toggle.click();
    await fixture.whenStable();
    expect(list.hidden).toBe(false);
  });

  it('says when the lesson has no words', async () => {
    await render([], []);
    expect(text(el.querySelector('.empty'))).toBe('Bài này chưa có từ vựng.');
  });
});
