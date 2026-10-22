import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';

import { SpeakQuestion } from '../../../../core/models/practice';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { RecognizerState, SpeechRecognizerService } from '../../../../core/stt/speech-recognizer.service';
import { Recorder, RECORDING, Recording } from '../../../../core/stt/voice-recorder';

import { frameLines, markSentence, previewMarks, SpeakLesson } from './speak-lesson';

/** A recorder that finishes when the spec says so. */
class FakeRecorder implements Recorder {
  finish!: (r: Recording) => void;
  record(): Promise<Recording> {
    return new Promise((resolve) => (this.finish = resolve));
  }
  stop(): void {
    /* nothing to stop */
  }
}

const QUESTIONS: SpeakQuestion[] = [
  { question: 'What music do you like?', questionVi: 'Bạn thích nhạc gì?', frame: 'I like ___ music.' },
  { question: 'When do you listen to music?', questionVi: 'Bạn nghe nhạc khi nào?', frame: 'I listen to music ___.' },
];

describe('markSentence', () => {
  it('keeps the written words and marks each one', () => {
    expect(markSentence('She plays the piano.', 'she play the')).toEqual([
      { text: 'She', mark: 'ok' },
      { text: 'plays', mark: 'close' },
      { text: 'the', mark: 'ok' },
      { text: 'piano.', mark: 'miss' },
    ]);
  });
});

describe('frameLines', () => {
  it('splits lines around the blanks', () => {
    expect(frameLines('I like ___ music.\nMy singer is ___.')).toEqual([
      ['I like ', ' music.'],
      ['My singer is ', '.'],
    ]);
    expect(frameLines('')).toEqual([]);
  });
});

