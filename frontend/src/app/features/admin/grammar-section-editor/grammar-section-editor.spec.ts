import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GrammarContent } from '../../../core/models/grammar-admin';
import { GrammarSection, GrammarSectionEditor } from './grammar-section-editor';

const content: GrammarContent = {
  objective: 'Mục tiêu.',
  explanation: ['Đoạn một.', 'Đoạn hai.'],
  usage: ['Ý một.', 'Ý hai.'],
  structures: [
    { label: 'Khẳng định', pattern: 'chủ ngữ + am / is / are', example: 'I am here.' },
    { label: 'Phủ định', pattern: "chủ ngữ + am not / isn't / aren't", example: "I'm not here." },
    { label: 'Câu hỏi', pattern: 'Am / Is / Are + chủ ngữ + …?', example: 'Are you here?' },
  ],
  examples: [
    { en: 'I am here.', vi: 'Tôi ở đây.', structure: 1, marked: '{1:I} {2:am} here.' },
    {
      en: "She isn't here.",
      vi: 'Cô ấy không ở đây.',
      structure: 2,
      marked: "{1:She} {2:isn't} here.",
    },
    { en: 'Are you here?', vi: 'Bạn ở đây à?', structure: 3, marked: '{1:Are} {2:you} here?' },
    { en: 'It is late.', vi: 'Muộn rồi.' },
  ],
  mistakes: [{ wrong: 'She are.', right: 'She is.', noteVi: 'She đi với is.' }],
  practice: [{ id: 'p1', kind: 'fill', text: 'I ___ here.', answers: ['am'], explanationVi: 'am' }],
  mastery: [
    { id: 'm1', kind: 'fill', text: 'She ___ here.', answers: ['is'], explanationVi: 'is' },
  ],
};

describe('GrammarSectionEditor', () => {
  let fixture: ComponentFixture<GrammarSectionEditor>;
  let el: HTMLElement;

  const setup = async (section: GrammarSection) => {
    TestBed.configureTestingModule({ imports: [GrammarSectionEditor] });
    fixture = TestBed.createComponent(GrammarSectionEditor);
    fixture.componentRef.setInput('section', section);
    fixture.componentRef.setInput('content', content);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };
  const set = async (selector: string, value: string) => {
    const field = el.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!;
    field.value = value;
    field.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const click = async (label: string) => {
    Array.from(el.querySelectorAll<HTMLButtonElement>('button'))
      .find((b) => b.getAttribute('aria-label') === label || b.textContent?.trim() === label)!
      .click();
    await fixture.whenStable();
  };
  const result = () => fixture.componentInstance.result();

  it('reads the theory back: paragraphs at blank lines, one usage per line', async () => {
    await setup('theory');
    await set('#sec-objective', '  Mục tiêu mới. ');
    await set('#sec-explanation', 'Đoạn một\ntiếp tục.\n\n\nĐoạn hai.');
    await set('#sec-usage', 'Ý một.\n\n  Ý hai.  \n');
    const out = result();
    expect(out.objective).toBe('Mục tiêu mới.');
    expect(out.explanation).toEqual(['Đoạn một tiếp tục.', 'Đoạn hai.']);
    expect(out.usage).toEqual(['Ý một.', 'Ý hai.']);
    expect(out.practice).toBe(content.practice);
    expect(out.structures).toBe(content.structures);
  });

  it('renumbers the examples when a structure is removed; those of the removed one become general', async () => {
    await setup('structures');
    await click('Xoá cấu trúc 2');
    const out = result();
    expect(out.structures.map((s) => s.label)).toEqual(['Khẳng định', 'Câu hỏi']);
    expect(out.examples[0]).toEqual(content.examples[0]);
    expect(out.examples[1]).toEqual({ en: "She isn't here.", vi: 'Cô ấy không ở đây.' });
    expect(out.examples[2]).toEqual({ ...content.examples[2], structure: 2 });
    expect(out.examples[3]).toEqual(content.examples[3]);
  });

  it('warns when a pattern changes its number of pieces under marked examples, and can drop the marks', async () => {
    await setup('structures');
    expect(el.querySelector('.warning')).toBeNull();
    await set('#sec-s-pattern-0', 'chủ ngữ + am / is / are + tính từ');
    expect(el.querySelector('.warning')?.textContent).toContain('Cấu trúc 1: 2 → 3 phần');
    expect(el.querySelector('.warning')?.textContent).toContain('I am here.');
    expect(result().examples[0].marked).toBe('{1:I} {2:am} here.');
    el.querySelector<HTMLInputElement>('.warning input[type=checkbox]')!.click();
    await fixture.whenStable();
    expect(result().examples[0]).toEqual({ en: 'I am here.', vi: 'Tôi ở đây.', structure: 1 });
    expect(result().examples[1].marked).toBe("{1:She} {2:isn't} here.");
  });

  it('previews the pattern as learners will see it while typing', async () => {
    await setup('structures');
    await set('#sec-s-pattern-0', 'I / he / she / it + was; you / we / they + were');
    expect(el.querySelectorAll('.row')[0].querySelectorAll('lu-pattern .line').length).toBe(2);
  });

  it('adds a structure at the end, with no examples moved', async () => {
    await setup('structures');
    await click('Thêm cấu trúc');
    await set('#sec-s-label-3', 'Viết tắt');
    await set('#sec-s-pattern-3', "I'm / he's");
    await set('#sec-s-example-3', "I'm here.");
    const out = result();
    expect(out.structures[3]).toEqual({
      label: 'Viết tắt',
      pattern: "I'm / he's",
      example: "I'm here.",
    });
    expect(out.examples).toEqual(content.examples);
  });

  it('edits the examples: marks kept only with a structure number, removal', async () => {
    await setup('examples');
    await click('Xoá câu 1');
    await set('#sec-e-marked-2', '{1:It} is late.');
    const out = result();
    expect(out.examples.length).toBe(3);
    expect(out.examples[2]).toEqual({ en: 'It is late.', vi: 'Muộn rồi.' });
    expect(out.examples[0].structure).toBe(2);
  });

  it('edits the mistakes', async () => {
    await setup('mistakes');
    await click('Thêm lỗi');
    await set('#sec-m-wrong-1', ' He are. ');
    await set('#sec-m-right-1', 'He is.');
    await set('#sec-m-note-1', 'He đi với is.');
    expect(result().mistakes[1]).toEqual({
      wrong: 'He are.',
      right: 'He is.',
      noteVi: 'He đi với is.',
    });
  });

  it('emits the content on submit unless busy', async () => {
    await setup('mistakes');
    const seen: GrammarContent[] = [];
    fixture.componentInstance.saved.subscribe((c) => seen.push(c));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    fixture.componentRef.setInput('busy', true);
    await fixture.whenStable();
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(seen.length).toBe(1);
  });
});
