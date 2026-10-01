import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GrammarNote } from './grammar-note';

describe('GrammarNote', () => {
  let fixture: ComponentFixture<GrammarNote>;
  let el: HTMLElement;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [GrammarNote] }).compileComponents();
    fixture = TestBed.createComponent(GrammarNote);
    fixture.componentRef.setInput('note', {
      title: 'Thì quá khứ đơn',
      bodyVi: 'Dùng cho việc đã xong.\n\nĐộng từ thêm -ed.',
      examples: ['We went to the park.', 'He gave up smoking.'],
    });
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  it('shows the title, the body paragraphs and the examples', () => {
    expect(el.querySelector('summary')?.textContent?.trim()).toBe('Ngữ pháp: Thì quá khứ đơn');
    expect(Array.from(el.querySelectorAll('.body > p')).map((p) => p.textContent?.trim())).toEqual([
      'Dùng cho việc đã xong.',
      'Động từ thêm -ed.',
      'Ví dụ trong bài:',
    ]);
    expect(Array.from(el.querySelectorAll('.examples li')).map((li) => li.textContent?.trim())).toEqual([
      'We went to the park.',
      'He gave up smoking.',
    ]);
  });

  it('is open by default and collapses', () => {
    const details = el.querySelector('details')!;
    expect(details.open).toBe(true);
    details.open = false;
    expect(details.hasAttribute('open')).toBe(false);
  });
});
