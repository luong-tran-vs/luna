import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Step, StepState } from '../../../core/models/study';
import { StepIndicator } from './step-indicator';

@Component({
  imports: [StepIndicator],
  template: `<lu-step-indicator [steps]="steps()" />`,
})
class Host {
  readonly steps = signal<Record<Step, StepState>>({ review: 'done', read: 'current', listen: 'locked', write: 'locked' });
}

describe('StepIndicator', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;
  const text = (node: Element | null) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  beforeEach(async () => {
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('shows the four steps with their state in text, not only colour', () => {
    const items = Array.from(el.querySelectorAll('li'));
    expect(items.map((i) => text(i.querySelector('.mark')))).toEqual(['✓', '●', '🔒', '🔒']);
    expect(items.map((i) => text(i.querySelector('.label')))).toEqual(['Ôn', 'Đọc', 'Nghe', 'Viết']);
    expect(items.map((i) => text(i.querySelector('.visually-hidden')))).toEqual(['(đã xong)', '(đang làm)', '(chưa mở)', '(chưa mở)']);
    expect(items.map((i) => i.dataset['state'])).toEqual(['done', 'current', 'locked', 'locked']);
    expect(items[1].getAttribute('aria-current')).toBe('step');
    expect(items[0].getAttribute('aria-current')).toBeNull();
  });

  it('shows the progress as a progress bar', async () => {
    const bar = () => el.querySelector('.progress [role="progressbar"]')!;
    expect(bar().getAttribute('aria-valuenow')).toBe('1');
    expect(bar().getAttribute('aria-valuemax')).toBe('4');
    expect(text(el.querySelector('.progress .count'))).toBe('1/4 bước');
    fixture.componentInstance.steps.set({ review: 'done', read: 'done', listen: 'done', write: 'done' });
    await fixture.whenStable();
    expect(bar().getAttribute('aria-valuenow')).toBe('4');
    expect(text(el.querySelector('.progress .count'))).toBe('4/4 bước');
  });
});
