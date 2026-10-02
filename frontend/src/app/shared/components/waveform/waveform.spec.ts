import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { fakeWaveform } from '../../utils/fake-waveform';
import { Waveform } from './waveform';

@Component({
  imports: [Waveform],
  template: `<lu-waveform [text]="text()" [progress]="progress()" [bars]="10" />`,
})
class Host {
  readonly text = signal('Nice to meet you.');
  readonly progress = signal(0);
}

describe('Waveform', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;

  beforeEach(async () => {
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  const clipWidth = () => el.querySelector('clipPath rect')!.getAttribute('width');

  it('draws one bar per height, hidden from screen readers', () => {
    const svg = el.querySelector('svg')!;
    expect(svg.getAttribute('aria-hidden')).toBe('true');
    expect(svg.getAttribute('viewBox')).toBe('0 0 30 24');
    const d = el.querySelector('path.track')!.getAttribute('d')!;
    expect(d.match(/M/g)?.length).toBe(fakeWaveform('Nice to meet you.', 10).length);
    expect(el.querySelector('path.played')!.getAttribute('d')).toBe(d);
  });

  it('fills the played part and clamps the progress', async () => {
    expect(clipWidth()).toBe('0');
    fixture.componentInstance.progress.set(0.5);
    await fixture.whenStable();
    expect(clipWidth()).toBe('15');
    fixture.componentInstance.progress.set(3);
    await fixture.whenStable();
    expect(clipWidth()).toBe('30');
  });

  it('changes shape with the text', async () => {
    const before = el.querySelector('path.track')!.getAttribute('d');
    fixture.componentInstance.text.set('We went home, then we slept.');
    await fixture.whenStable();
    expect(el.querySelector('path.track')!.getAttribute('d')).not.toBe(before);
  });
});
