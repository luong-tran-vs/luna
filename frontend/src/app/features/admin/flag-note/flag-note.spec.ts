import { ComponentFixture, TestBed } from '@angular/core/testing';

import { LessonFlag } from '../../../core/models/lesson';
import { FlagNote } from './flag-note';

describe('FlagNote', () => {
  let fixture: ComponentFixture<FlagNote>;
  const make = (over: Partial<LessonFlag>) => {
    fixture = TestBed.createComponent(FlagNote);
    fixture.componentRef.setInput('flag', { area: 'question', index: 0, kind: 'mismatch', noteVi: 'Đáp án B hợp lý hơn', confirmed: false, ...over });
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  };

  it('shows the label for each kind and the note', () => {
    const labels: Record<string, string> = {
      mismatch: 'Cần xem: AI giải ra đáp án khác',
      ambiguous: 'AI thấy câu mơ hồ',
      wrong: 'AI thấy có thể sai',
      unchecked: 'AI chưa kiểm tra được câu này',
    };
    for (const [kind, text] of Object.entries(labels)) {
      const el = make({ kind: kind as LessonFlag['kind'] });
      expect(el.textContent).toContain(text);
      expect(el.textContent).toContain('Đáp án B hợp lý hơn');
    }
  });

  it('emits confirm and then shows the reviewed label without the button', () => {
    const el = make({});
    let got: LessonFlag | null = null;
    fixture.componentInstance.confirm.subscribe((f) => (got = f));
    el.querySelector('button')!.click();
    expect(got).toMatchObject({ area: 'question', index: 0 });
    const done = make({ confirmed: true });
    expect(done.textContent).toContain('Đã xem, giữ nguyên');
    expect(done.querySelector('button')).toBeNull();
  });
});
