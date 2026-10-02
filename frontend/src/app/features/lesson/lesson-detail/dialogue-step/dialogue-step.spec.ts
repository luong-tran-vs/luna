import { Component, signal, viewChild } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeDialogue } from '../../../../core/models/practice';
import { AudioPlayer } from '../../../../shared/components/audio-player/audio-player';
import { DialogueStep } from './dialogue-step';

const dialogue: PracticeDialogue = {
  speakers: ['Minh', 'Anna'],
  turns: [
    { speaker: 0, text: 'Hi, I am Minh.', meaningVi: 'Chào, mình là Minh.', audioUrl: '/a/turn/0' },
    { speaker: 1, text: 'Hello, Minh.', meaningVi: 'Chào Minh.', audioUrl: null },
    {
      speaker: 1,
      text: 'Nice to meet you.',
      meaningVi: 'Rất vui được gặp bạn.',
      audioUrl: '/a/turn/2',
    },
    {
      speaker: 0,
      text: 'Nice to meet you too.',
      meaningVi: 'Mình cũng vậy.',
      audioUrl: '/a/turn/3',
    },
  ],
};

@Component({
  imports: [AudioPlayer, DialogueStep],
  template: `<lu-audio-player #p [rate]="rate()" />
    <lu-dialogue-step [dialogue]="dialogue" [player]="p" [(rate)]="rate" />`,
})
class Host {
  readonly dialogue = dialogue;
  readonly rate = signal(1);
  readonly player = viewChild.required(AudioPlayer);
}

describe('DialogueStep', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;
  let audio: HTMLAudioElement;
  let play: ReturnType<typeof vi.spyOn>;
  let pause: ReturnType<typeof vi.spyOn>;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const src = () => audio.getAttribute('src');
  const end = async () => {
    audio.dispatchEvent(new Event('ended'));
    await settle();
  };
  const current = () =>
    Array.from(el.querySelectorAll('.turn')).findIndex(
      (t) => t.getAttribute('aria-current') === 'true',
    );

  beforeEach(async () => {
    play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined);
    pause = vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => undefined);
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    audio = el.querySelector('audio')!;
  });

  afterEach(() => {
    fixture.destroy(); // before the mocks go: destroying the player pauses it
    vi.restoreAllMocks();
  });

  it('lists each turn with its speaker, text and meaning', () => {
    const turns = Array.from(el.querySelectorAll('.turn'));
    expect(turns.length).toBe(4);
    expect(text(turns[0].querySelector('.speaker'))).toBe('Minh');
    expect(text(turns[1].querySelector('.speaker'))).toBe('Anna');
    expect(text(turns[0].querySelector('.turn-text'))).toBe('Hi, I am Minh.');
    expect(text(turns[0].querySelector('.turn-meaning'))).toBe('Chào, mình là Minh.');
    // A turn without audio has no play button.
    expect(turns[1].querySelector('button')).toBeNull();
  });

  it('plays all turns in order, skipping turns without audio', async () => {
    el.querySelector<HTMLButtonElement>('button[aria-label="Nghe cả đoạn"]')!.click();
    await settle();
    expect(src()).toBe('/a/turn/0');
    expect(current()).toBe(0);
    expect(text(el.querySelector('.play-info'))).toBe('00:00 Lượt 1/4');

    audio.currentTime = 4.2;
    audio.dispatchEvent(new Event('timeupdate'));
    await fixture.whenStable();
    expect(text(el.querySelector('.time'))).toBe('00:04');

    await end();
    expect(src()).toBe('/a/turn/2');
    expect(current()).toBe(2);
    expect(text(el.querySelector('.play-info'))).toBe('00:04 Lượt 3/4');

    audio.currentTime = 3;
    audio.dispatchEvent(new Event('timeupdate'));
    await fixture.whenStable();
    expect(text(el.querySelector('.time'))).toBe('00:07');

    await end();
    expect(src()).toBe('/a/turn/3');
    await end();
    expect(current()).toBe(-1);
    expect(el.querySelector('button[aria-label="Nghe cả đoạn"]')).not.toBeNull();
    expect(play).toHaveBeenCalledTimes(3);
  });

  it('skips a turn that fails to play', async () => {
    play.mockRejectedValueOnce(new Error('NotAllowedError'));
    el.querySelector<HTMLButtonElement>('button[aria-label="Nghe cả đoạn"]')!.click();
    await settle();
    expect(src()).toBe('/a/turn/2');
    expect(current()).toBe(2);
  });

  it('stops in the middle', async () => {
    el.querySelector<HTMLButtonElement>('button[aria-label="Nghe cả đoạn"]')!.click();
    await settle();
    pause.mockClear();
    el.querySelector<HTMLButtonElement>('button[aria-label="Dừng"]')!.click();
    await settle();
    expect(pause).toHaveBeenCalled();
    expect(current()).toBe(-1);
    await end();
    expect(play).toHaveBeenCalledTimes(1);
  });

  it('plays a single turn and stops after it', async () => {
    el.querySelector<HTMLButtonElement>('button[aria-label="Nghe lượt 3"]')!.click();
    await settle();
    expect(src()).toBe('/a/turn/2');
    expect(current()).toBe(2);
    await end();
    expect(current()).toBe(-1);
    expect(play).toHaveBeenCalledTimes(1);
  });

  it('passes the chosen speed to the player', async () => {
    const options = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
    expect(options.map((o) => text(o))).toEqual(['0.75x', '1x', '1.25x']);
    expect(options[1].getAttribute('aria-checked')).toBe('true');
    options[0].click();
    await fixture.whenStable();
    expect(fixture.componentInstance.rate()).toBe(0.75);
    expect(audio.playbackRate).toBe(0.75);
    expect(options[0].getAttribute('aria-checked')).toBe('true');

    options[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
    await fixture.whenStable();
    expect(fixture.componentInstance.rate()).toBe(1.25);
    expect(document.activeElement).toBe(options[2]);
  });

  it('hides and shows every meaning', async () => {
    const toggle = el.querySelector<HTMLButtonElement>('.meaning-toggle')!;
    expect(toggle.getAttribute('aria-pressed')).toBe('true');
    toggle.click();
    await fixture.whenStable();
    expect(toggle.getAttribute('aria-pressed')).toBe('false');
    expect(el.querySelectorAll('.turn-meaning').length).toBe(0);
    toggle.click();
    await fixture.whenStable();
    expect(el.querySelectorAll('.turn-meaning').length).toBe(4);
  });
});
