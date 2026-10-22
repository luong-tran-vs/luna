import { TestBed } from '@angular/core/testing';

import { Mistake } from '../../../core/models/writing';
import { Corrections, markCorrections } from './corrections';

const text = 'I very like Taylor Swift song. My mother like old songs. She sing every morning.';
const corrected = 'I really like Taylor Swift songs. My mother likes old songs. She sings every morning.';
const mistakes: Mistake[] = [
  { original: 'very like', corrected: 'really like', type: 'word-order', noteVi: '"very" không đứng trước động từ' },
  { original: 'song', corrected: 'songs', type: 'plural', noteVi: 'nhiều bài hát → songs' },
  { original: 'like', corrected: 'likes', type: 'verb-s', noteVi: 'my mother = she → likes' },
  { original: 'sing', corrected: 'sings', type: 'verb-s', noteVi: 'she → sings' },
];

describe('markCorrections', () => {
  const marks = (t: string, c: string, m: Mistake[]) =>
    markCorrections(t, c, m)
      .filter((p) => p.mark !== null)
      .map((p) => [p.word, p.mark]);

  it('numbers each change after the mistake it is', () => {
    expect(marks(text, corrected, mistakes)).toEqual([
      ['really', 1],
      ['songs.', 2],
      ['likes', 3],
      ['sings', 4],
    ]);
  });

  it('gives the same mistake made twice its one number, and none to a change matching no mistake', () => {
    const m: Mistake[] = [{ original: 'like', corrected: 'likes', type: 'verb-s', noteVi: '' }];
    expect(marks('She like tea. He like milk. A apple.', 'She likes tea. He likes milk. An apple.', m)).toEqual([
      ['likes', 1],
      ['likes', 1],
    ]);
  });

  it('leaves the diff unnumbered without mistakes', () => {
    expect(markCorrections(text, corrected, []).every((p) => p.mark === null)).toBe(true);
  });
});

describe('Corrections', () => {
  const render = (m: Mistake[]) => {
    const fixture = TestBed.createComponent(Corrections);
    fixture.componentRef.setInput('text', text);
    fixture.componentRef.setInput('corrected', corrected);
    fixture.componentRef.setInput('mistakes', m);
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  };
  const clean = (n: Element | null) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  it('marks the changes in the text and lists the mistakes with their tags', () => {
    const el = render(mistakes);
    expect(Array.from(el.querySelectorAll('.diff sup')).map((s) => clean(s))).toEqual([
      '(lỗi 1)',
      '(lỗi 2)',
      '(lỗi 3)',
      '(lỗi 4)',
    ]);
    const items = Array.from(el.querySelectorAll('.mistakes li'));
    expect(items.map((li) => clean(li.querySelector('lu-mistake-tag')))).toEqual(['Trật tự từ', 'Số nhiều', 'Động từ thêm -s', 'Động từ thêm -s']);
    expect(clean(items[1])).toBe('Số nhiều, lỗi 2 nhiều bài hát → songs');
    // The text reads back as the original with the corrections in it.
    expect(clean(el.querySelector('.diff'))).toContain('Taylor Swift');
  });

  it('shows only the marked text for a grade without mistakes', () => {
    const el = render([]);
    expect(el.querySelector('.mistakes')).toBeNull();
    expect(el.querySelector('.diff sup')).toBeNull();
    expect(el.querySelectorAll('.diff del').length).toBeGreaterThan(0);
  });
});
