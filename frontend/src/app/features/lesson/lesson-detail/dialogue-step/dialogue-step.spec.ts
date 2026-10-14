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
  template: `<lu-dialogue-step
    [dialogue]="dialogue"
    [finishLabel]="finishLabel"
    [(rate)]="rate"
    (finished)="finished = finished + 1"
  />`,
})
class Host {
  readonly dialogue = dialogue;
  readonly rate = signal(1);
  finishLabel = 'Sang bước Viết';
  finished = 0;
}

describe('DialogueStep', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;
  let speech: FakeSpeech;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const bubbles = () => Array.from(el.querySelectorAll<HTMLElement>('.bubble'));
  const rows = () => Array.from(el.querySelectorAll<HTMLElement>('.chat-row'));
  const current = () => bubbles().findIndex((b) => b.getAttribute('aria-current') === 'true');
  /** Percent of the bubble being read that is filled. */
  const filled = () => parseFloat(el.querySelector<HTMLElement>('.bubble .read')?.style.width ?? '0');
  const button = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  const named = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label)!;
  const wordsSwitch = () => el.querySelector<HTMLButtonElement>('.words-switch')!;
  const meaningSwitch = () => el.querySelector<HTMLButtonElement>('.meaning-switch')!;
  const stable = () => fixture.whenStable();

  const setup = async (supported = true) => {
    speech = new FakeSpeech();
    speech.supported = supported;
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(speech)] });
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await stable();
  };

  it('shows every turn as a chat bubble, speakers on either side, words hidden until switched on', async () => {
    await setup();
    expect(bubbles().length).toBe(3);
    expect(rows().map((r) => r.classList.contains('right'))).toEqual([false, true, true]);
    expect(bubbles().map((b) => text(b))).toEqual([
      'Minh Chạm để nghe lượt 1',
      'Anna Chạm để nghe lượt 2',
      'Anna Chạm để nghe lượt 3',
    ]);
    expect(wordsSwitch().getAttribute('role')).toBe('switch');
    expect(wordsSwitch().getAttribute('aria-checked')).toBe('false');

    wordsSwitch().click();
    await stable();
    expect(wordsSwitch().getAttribute('aria-checked')).toBe('true');
    expect(text(bubbles()[0])).toBe('Minh Hi, I am Minh. Chào, mình là Minh.');
  });

  it('hides and shows the meanings', async () => {
    await setup();
    wordsSwitch().click();
    await stable();
    expect(meaningSwitch().getAttribute('aria-checked')).toBe('true');
    meaningSwitch().click();
    await stable();
    expect(el.querySelectorAll('.meaning').length).toBe(0);
    meaningSwitch().click();
    await stable();
    expect(el.querySelectorAll('.meaning').length).toBe(3);
  });

  it('reads all turns in order with the elapsed time, the bubble read marked and filled', async () => {
    await setup();
    button('Nghe cả bài').click();
    await stable();
    expect(speech.last().text).toBe('Hi, I am Minh.');
    expect(current()).toBe(0);
    expect(text(el.querySelector('.play-info'))).toBe('00:00 Lượt 1/3');

    speech.last().handlers.progress!(0.5, 1.2);
    await stable();
    expect(filled()).toBe(50);
    speech.last().handlers.progress!(1, 2.4);
    speech.last().handlers.ended!();
    await stable();
    expect(speech.last().text).toBe('Hello, Minh.');
    expect(current()).toBe(1);

    speech.last().handlers.progress!(0.4, 1.5);
    await stable();
    expect(text(el.querySelector('.time'))).toBe('00:03');

    speech.last().handlers.ended!();
    await stable();
    speech.last().handlers.ended!();
    await stable();
    expect(current()).toBe(-1);
    expect(speech.spoken.length).toBe(3);
    expect(button('Nghe cả bài')).not.toBeNull();
  });

  it('skips a turn the browser fails to read', async () => {
    await setup();
    button('Nghe cả bài').click();
    await stable();
    speech.last().handlers.failed!('synthesis-failed');
    await stable();
    expect(speech.last().text).toBe('Hello, Minh.');
    expect(current()).toBe(1);
  });

  it('stops in the middle', async () => {
    await setup();
    button('Nghe cả bài').click();
    await stable();
    button('Dừng').click();
    await stable();
    expect(speech.stops).toBeGreaterThan(0);
    expect(current()).toBe(-1);
  });

  it('reads a tapped turn alone, and stops it when tapped again', async () => {
    await setup();
    bubbles()[2].click();
    await stable();
    expect(speech.last().text).toBe('Nice to meet you.');
    expect(current()).toBe(2);
    speech.last().handlers.ended!();
    await stable();
    expect(current()).toBe(-1);
    expect(speech.spoken.length).toBe(1);

    bubbles()[1].click();
    await stable();
    speech.stops = 0;
    bubbles()[1].click();
    await stable();
    expect(speech.stops).toBeGreaterThan(0);
    expect(current()).toBe(-1);
  });

  it('cycles the speed with one button', async () => {
    await setup();
    const speed = () => el.querySelector<HTMLButtonElement>('.speed-btn')!;
    expect(text(speed())).toBe('1x');
    expect(speed().getAttribute('aria-label')).toBe('Tốc độ 1x, bấm để đổi');
    speed().click();
    await stable();
    expect(fixture.componentInstance.rate()).toBe(1.25);
    speed().click();
    await stable();
    expect(fixture.componentInstance.rate()).toBe(0.75);
    bubbles()[0].click();
    expect(speech.last().rate).toBe(0.75);
  });

  it('ends with one button named after what comes next, stopping the reading', async () => {
    await setup();
    button('Nghe cả bài').click();
    await stable();
    speech.stops = 0;
    named('Sang bước Viết').click();
    await stable();
    expect(fixture.componentInstance.finished).toBe(1);
    expect(speech.stops).toBeGreaterThan(0);
  });

  it('shows the words and no player when the browser has no voice', async () => {
    await setup(false);
    expect(el.textContent).toContain('Trình duyệt này không có giọng đọc');
    expect(el.querySelector('.playbar')).toBeNull();
    expect(el.querySelector('.words-switch')).toBeNull();
    expect(el.querySelector('button.bubble')).toBeNull();
    expect(text(bubbles()[0])).toBe('Minh Hi, I am Minh. Chào, mình là Minh.');
  });

  it('stops reading when it goes away', async () => {
    await setup();
    button('Nghe cả bài').click();
    await stable();
    speech.stops = 0;
    fixture.destroy();
    expect(speech.stops).toBe(1);
  });
});
