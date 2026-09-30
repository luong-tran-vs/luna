import { Component, signal, viewChild } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AudioPlayer } from './audio-player';

@Component({
  imports: [AudioPlayer],
  template: `<lu-audio-player
    [src]="src()"
    [rate]="rate()"
    (started)="events.push('started')"
    (finished)="events.push('finished')"
    (failed)="events.push('failed')"
  />`,
})
class Host {
  readonly src = signal<string | null>('/a.wav');
  readonly rate = signal(1);
  readonly events: string[] = [];
  readonly player = viewChild.required(AudioPlayer);
}

describe('AudioPlayer', () => {
  let fixture: ComponentFixture<Host>;
  let host: Host;
  let audio: HTMLAudioElement;
  let play: ReturnType<typeof vi.spyOn>;
  let pause: ReturnType<typeof vi.spyOn>;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  beforeEach(async () => {
    play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined);
    pause = vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => undefined);
    fixture = TestBed.createComponent(Host);
    host = fixture.componentInstance;
    await fixture.whenStable();
    audio = (fixture.nativeElement as HTMLElement).querySelector('audio')!;
  });

  afterEach(() => vi.restoreAllMocks());

  it('renders a hidden audio element', () => {
    expect(audio).toBeTruthy();
    expect(audio.hidden).toBe(true);
  });

  it('play() loads the source, applies the rate and plays', async () => {
    host.rate.set(0.75);
    await fixture.whenStable();
    host.player().play();
    await settle();
    expect(audio.getAttribute('src')).toBe('/a.wav');
    expect(audio.playbackRate).toBe(0.75);
    expect(play).toHaveBeenCalledTimes(1);
    expect(host.events).toEqual(['started']);
  });

  it('changing src stops the previous audio and keeps the rate', async () => {
    host.rate.set(1.25);
    host.player().play();
    await settle();
    pause.mockClear();
    host.src.set('/b.wav');
    await fixture.whenStable();
    expect(pause).toHaveBeenCalled();
    host.player().play();
    await settle();
    expect(audio.getAttribute('src')).toBe('/b.wav');
    expect(audio.playbackRate).toBe(1.25);
  });

  it('play(src) switches immediately and the matching input change does not stop it', async () => {
    host.player().play('/c.wav');
    await settle();
    expect(audio.getAttribute('src')).toBe('/c.wav');
    pause.mockClear();
    host.src.set('/c.wav');
    await fixture.whenStable();
    expect(pause).not.toHaveBeenCalled();
  });

  it('replay() restarts from the beginning', async () => {
    host.player().play();
    await settle();
    audio.currentTime = 2;
    host.player().replay();
    await settle();
    expect(audio.currentTime).toBe(0);
    expect(play).toHaveBeenCalledTimes(2);
  });

  it('applies a rate change immediately', async () => {
    host.player().play();
    await settle();
    host.rate.set(0.5);
    await fixture.whenStable();
    expect(audio.playbackRate).toBe(0.5);
  });

  it('stop() pauses', () => {
    host.player().stop();
    expect(pause).toHaveBeenCalled();
  });

  it('emits failed when playing is rejected', async () => {
    play.mockRejectedValueOnce(new Error('NotAllowedError'));
    host.player().play();
    await settle();
    expect(host.events).toEqual(['failed']);
  });

  it('does nothing without a source', async () => {
    host.src.set(null);
    await fixture.whenStable();
    host.player().play();
    await settle();
    expect(play).not.toHaveBeenCalled();
  });

  it('emits finished from the media event', () => {
    audio.dispatchEvent(new Event('ended'));
    expect(host.events).toEqual(['finished']);
  });
});
