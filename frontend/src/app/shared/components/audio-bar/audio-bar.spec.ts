import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AudioBar, clock } from './audio-bar';

describe('clock', () => {
  it('shows mm:ss, rounded down', () => {
    expect(clock(0)).toBe('00:00');
    expect(clock(3.9)).toBe('00:03');
    expect(clock(72)).toBe('01:12');
    expect(clock(-1)).toBe('00:00');
  });
});

describe('AudioBar', () => {
  let fixture: ComponentFixture<AudioBar>;
  let el: HTMLElement;
  let toggled: number;

  const text = (selector: string) =>
    el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = () => el.querySelector<HTMLButtonElement>('.play')!;

  beforeEach(async () => {
    fixture = TestBed.createComponent(AudioBar);
    fixture.componentRef.setInput('duration', 72);
    toggled = 0;
    fixture.componentInstance.playPressed.subscribe(() => toggled++);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  });

  it('says the audio is being prepared, once for screen readers', async () => {
    fixture.componentRef.setInput('preparing', true);
    await fixture.whenStable();
    expect(button().classList.contains('preparing')).toBe(true);
    expect(button().getAttribute('aria-busy')).toBe('true');
    expect(text('.time')).toBe('Đang chuẩn bị…');
    expect(text('[role="status"]')).toBe('Đang chuẩn bị giọng đọc');
    fixture.componentRef.setInput('preparing', false);
    await fixture.whenStable();
    expect(text('.time')).toBe('Đã nghe 00:00 / 01:12');
    expect(text('[role="status"]')).toBe('');
  });

  it('shows the played time over the length, and the played part of the track', async () => {
    expect(text('.time')).toBe('Đã nghe 00:00 / 01:12');
    fixture.componentRef.setInput('progress', 0.5);
    await fixture.whenStable();
    expect(text('.time')).toBe('Đã nghe 00:36 / 01:12');
    expect(el.querySelector<HTMLElement>('.played')!.style.width).toBe('50%');
    expect(el.querySelector<HTMLElement>('.knob')!.style.left).toBe('50%');
    fixture.componentRef.setInput('progress', 2);
    await fixture.whenStable();
    expect(el.querySelector<HTMLElement>('.played')!.style.width).toBe('100%');
  });

  it('names the button after what it does', async () => {
    expect(button().getAttribute('aria-label')).toBe('Phát');
    fixture.componentRef.setInput('playing', true);
    await fixture.whenStable();
    expect(button().getAttribute('aria-label')).toBe('Dừng');
    button().click();
    expect(toggled).toBe(1);
  });
});
