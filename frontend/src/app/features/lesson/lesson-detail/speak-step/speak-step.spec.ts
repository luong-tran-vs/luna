import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { RecognizerState, SpeechRecognizerService } from '../../../../core/stt/speech-recognizer.service';
import { Recorder, RECORDING, Recording } from '../../../../core/stt/voice-recorder';

import { recordError, SpeakStep } from './speak-step';

/** A recorder that finishes when the spec says so. */
class FakeRecorder implements Recorder {
  finish!: (r: Recording) => void;
  fail!: (e: unknown) => void;
  stopped = 0;
  record(onTick: (level: number, seconds: number) => void): Promise<Recording> {
    onTick(0.05, 1);
    return new Promise((resolve, reject) => {
      this.finish = resolve;
      this.fail = reject;
    });
  }
  stop(): void {
    this.stopped++;
  }
}

describe('recordError', () => {
  it('explains a refused or missing microphone', () => {
    expect(recordError(new DOMException('x', 'NotAllowedError'))).toContain('Chưa cho phép dùng micro');
    expect(recordError(new DOMException('x', 'NotFoundError'))).toBe('Không tìm thấy micro trên máy này.');
    expect(recordError(new Error('x'))).toBe('Chưa chấm được lần đọc này. Hãy thử lại.');
  });
});

describe('SpeakStep', () => {
  let fixture: ComponentFixture<SpeakStep>;
  let el: HTMLElement;
  let recorder: FakeRecorder;
  let speech: FakeSpeech;
  const state = signal<RecognizerState>('ready');
  const recognizer = {
    supported: true,
    state,
    starting: signal(false),
    percent: signal(30),
    load: vi.fn(),
    transcribe: vi.fn<(audio: Float32Array) => Promise<string>>(async () => 'nice to meet you'),
  };

  const text = (selector: string) =>
    el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === label);
  const mic = () => el.querySelector<HTMLButtonElement>('.mic')!;
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };

  const setup = async (available = true) => {
    recorder = new FakeRecorder();
    speech = new FakeSpeech();
    state.set('ready');
    recognizer.load.mockClear();
    recognizer.transcribe.mockClear();
    TestBed.configureTestingModule({
      providers: [
        provideFakeSpeech(speech),
        { provide: SpeechRecognizerService, useValue: recognizer },
        { provide: RECORDING, useValue: { available: () => available, create: () => recorder } },
      ],
    });
    fixture = TestBed.createComponent(SpeakStep);
    fixture.componentRef.setInput('number', 4);
    fixture.componentRef.setInput('sentences', ['Nice to meet you!', 'Where do you live?']);
    el = fixture.nativeElement;
    await fixture.whenStable();
  };

  const say = async () => {
    mic().click();
    await fixture.whenStable();
    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
  };

  beforeEach(() => {
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:rec', revokeObjectURL: vi.fn() }));
  });

  afterEach(() => vi.unstubAllGlobals());

  it('shows the sentence to repeat, plays the sample and loads the recognizer', async () => {
    await setup();
    expect(text('#speak-heading')).toBe('4. Luyện nói (Speaking)');
    expect(text('.sentence')).toBe('Nice to meet you!');
    expect(text('.status')).toBe('Bấm micro rồi đọc theo câu trên');
    expect(recognizer.load).toHaveBeenCalled();
    el.querySelector<HTMLButtonElement>('[aria-label="Nghe câu mẫu"]')!.click();
    expect(speech.last().text).toBe('Nice to meet you!');
  });

  it('records, says it is recording, then shows how the sentence was read', async () => {
    await setup();
    mic().click();
    await fixture.whenStable();
    expect(mic().getAttribute('aria-label')).toBe('Dừng ghi âm');
    expect(text('.status')).toContain('Đang ghi âm...');

    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
    expect(recognizer.transcribe).toHaveBeenCalled();
    expect(text('.verdict-label')).toContain('Chính xác!');
    expect(text('.score')).toBe('4/4 từ đúng');
    expect(text('.heard')).toContain('"nice to meet you"');
    expect(text('.tried')).toBe('Đã đọc 1/2 câu');
  });

  it('marks misheard and missing words, and moves on with Câu tiếp theo', async () => {
    await setup();
    recognizer.transcribe.mockResolvedValueOnce('nice to eat');
    await say();
    expect(text('.verdict-label')).toContain('Gần đúng');
    expect(el.querySelectorAll('.word.ok')).toHaveLength(2);
    expect(el.querySelector('.word.wrong .said')?.textContent).toBe('eat');
    expect(el.querySelectorAll('.word.missing')).toHaveLength(1);

    button('Câu tiếp theo')!.click();
    await fixture.whenStable();
    expect(text('.sentence')).toBe('Where do you live?');
    expect(el.querySelector('.result')).toBeNull();
  });

  it('stops the recording early when the microphone is pressed again', async () => {
    await setup();
    mic().click();
    await fixture.whenStable();
    mic().click();
    expect(recorder.stopped).toBe(1);
  });

  it('explains a refused microphone', async () => {
    await setup();
    mic().click();
    await fixture.whenStable();
    recorder.fail(new DOMException('x', 'NotAllowedError'));
    await settle();
    expect(text('[role="alert"]')).toContain('Chưa cho phép dùng micro');
    expect(mic().disabled).toBe(false);
  });

  it('waits for the recognizer to load before recording', async () => {
    await setup();
    state.set('loading');
    await fixture.whenStable();
    expect(mic().disabled).toBe(true);
    expect(text('.status')).toBe('Đang tải bộ chấm phát âm (một lần, khoảng 60MB)... 30%');
  });

  it('without a usable microphone, explains HTTPS and keeps the sample', async () => {
    await setup(false);
    expect(text('.banner-warning')).toContain('chỉ cho dùng micro khi app mở qua HTTPS');
    expect(mic().disabled).toBe(true);
    expect(recognizer.load).not.toHaveBeenCalled();
    expect(text('.sentence')).toBe('Nice to meet you!');
  });

  it('Làm lại bước này forgets every try and goes back to the first sentence', async () => {
    await setup();
    expect(button('Làm lại bước này')).toBeUndefined();
    await say();
    button('Câu tiếp theo')!.click();
    await fixture.whenStable();
    expect(text('.sentence')).toBe('Where do you live?');

    button('Làm lại bước này')!.click();
    await fixture.whenStable();
    expect(text('.tried')).toBe('Đã đọc 0/2 câu');
    expect(text('.sentence')).toBe('Nice to meet you!');
    expect(el.querySelector('.result')).toBeNull();
    expect(button('Làm lại bước này')).toBeUndefined();
  });
});
