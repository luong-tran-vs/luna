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
  template: `<lu-dialogue-step [dialogue]="dialogue" [(rate)]="rate" />`,
})
class Host {
  readonly dialogue = dialogue;
  readonly rate = signal(1);
}

describe('DialogueStep', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;
  let speech: FakeSpeech;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const turns = () => Array.from(el.querySelectorAll('.turn'));
  const reveal = (i: number) => turns()[i].querySelector<HTMLButtonElement>('.reveal')!;
  const filled = (i: number) => Number(turns()[i].querySelector('lu-waveform clipPath rect')!.getAttribute('width'));
  const current = () => turns().findIndex((t) => t.getAttribute('aria-current') === 'true');
  const button = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  const stable = () => fixture.whenStable();

  const setup = async (supported = true) => {
    speech = new FakeSpeech();
    speech.supported = supported;
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(speech)] });
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement as HTMLElement;
    await stable();
  };

  it('lists each turn with its speaker and a waveform, words hidden until revealed', async () => {
    await setup();
    expect(turns().length).toBe(3);
    expect(text(turns()[0].querySelector('.speaker'))).toBe('Minh');
    expect(text(turns()[1].querySelector('.speaker'))).toBe('Anna');
    expect(turns()[0].querySelector('lu-waveform svg')!.getAttribute('aria-hidden')).toBe('true');
    expect(el.querySelectorAll('.turn-text').length).toBe(0);
    expect(reveal(0).getAttribute('aria-label')).toBe('Hiện lời lượt 1');
    expect(reveal(0).getAttribute('aria-expanded')).toBe('false');

    reveal(0).click();
    await stable();
    expect(text(turns()[0].querySelector('.turn-text'))).toBe('Hi, I am Minh.');
    expect(text(turns()[0].querySelector('.turn-meaning'))).toBe('Chào, mình là Minh.');
    expect(reveal(0).getAttribute('aria-expanded')).toBe('true');
    expect(reveal(0).getAttribute('aria-controls')).toBe('turn-lyrics-0');
    expect(text(reveal(0))).toBe('Ẩn lời');

    reveal(0).click();
    await stable();
    expect(turns()[0].querySelector('.turn-text')).toBeNull();
  });

  it('reveals and hides every turn at once', async () => {
    await setup();
    const all = el.querySelector<HTMLButtonElement>('.toggles button')!;
    expect(text(all)).toBe('Hiện lời tất cả');
    all.click();
    await stable();
    expect(el.querySelectorAll('.turn-text').length).toBe(3);
    expect(all.getAttribute('aria-pressed')).toBe('true');
    all.click();
    await stable();
    expect(el.querySelectorAll('.turn-text').length).toBe(0);
  });

  it('reads all turns in order with the elapsed time, filling each waveform', async () => {
    await setup();
    button('Nghe cả đoạn').click();
    await stable();
    expect(speech.last().text).toBe('Hi, I am Minh.');
    expect(current()).toBe(0);
    expect(text(el.querySelector('.play-info'))).toBe('00:00 Lượt 1/3');

    speech.last().handlers.progress!(0.5, 1.2);
    await stable();
    expect(filled(0)).toBe(72); // 48 bars × 3 units, half read
    speech.last().handlers.progress!(1, 2.4);
    speech.last().handlers.ended!();
    await stable();
    expect(speech.last().text).toBe('Hello, Minh.');
    expect(current()).toBe(1);
    expect(filled(0)).toBe(144);
    expect(filled(1)).toBe(0);

    speech.last().handlers.progress!(0.4, 1.5);
    await stable();
    expect(text(el.querySelector('.time'))).toBe('00:03');

    speech.last().handlers.ended!();
    await stable();
    speech.last().handlers.ended!();
    await stable();
    expect(current()).toBe(-1);
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
    expect(current()).toBe(1);
  });

  it('stops in the middle', async () => {
    await setup();
    button('Nghe cả đoạn').click();
    await stable();
    button('Dừng').click();
    await stable();
    expect(speech.stops).toBeGreaterThan(0);
    expect(current()).toBe(-1);
    expect(filled(0)).toBe(0);
  });

  it('reads a single turn and stops after it', async () => {
    await setup();
    button('Nghe lượt 3').click();
    await stable();
    expect(speech.last().text).toBe('Nice to meet you.');
    expect(current()).toBe(2);
    speech.last().handlers.ended!();
    await stable();
    expect(current()).toBe(-1);
    expect(speech.spoken.length).toBe(1);
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

  it('hides and shows the meaning of the revealed turns', async () => {
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
    expect(el.querySelectorAll('.turn-meaning').length).toBe(3);
  });

  it('shows the words and no player when the browser has no voice', async () => {
    await setup(false);
    expect(el.textContent).toContain('Trình duyệt này không có giọng đọc');
    expect(el.querySelector('.playbar')).toBeNull();
    expect(el.querySelector('lu-waveform')).toBeNull();
    expect(el.querySelectorAll('.turn-text').length).toBe(3);
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
