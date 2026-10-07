import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeDialogue } from '../../../../core/models/practice';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { DialogueStep } from './dialogue-step';

const dialogue: PracticeDialogue = {
  speakers: ['Minh', 'Anna'],
  turns: [
    { speaker: 0, text: 'Hi, I am Minh.', meaningVi: 'Chào, mình là Minh.' },
    { speaker: 1, text: 'Hello, Minh.', meaningVi: 'Chào Minh.' },
    { speaker: 1, text: 'Nice to meet you.', meaningVi: 'Rất vui được gặp bạn.' },
  ],
};

@Component({
  imports: [DialogueStep],
  template: `<lu-dialogue-step [dialogue]="dialogue" [(rate)]="rate" (finished)="finished = finished + 1" />`,
})
class Host {
  readonly dialogue = dialogue;
  readonly rate = signal(1);
  finished = 0;
}

describe('DialogueStep', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;
  let speech: FakeSpeech;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const turn = () => el.querySelector('.turn')!;
  const reveal = () => turn().querySelector<HTMLButtonElement>('.reveal')!;
  const counter = () => text(turn().querySelector('.step-card-head .muted'));
  /** Percent of the shown turn's player track shown as played. */
  const filled = () => parseFloat(turn().querySelector<HTMLElement>('lu-audio-bar .played')!.style.width || '0');
  const isCurrent = () => turn().getAttribute('aria-current') === 'true';
  const button = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  /** A button by its text, visually hidden text included (the ← → of the card). */
  const named = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label)!;
  const stable = () => fixture.whenStable();

  const setup = async (supported = true) => {
    speech = new FakeSpeech();
    speech.supported = supported;
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(speech)] });
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await stable();
  };

  it('shows one turn at a time with its speaker and a player bar, words hidden until revealed', async () => {
    await setup();
    expect(el.querySelectorAll('.turn').length).toBe(1);
    expect(text(turn().querySelector('.speaker'))).toBe('Minh');
    expect(counter()).toBe('Lượt 1/3');
    expect(turn().querySelector('lu-audio-bar .track')!.getAttribute('aria-hidden')).toBe('true');
    expect(el.querySelectorAll('.turn-text').length).toBe(0);
    expect(reveal().getAttribute('aria-label')).toBe('Hiện lời lượt 1');
    expect(reveal().getAttribute('aria-expanded')).toBe('false');

    reveal().click();
    await stable();
    expect(text(turn().querySelector('.turn-text'))).toBe('Hi, I am Minh.');
    expect(text(turn().querySelector('.turn-meaning'))).toBe('Chào, mình là Minh.');
    expect(reveal().getAttribute('aria-expanded')).toBe('true');
    expect(reveal().getAttribute('aria-controls')).toBe('turn-lyrics-0');
    expect(text(reveal())).toBe('Ẩn lời');

    reveal().click();
    await stable();
    expect(turn().querySelector('.turn-text')).toBeNull();
  });

  it('moves between the turns with ← Lượt tiếp theo →, then Hoàn thành ends the step', async () => {
    await setup();
    expect(named('Lượt trước').disabled).toBe(true);
    named('Lượt tiếp theo').click();
    await stable();
    expect(counter()).toBe('Lượt 2/3');
    expect(text(turn().querySelector('.speaker'))).toBe('Anna');
    expect(reveal().getAttribute('aria-label')).toBe('Hiện lời lượt 2');

    named('Lượt sau').click();
    await stable();
    expect(counter()).toBe('Lượt 3/3');
    expect(named('Lượt sau').disabled).toBe(true);
    expect(fixture.componentInstance.finished).toBe(0);
    named('Hoàn thành').click();
    expect(fixture.componentInstance.finished).toBe(1);

    named('Lượt trước').click();
    await stable();
    expect(counter()).toBe('Lượt 2/3');
  });

  it('reveals and hides every turn at once', async () => {
    await setup();
    const all = el.querySelector<HTMLButtonElement>('.toggles button')!;
    expect(text(all)).toBe('Hiện lời tất cả');
    all.click();
    await stable();
    expect(text(turn().querySelector('.turn-text'))).toBe('Hi, I am Minh.');
    expect(all.getAttribute('aria-pressed')).toBe('true');
    named('Lượt tiếp theo').click();
    await stable();
    expect(text(turn().querySelector('.turn-text'))).toBe('Hello, Minh.');
    all.click();
    await stable();
    expect(el.querySelectorAll('.turn-text').length).toBe(0);
  });

  it('reads all turns in order with the elapsed time, the screen following the turn read', async () => {
    await setup();
    button('Nghe cả đoạn').click();
    await stable();
    expect(speech.last().text).toBe('Hi, I am Minh.');
    expect(isCurrent()).toBe(true);
    expect(counter()).toBe('Lượt 1/3');
    expect(text(el.querySelector('.play-info'))).toBe('00:00 Lượt 1/3');

    speech.last().handlers.progress!(0.5, 1.2);
    await stable();
    expect(filled()).toBe(50); // half read
    speech.last().handlers.progress!(1, 2.4);
    speech.last().handlers.ended!();
    await stable();
    expect(speech.last().text).toBe('Hello, Minh.');
    expect(counter()).toBe('Lượt 2/3');
    expect(isCurrent()).toBe(true);
    expect(filled()).toBe(0);

    speech.last().handlers.progress!(0.4, 1.5);
    await stable();
    expect(text(el.querySelector('.time'))).toBe('00:03');

    speech.last().handlers.ended!();
    await stable();
    speech.last().handlers.ended!();
    await stable();
    expect(isCurrent()).toBe(false);
    expect(counter()).toBe('Lượt 3/3');
    expect(speech.spoken.length).toBe(3);
    expect(button('Nghe cả đoạn')).not.toBeNull();
  });

  it('skips a turn the browser fails to read', async () => {
    await setup();
    button('Nghe cả đoạn').click();
    await stable();
    speech.last().handlers.failed!('synthesis-failed');
    await stable();
    expect(speech.last().text).toBe('Hello, Minh.');
    expect(counter()).toBe('Lượt 2/3');
    expect(isCurrent()).toBe(true);
  });

  it('stops in the middle', async () => {
    await setup();
    button('Nghe cả đoạn').click();
    await stable();
    button('Dừng').click();
    await stable();
    expect(speech.stops).toBeGreaterThan(0);
    expect(isCurrent()).toBe(false);
    expect(filled()).toBe(0);
  });

  it('reads a single turn and stops after it', async () => {
    await setup();
    named('Lượt sau').click();
    named('Lượt sau').click();
    await stable();
    button('Nghe lượt 3').click();
    await stable();
    expect(speech.last().text).toBe('Nice to meet you.');
    expect(isCurrent()).toBe(true);
    expect(button('Dừng lượt 3')).not.toBeNull();
    speech.last().handlers.ended!();
    await stable();
    expect(isCurrent()).toBe(false);
    expect(speech.spoken.length).toBe(1);
  });

  it('stops a single turn from its own player button', async () => {
    await setup();
    named('Lượt sau').click();
    await stable();
    button('Nghe lượt 2').click();
    await stable();
    speech.stops = 0;
    button('Dừng lượt 2').click();
    await stable();
    expect(speech.stops).toBeGreaterThan(0);
    expect(isCurrent()).toBe(false);
    expect(button('Nghe lượt 2')).not.toBeNull();
  });

  it('stops a single turn when the learner moves to another one', async () => {
    await setup();
    button('Nghe lượt 1').click();
    await stable();
    speech.stops = 0;
    named('Lượt tiếp theo').click();
    await stable();
    expect(speech.stops).toBeGreaterThan(0);
    expect(counter()).toBe('Lượt 2/3');
    expect(isCurrent()).toBe(false);
  });

  it('reads at the chosen speed', async () => {
    await setup();
    const options = Array.from(el.querySelectorAll<HTMLButtonElement>('[role="radio"]'));
    expect(options.map((o) => text(o))).toEqual(['0.75x', '1x', '1.25x']);
    options[0].click();
    await stable();
    expect(fixture.componentInstance.rate()).toBe(0.75);
    button('Nghe lượt 1').click();
    expect(speech.last().rate).toBe(0.75);

    options[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
    await stable();
    expect(fixture.componentInstance.rate()).toBe(1.25);
    expect(document.activeElement).toBe(options[2]);
  });

  it('hides and shows the meaning of the revealed turn', async () => {
    await setup();
    el.querySelector<HTMLButtonElement>('.toggles button')!.click();
    await stable();
    const toggle = el.querySelector<HTMLButtonElement>('.meaning-toggle')!;
    expect(toggle.getAttribute('aria-pressed')).toBe('true');
    toggle.click();
    await stable();
    expect(el.querySelectorAll('.turn-meaning').length).toBe(0);
    toggle.click();
    await stable();
    expect(el.querySelectorAll('.turn-meaning').length).toBe(1);
  });

  it('shows the words and no player when the browser has no voice', async () => {
    await setup(false);
    expect(el.textContent).toContain('Trình duyệt này không có giọng đọc');
    expect(el.querySelector('.playbar')).toBeNull();
    expect(el.querySelector('lu-audio-bar')).toBeNull();
    expect(text(turn().querySelector('.turn-text'))).toBe('Hi, I am Minh.');
    expect(el.querySelector('.reveal')).toBeNull();
  });

  it('stops reading when it goes away', async () => {
    await setup();
    button('Nghe cả đoạn').click();
    await stable();
    speech.stops = 0;
    fixture.destroy();
    expect(speech.stops).toBe(1);
  });
});