describe('SpeakLesson', () => {
  let fixture: ComponentFixture<SpeakLesson>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let recorder: FakeRecorder;
  let speech: FakeSpeech;
  const recognizer = {
    supported: true,
    state: signal<RecognizerState>('ready'),
    starting: signal(false),
    percent: signal(0),
    load: vi.fn(),
    transcribe: vi.fn<(audio: Float32Array) => Promise<string>>(async () => ''),
  };

  const text = (selector: string) =>
    el.querySelector(selector)?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
      b.textContent?.replace(/\s+/g, ' ').trim().startsWith(label),
    );
  const main = () => el.querySelector<HTMLButtonElement>('.step-nav .btn-primary')!;
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const click = async (b: HTMLElement | undefined) => {
    b!.click();
    await fixture.whenStable();
  };

  const setup = async (questions: SpeakQuestion[] = QUESTIONS, mic = true) => {
    recorder = new FakeRecorder();
    speech = new FakeSpeech();
    recognizer.transcribe.mockReset();
    recognizer.load.mockClear();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        provideFakeSpeech(speech),
        { provide: SpeechRecognizerService, useValue: recognizer },
        { provide: RECORDING, useValue: { available: () => mic, create: () => recorder } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(SpeakLesson);
    fixture.componentRef.setInput('lessonId', 'L1');
    fixture.componentRef.setInput('number', 5);
    fixture.componentRef.setInput('topicId', 'T1');
    fixture.componentRef.setInput('topicName', 'Âm nhạc');
    fixture.componentRef.setInput('readAloud', [
      { text: 'She plays the piano.', tipVi: 'plays /pleɪz/: nhớ đọc âm cuối /z/.' },
      { text: 'He sings well.', tipVi: '' },
    ]);
    fixture.componentRef.setInput('questions', questions);
    fixture.componentRef.setInput('finishLabel', 'Sang bước Viết');
    el = fixture.nativeElement;
    await fixture.whenStable();
  };

  const say = async (heard: string) => {
    recognizer.transcribe.mockResolvedValueOnce(heard);
    el.querySelector<HTMLButtonElement>('.mic')!.click();
    await fixture.whenStable();
    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
  };

  beforeEach(() => {
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:rec', revokeObjectURL: vi.fn() }));
  });

  afterEach(() => {
    http.verify();
    vi.unstubAllGlobals();
  });

  it('repeats the sentences with their tips and colours the words heard', async () => {
    await setup();
    expect(text('#speak-heading')).toBe('5. Nói · đọc theo');
    // Before reading, the word the tip is about is marked.
    expect(text('.sentence')).toBe('She plays (âm khó) the piano. (chú ý)');
    expect(text('.sentence .miss')).toBe('plays (âm khó)');
    expect(text('.tip')).toContain('âm cuối /z/');
    await click(button('Mẫu'));
    expect(speech.last().text).toBe('She plays the piano.');
    expect(button('Giọng tôi')!.disabled).toBe(true);

    await say('she plays the');
    expect(el.querySelectorAll('.sentence .w.ok')).toHaveLength(3);
    expect(el.querySelector('.sentence .w.miss')?.textContent).toContain('piano.');
    expect(button('Giọng tôi')!.disabled).toBe(false);

    expect(main().textContent?.trim()).toBe('Câu tiếp');
    await click(main());
    expect(text('.sentence')).toBe('He sings well.');
    expect(el.querySelector('.tip')).toBeNull();
    expect(main().textContent?.trim()).toBe('Sang phần trả lời');
  });

  it('without questions the step is only the repeating, then done', async () => {
    await setup([]);
    const done = vi.fn();
    fixture.componentInstance.done.subscribe(done);
    await click(main());
    expect(main().textContent?.trim()).toBe('Sang bước Viết');
    await click(main());
    expect(done).toHaveBeenCalled();
  });

  it('← goes back a sentence first, then to the step before', async () => {
    await setup();
    const back = vi.fn();
    fixture.componentInstance.back.subscribe(back);
    expect(el.querySelector('.back-step')).toBeNull();
    await click(main());
    await click(el.querySelector<HTMLButtonElement>('.back-step')!);
    expect(text('.sentence')).toBe('She plays (âm khó) the piano. (chú ý)');
    expect(back).not.toHaveBeenCalled();
  });

  it('answers, checks all answers in one request and shows the corrections', async () => {
    await setup();
    await click(main());
    await click(main());
    expect(text('#speak-heading')).toBe('5. Nói · trả lời');
    expect(text('.bubble .q')).toBe('What music do you like?');
    expect(text('.q-vi')).toBe('Bạn thích nhạc gì?');
    expect(el.querySelectorAll('.frame .blank')).toHaveLength(1);

    await say('I like pop music');
    expect(text('.said')).toContain('"I like pop music"');
    await click(main());

    // The second answer is typed.
    await click(button('Gõ'));
    const box = el.querySelector<HTMLTextAreaElement>('textarea')!;
    box.value = 'I listen music every night';
    box.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(main().textContent?.trim()).toBe('Xem nhận xét');
    await click(main());

    expect(text('#speak-heading')).toBe('Bạn đã nói 2 câu 👏');
    const req = http.expectOne('/lessons/L1/speak/check');
    expect(req.request.body).toEqual({
      answers: [
        { question: 'What music do you like?', said: 'I like pop music' },
        { question: 'When do you listen to music?', said: 'I listen music every night' },
      ],
    });
    req.flush({
      results: [
        { said: 'I like pop music', corrected: 'I like pop music', noteVi: '' },
        { said: 'I listen music every night', corrected: 'I listen to music every night.', noteVi: 'Thiếu "to".' },
      ],
    });
    await fixture.whenStable();

    const items = el.querySelectorAll('.answers li');
    expect(items[0].querySelector('.tick')).not.toBeNull();
    expect(items[1].querySelector('.was s')?.textContent).toContain('I listen music every night');
    expect(Array.from(items[1].querySelectorAll('mark.add')).map((m) => m.textContent?.trim())).toEqual([
      'to',
      'night.',
    ]);
    expect(items[1].textContent).toContain('Thiếu "to".');
    expect(text('.talk')).toContain('Nói chuyện với Leo về Âm nhạc');
    expect(main().textContent?.trim()).toBe('Sang bước Viết');

    // Back and forth with the same answers: not checked again.
    await click(el.querySelector<HTMLButtonElement>('.back-step')!);
    await click(main());
    http.expectNone('/lessons/L1/speak/check');
  });

  it('when the check fails, shows the answers without corrections and can still finish', async () => {
    await setup([QUESTIONS[0]]);
    const done = vi.fn();
    fixture.componentInstance.done.subscribe(done);
    await click(main());
    await click(main());
    await say('I like rock');
    await click(main());
    http
      .expectOne('/lessons/L1/speak/check')
      .flush({ error: 'speak_limit' }, { status: 429, statusText: 'Too Many' });
    await fixture.whenStable();
    expect(text('[role="status"]')).toContain('đủ lượt');
    expect(text('.answers li')).toContain('I like rock');
    expect(el.querySelector('.answers .tick')).toBeNull();
    await click(main());
    expect(done).toHaveBeenCalled();
  });

  it('empty answers are allowed and need no check', async () => {
    await setup([QUESTIONS[0]]);
    await click(main());
    await click(main());
    await click(main());
    expect(text('#speak-heading')).toBe('Xong phần nói');
    expect(text('.answers li')).toContain('(Chưa trả lời)');
    http.expectNone('/lessons/L1/speak/check');
  });

  it('without a microphone, the answer is typed', async () => {
    await setup([QUESTIONS[0]], false);
    expect(text('.banner-warning')).toContain('Micro chưa dùng được');
    expect(el.querySelector<HTMLButtonElement>('.mic')!.disabled).toBe(true);
    expect(recognizer.load).not.toHaveBeenCalled();
    await click(main());
    await click(main());
    expect(el.querySelector('textarea')).not.toBeNull();
  });

  it('the talk card starts a talk on the topic', async () => {
    await setup([QUESTIONS[0]]);
    const router = TestBed.inject(Router);
    const nav = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    await click(main());
    await click(main());
    await click(main());
    await click(el.querySelector<HTMLButtonElement>('.talk')!);
    const req = http.expectOne('/talk/sessions');
    expect(req.request.body).toEqual({ topicId: 'T1' });
    req.flush({ session: { id: 'S9' } });
    expect(nav).toHaveBeenCalledWith(['/vocabulary/talk', 'S9']);
  });
});

describe('previewMarks', () => {
  it('colours the word of the tip red, the longest other word yellow, the rest green', () => {
    expect(previewMarks('She plays the piano and sings.', 'sings /sɪŋz/: nhớ đọc âm cuối /z/.')).toEqual([
      { text: 'She', mark: 'ok' },
      { text: 'plays', mark: 'close' },
      { text: 'the', mark: 'ok' },
      { text: 'piano', mark: 'ok' },
      { text: 'and', mark: 'ok' },
      { text: 'sings.', mark: 'miss' },
    ]);
  });

  it('is all green without a tip', () => {
    expect(previewMarks('I like music.', '').every((w) => w.mark === 'ok')).toBe(true);
  });
});
