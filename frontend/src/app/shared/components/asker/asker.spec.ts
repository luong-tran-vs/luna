import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Asker } from './asker';

@Component({
  imports: [Asker],
  template: `<lu-asker [label]="label()" [tone]="tone()"><p class="q">What time is it?</p></lu-asker>`,
})
class Host {
  readonly label = signal('Câu 1/4');
  readonly tone = signal<'right' | 'wrong' | null>(null);
}

describe('Asker', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;

  beforeEach(async () => {
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('shows the label and the content in the bubble, the boy as decoration', () => {
    expect(el.querySelector('.bubble .counter')?.textContent?.trim()).toBe('Câu 1/4');
    expect(el.querySelector('.bubble .q')?.textContent).toBe('What time is it?');
    const boy = el.querySelector('img.boy')!;
    expect(boy.getAttribute('alt')).toBe('');
  });

  it('leaves the label out when empty, and colours the bubble once answered', async () => {
    fixture.componentInstance.label.set('');
    fixture.componentInstance.tone.set('right');
    await fixture.whenStable();
    expect(el.querySelector('.counter')).toBeNull();
    expect(el.querySelector('.bubble')!.classList).toContain('right');
    fixture.componentInstance.tone.set('wrong');
    await fixture.whenStable();
    expect(el.querySelector('.bubble')!.classList).toContain('wrong');
  });
});
