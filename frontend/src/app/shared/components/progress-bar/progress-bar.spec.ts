import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ProgressBar } from './progress-bar';

describe('ProgressBar', () => {
  let fixture: ComponentFixture<ProgressBar>;

  const render = async (value: number, max: number, extra: Record<string, unknown> = {}) => {
    fixture = TestBed.createComponent(ProgressBar);
    fixture.componentRef.setInput('label', 'A1 · Gia đình');
    fixture.componentRef.setInput('value', value);
    fixture.componentRef.setInput('max', max);
    for (const [k, v] of Object.entries(extra)) {
      fixture.componentRef.setInput(k, v);
    }
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  };

  it('exposes the value to screen readers and shows the count', async () => {
    const el = await render(2, 12, { unit: 'bài' });
    const bar = el.querySelector('[role="progressbar"]')!;
    expect(bar.getAttribute('aria-label')).toBe('A1 · Gia đình');
    expect(bar.getAttribute('aria-valuenow')).toBe('2');
    expect(bar.getAttribute('aria-valuemin')).toBe('0');
    expect(bar.getAttribute('aria-valuemax')).toBe('12');
    expect(bar.getAttribute('aria-valuetext')).toBe('2/12 bài');
    expect(el.querySelector('.count')!.textContent).toBe('2/12 bài');
    expect((el.querySelector('.fill') as HTMLElement).style.width).toBe(`${(2 / 12) * 100}%`);
  });

  it('shows an empty bar when max is 0', async () => {
    const el = await render(0, 0);
    expect((el.querySelector('.fill') as HTMLElement).style.width).toBe('0%');
    expect(el.querySelector('.count')!.textContent).toBe('0/0');
  });

  it('caps the fill at 100%', async () => {
    const el = await render(5, 3);
    expect((el.querySelector('.fill') as HTMLElement).style.width).toBe('100%');
  });

  it('marks its tone for the skill colour', async () => {
    const el = await render(1, 3, { tone: 'listen' });
    expect(el.getAttribute('data-tone')).toBe('listen');
  });
});
