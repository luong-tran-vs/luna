import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PracticeDialogue } from '../../../../../core/models/practice';
import { FakeSpeech, provideFakeSpeech } from '../../../../../core/services/speech.service.testing';
import {
  RecognizerState,
  SpeechRecognizerService,
} from '../../../../../core/stt/speech-recognizer.service';
import { Recorder, RECORDING, Recording } from '../../../../../core/stt/voice-recorder';
import { compareAnswer, hintText, RoleAnswer, summarize } from './roleplay-logic';
import { Roleplay } from './roleplay';

const dialogue: PracticeDialogue = {
  speakers: ['Minh', 'Anna'],
  turns: [
    {
      speaker: 0,
      text: 'What is your favorite song?',
      meaningVi: 'Bài hát yêu thích của bạn là gì?',
    },
    { speaker: 1, text: 'I like Yellow.', meaningVi: 'Mình thích bài Yellow.' },
    { speaker: 0, text: 'Do you sing it?', meaningVi: 'Bạn có hát nó không?' },
    { speaker: 1, text: 'Yes, I do.', meaningVi: 'Có, mình có hát.' },
  ],
};

describe('roleplay logic', () => {
  it('shows the line one word at a time', () => {
    expect(hintText('I like Yellow.', 0)).toBe('___ ___ ___');
    expect(hintText('I like Yellow.', 2)).toBe('I like ___');
  });

  it('forgives spelling slips only when typed', () => {
    expect(compareAnswer('I like music.', 'I like musik', true).correctWords).toBe(3);
    expect(compareAnswer('I like music.', 'I like musik', false).correctWords).toBe(2);
  });

  it('sums up the lines: right ones, skipped ones and the words missed', () => {
    const answers = new Map<number, RoleAnswer>([
      [
        1,
        {
          said: 'I like Yellow',
          comparison: compareAnswer('I like Yellow.', 'I like Yellow', false),
          typed: false,
        },
      ],
      [5, { said: 'yes I', comparison: compareAnswer('Yes, I do.', 'yes I', false), typed: false }],
    ]);
    expect(summarize([1, 3, 5], answers)).toEqual({
      lines: 3,
      answered: 2,
      right: 1,
      missedWords: ['do'],
    });
  });
});

/** A recorder that finishes when the spec says so. */
class FakeRecorder implements Recorder {
  finish!: (r: Recording) => void;
  record(): Promise<Recording> {
    return new Promise((resolve) => (this.finish = resolve));
  }
  stop(): void {
    // nothing to stop
  }
}

describe('Roleplay', () => {
  let fixture: ComponentFixture<Roleplay>;
  let el: HTMLElement;
  let speech: FakeSpeech;
  let recorder: FakeRecorder;
  const state = signal<RecognizerState>('off');
  const recognizer = {
    supported: true,
    state,
    starting: signal(false),
    percent: signal(0),
    load: vi.fn(),
    transcribe: vi.fn<(audio: Float32Array) => Promise<string>>(async () => 'I like yellow'),
  };

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const named = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);
  const bubbles = () => Array.from(el.querySelectorAll<HTMLElement>('.bubble'));
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  /** The voice finishes the line it is reading. */
  const endLine = async () => {
    speech.last().handlers.ended?.();
    await fixture.whenStable();
  };
  const type = async (value: string) => {
    const input = el.querySelector<HTMLInputElement>('#roleplay-typed')!;
    input.value = value;
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    named('Kiểm tra')!.click();
    await fixture.whenStable();
  };

  const setup = async ({ mic = true, recognizerState = 'off' as RecognizerState } = {}) => {
    speech = new FakeSpeech();
    recorder = new FakeRecorder();
    state.set(recognizerState);
    recognizer.load.mockClear();
    TestBed.configureTestingModule({
      providers: [
        provideFakeSpeech(speech),
        { provide: SpeechRecognizerService, useValue: recognizer },
        { provide: RECORDING, useValue: { available: () => mic, create: () => recorder } },
      ],
    });
    fixture = TestBed.createComponent(Roleplay);
    fixture.componentRef.setInput('dialogue', dialogue);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('reads the other speaker, then waits for the learner with the Vietnamese of their line', async () => {
    await setup();
    expect(text(el.querySelector('.pick-title'))).toBe('Bạn muốn đóng vai ai?');
    named('Vai Anna')!.click();
    await fixture.whenStable();

    expect(speech.last().text).toBe('What is your favorite song?');
    expect(bubbles().length).toBe(1);
    await endLine();

    expect(bubbles().length).toBe(2);
    expect(text(bubbles()[1])).toBe('Bạn (Anna) Nói: “Mình thích bài Yellow.”');
    expect(text(el.querySelector('.role-bar'))).toContain('Lượt 1/2');
    // The learner's line is not read out.
    expect(speech.texts()).toEqual(['What is your favorite song?']);
  });

  it('asks before downloading the recogniser and lets the learner type instead', async () => {
    await setup();
    named('Vai Anna')!.click();
    await fixture.whenStable();
    await endLine();

    expect(el.querySelector('.mic')).toBeNull();
    expect(text(el.querySelector('.controls'))).toContain('khoảng 60MB');
    expect(recognizer.load).not.toHaveBeenCalled();
    named('Dùng micro')!.click();
    expect(recognizer.load).toHaveBeenCalled();

    await type('I like yelow');
    expect(text(bubbles()[1])).toContain('Chính xác!');
    expect(text(bubbles()[1])).toContain('Câu mẫu: I like Yellow.');
  });

  it('shows the line word by word with Gợi ý', async () => {
    await setup();
    named('Vai Anna')!.click();
    await fixture.whenStable();
    await endLine();

    el.querySelector<HTMLButtonElement>('.controls .actions .btn')!.click();
    await fixture.whenStable();
    expect(text(el.querySelector('.hint'))).toBe('I ___ ___');
  });

  it('scores a spoken line, goes on, and sums up at the end', async () => {
    await setup({ recognizerState: 'ready' });
    named('Vai Anna')!.click();
    await fixture.whenStable();
    await endLine();

    el.querySelector<HTMLButtonElement>('.mic')!.click();
    await fixture.whenStable();
    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
    expect(text(bubbles()[1])).toContain('Chính xác!');

    named('Lượt tiếp')!.click();
    await fixture.whenStable();
    expect(speech.last().text).toBe('Do you sing it?');
    await endLine();

    named('Bỏ qua')!.click();
    await fixture.whenStable();
    expect(text(el.querySelector('.summary'))).toContain('Bạn nói đúng 1/2 lượt');
    expect(text(el.querySelector('.summary'))).toContain('Bỏ qua 1 lượt.');
    expect(text(bubbles()[3])).toContain('(bỏ qua)');
  });

  it('works without a microphone or a voice: typing only, the other lines shown at once', async () => {
    await setup({ mic: false });
    speech.supported = false;
    fixture = TestBed.createComponent(Roleplay);
    fixture.componentRef.setInput('dialogue', dialogue);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();

    named('Vai Anna')!.click();
    await fixture.whenStable();
    expect(bubbles().length).toBe(2);
    expect(named('Dùng micro')).toBeUndefined();
    expect(el.querySelector('#roleplay-typed')).not.toBeNull();
  });
});
