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

  const played = () => el.querySelector<SVGElement>('svg.played')!;

  it('draws one thin bar per height, hidden from screen readers', () => {
    expect(el.querySelector('lu-waveform')!.getAttribute('aria-hidden')).toBe('true');
    expect(el.querySelector('svg')!.getAttribute('viewBox')).toBe('0 0 30 24');
    const d = el.querySelector('path.track')!.getAttribute('d')!;
    expect(d.match(/M/g)?.length).toBe(fakeWaveform('Nice to meet you.', 10).length);
    expect(d).toContain('h1v');
    expect(played().querySelector('path')!.getAttribute('d')).toBe(d);
  });

  it('fills the played part and clamps the progress, jumping back to the start', async () => {
    expect(played().style.clipPath).toBe('inset(0 100.00% 0 0)');
    expect(played().classList).toContain('restart');
    fixture.componentInstance.progress.set(0.5);
    await fixture.whenStable();
    expect(played().style.clipPath).toBe('inset(0 50.00% 0 0)');
    expect(played().classList).not.toContain('restart');
    fixture.componentInstance.progress.set(3);
    await fixture.whenStable();
    expect(played().style.clipPath).toBe('inset(0 0.00% 0 0)');
  });

  it('changes shape with the text', async () => {
    const before = el.querySelector('path.track')!.getAttribute('d');
    fixture.componentInstance.text.set('We went home, then we slept.');
    await fixture.whenStable();
    expect(el.querySelector('path.track')!.getAttribute('d')).not.toBe(before);
  });
});
