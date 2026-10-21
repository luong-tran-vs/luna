import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';
import { BehaviorSubject, of, Subject, throwError } from 'rxjs';

import { ApiError } from '../../../../core/interceptors/error-interceptor';
import { TalkSession } from '../../../../core/models/talk';
import { FakeSpeech, provideFakeSpeech } from '../../../../core/services/speech.service.testing';
import { TalkApiService } from '../../../../core/services/talk-api.service';
import {
  RecognizerState,
  SpeechRecognizerService,
} from '../../../../core/stt/speech-recognizer.service';
import { Recorder, RECORDING, Recording } from '../../../../core/stt/voice-recorder';
import { TalkSessionPage } from './talk-session';

const open: TalkSession = {
  id: 's1',
  topicId: 'music',
  topicName: 'Âm nhạc',
  level: 'A1',
  scenarioVi: 'Bạn gặp Sam ở buổi hoà nhạc.',
  turns: [
    {
      role: 'ai',
      text: 'Hi! Do you like this band?',
      textVi: 'Chào! Bạn thích ban nhạc này không?',
      reaction: 'happy',
      at: '2026-10-20T09:00:00Z',
    },
  ],
  hints: [
    { text: 'Yes, I love them.', vi: 'Có, mình rất thích.' },
    { text: 'Not really.', vi: 'Không hẳn.' },
  ],
  status: 'open',
  maxTurns: 20,
  createdAt: '2026-10-20T09:00:00Z',
};

/** The session after one more exchange. */
const answered = (said: string, spoken = false): TalkSession => ({
  ...open,
  turns: [
    ...open.turns,
    { role: 'learner', text: said, spoken, at: '2026-10-20T09:01:00Z' },
    {
      role: 'ai',
      text: 'Great! Who is your favorite singer?',
      reaction: 'laugh',
      at: '2026-10-20T09:01:00Z',
    },
  ],
  hints: [{ text: 'I like Adele.', vi: 'Mình thích Adele.' }],
});

class FakeRecorder implements Recorder {
  finish!: (r: Recording) => void;
  tick!: (level: number, seconds: number) => void;
  partial?: (audio: Float32Array) => void;
  record(
    onTick: (level: number, seconds: number) => void,
    onPartial?: (audio: Float32Array) => void,
  ): Promise<Recording> {
    this.tick = onTick;
    this.partial = onPartial;
    return new Promise((resolve) => (this.finish = resolve));
  }
  stop(): void {
    // nothing to stop
  }
}

describe('TalkSessionPage', () => {
  let fixture: ComponentFixture<TalkSessionPage>;
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
    transcribe: vi.fn(async () => 'yes I like them'),
  };
  const api = {
    get: vi.fn(),
    say: vi.fn(),
    end: vi.fn(),
    start: vi.fn(),
  };

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const named = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find(
      (b) => text(b) === label || b.getAttribute('aria-label') === label,
    );
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const click = async (label: string) => {
    const b = named(label);
    expect(b, label).toBeTruthy();
    b!.click();
    await settle();
  };
  const box = () => el.querySelector<HTMLTextAreaElement>('#talk-text');
  const typeLine = async (value: string) => {
    box()!.value = value;
    box()!.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const startRecording = async () => {
    el.querySelector<HTMLButtonElement>('.talk-btn')!.click();
    await settle();
  };
  const record = async () => {
    await startRecording();
    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
  };

  const setup = async (
    session: TalkSession = open,
    recognizerState: RecognizerState = 'ready',
    stored: Record<string, string> = {},
  ) => {
    localStorage.clear();
    Object.entries(stored).forEach(([k, v]) => localStorage.setItem(k, v));
    speech = new FakeSpeech();
    recorder = new FakeRecorder();
    state.set(recognizerState);
    Object.values(api).forEach((f) => f.mockReset());
    api.get.mockReturnValue(of(session));
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideFakeSpeech(speech),
        { provide: TalkApiService, useValue: api },
        { provide: SpeechRecognizerService, useValue: recognizer },
        { provide: RECORDING, useValue: { available: () => true, create: () => recorder } },
        {
          provide: ActivatedRoute,
          useValue: { paramMap: new BehaviorSubject(convertToParamMap({ id: 's1' })) },
        },
      ],
    });
    fixture = TestBed.createComponent(TalkSessionPage);
    el = fixture.nativeElement as HTMLElement;
    await settle();
  };

  it('opens on the situation and reads the line aloud, the Vietnamese on demand', async () => {
    await setup();
    expect(api.get).toHaveBeenCalledWith('s1');
    expect(text(el.querySelector('.who'))).toContain('0/20 · Âm nhạc');
    expect(text(el.querySelector('.scenario'))).toContain('Bạn gặp Sam ở buổi hoà nhạc.');
    expect(el.querySelector('.turns')!.getAttribute('aria-valuenow')).toBe('0');
    expect(speech.last().text).toBe('Hi! Do you like this band?');
    expect(text(el.querySelector('.bubble'))).not.toContain('Chào!');

    await click('Nghĩa');
    expect(text(el.querySelector('.bubble'))).toContain('Chào! Bạn thích ban nhạc này không?');

    await click('Chậm');
    expect(speech.last()).toMatchObject({ text: 'Hi! Do you like this band?', rate: 0.7 });

    await click('Ẩn tình huống');
    expect(el.querySelector('.scenario')).toBeNull();
    el.querySelector<HTMLButtonElement>('.who')!.click();
    await settle();
    expect(el.querySelector('.scenario')).not.toBeNull();
  });

  it('lights the words of the line as they are read, and Leo talks meanwhile', async () => {
    await setup();
    expect(el.querySelector('.leo')!.getAttribute('data-pose')).toBe('talking');
    speech.last().handlers.progress?.(0.4, 1);
    await settle();
    const said = Array.from(el.querySelectorAll('.line-text .said')).map((w) => text(w));
    expect(said).toEqual(['Hi!', 'Do', 'you']);
    expect(el.querySelectorAll('.line-text .ahead').length).toBe(3);

    speech.last().handlers.ended?.();
    await settle();
    expect(el.querySelectorAll('.line-text .said, .line-text .ahead').length).toBe(0);
    expect(el.querySelector('.leo')!.getAttribute('data-pose')).toBe('idle');
  });

  it('keeps the answers behind a pill, and reads a tapped one aloud to say, not send', async () => {
    await setup();
    expect(el.querySelector('.chip')).toBeNull();
    expect(text(el.querySelector('.hints-ask'))).toContain('Hiện sẵn gợi ý trả lời mỗi lượt?');
    await click('Để tôi tự xem');
    expect(localStorage.getItem('lu.talk.hints')).toBe('on-tap');
    expect(el.querySelector('.hints-ask')).toBeNull();
    await click('Chưa biết nói gì? Xem gợi ý');
    const chips = el.querySelectorAll<HTMLButtonElement>('.chip');
    expect(chips.length).toBe(2);
    expect(text(chips[0])).toContain('Có, mình rất thích.');

    chips[1].click();
    await settle();
    expect(speech.last().text).toBe('Not really.');
    expect(chips[1].getAttribute('aria-pressed')).toBe('true');
    expect(text(el.querySelector('.say-this'))).toBe('“Not really.”');
    expect(el.querySelector('.talk-btn')!.classList).toContain('pulse');
    expect(api.say).not.toHaveBeenCalled();
  });

  it('shows the answers by themselves each turn once the learner says so', async () => {
    await setup();
    await click('Có, hiện sẵn');
    expect(localStorage.getItem('lu.talk.hints')).toBe('always');
    expect(el.querySelectorAll('.chip').length).toBe(2);

    await record();
    api.say.mockReturnValue(of(answered('yes I like them', true)));
    await click('Gửi');
    expect(text(el.querySelector('.chip'))).toContain('I like Adele.');

    await click('Không hiện sẵn nữa');
    expect(localStorage.getItem('lu.talk.hints')).toBe('on-tap');
    expect(el.querySelector('.chip')).toBeNull();
  });

  it('opens with the answers shown when that was chosen before', async () => {
    await setup(open, 'ready', { 'lu.talk.hints': 'always' });
    expect(el.querySelector('.hints-ask')).toBeNull();
    expect(el.querySelectorAll('.chip').length).toBe(2);
  });

  it('listens with Leo leaning in, then shows what was heard in the learner bubble', async () => {
    await setup();
    await startRecording();
    expect(el.querySelector('.leo')!.getAttribute('data-pose')).toBe('listening');
    expect(text(el.querySelector('.listening'))).toBe('Leo đang lắng nghe');
    expect(text(el.querySelector('.me.waiting'))).toContain('Bạn đang nói');
    recorder.tick(0.1, 3);
    await settle();
    const heights = () =>
      Array.from(el.querySelectorAll<HTMLElement>('.talk-btn .bars i')).map((b) =>
        parseInt(b.style.height, 10),
      );
    expect(heights().length).toBe(22);
    // The voice shows as it is; the silence before it breathes between 4 and 7px.
    expect(heights()[21]).toBe(20);
    const silent = heights().slice(0, 21);
    expect(silent.every((h) => h >= 4 && h <= 7)).toBe(true);
    expect(new Set(silent).size).toBeGreaterThan(1);
    recorder.tick(0, 3.3);
    await settle();
    expect(heights().slice(0, 20)).not.toEqual(silent.slice(1));
    expect(text(el.querySelector('.talk-btn b'))).toBe('0:03');
    expect(named('Gõ câu trả lời')).toBeUndefined();

    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
    expect(text(el.querySelector('.heard-text'))).toBe('yes I like them');
    expect(el.querySelector<HTMLButtonElement>('.talk-btn')!.disabled).toBe(true);
  });

  it('shows the words while they are said, the text sent read from the whole recording', async () => {
    await setup();
    let answer!: (text: string) => void;
    recognizer.transcribe.mockClear();
    await startRecording();
    recorder.tick(0.005, 1);
    recorder.partial!(new Float32Array(16000));
    await settle();
    expect(recognizer.transcribe).not.toHaveBeenCalled(); // silence only: nothing to read yet

    recognizer.transcribe.mockImplementationOnce(() => new Promise((r) => (answer = r)));
    recorder.tick(0.1, 2);
    recorder.partial!(new Float32Array(32000));
    recorder.partial!(new Float32Array(48000)); // the first is still being read: skipped
    expect(recognizer.transcribe).toHaveBeenCalledTimes(1);
    answer(' yes I ');
    await settle();
    expect(text(el.querySelector('.me.waiting .live-text'))).toBe('yes I');

    recorder.finish({ blob: new Blob(), audio: new Float32Array(64000), seconds: 4 });
    await settle();
    expect(recognizer.transcribe).toHaveBeenCalledTimes(2);
    expect(text(el.querySelector('.heard-text'))).toBe('yes I like them');
    expect(el.querySelector('.live-text')).toBeNull();
  });

  it('drops what was being said with ✕', async () => {
    await setup();
    recognizer.transcribe.mockClear();
    await startRecording();
    await click('Huỷ câu đang nói');
    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
    expect(recognizer.transcribe).not.toHaveBeenCalled();
    expect(el.querySelector('.heard-text')).toBeNull();
    expect(text(el.querySelector('.talk-btn'))).toBe('Bấm để nói');
  });

  it('sends what was heard as spoken; Leo thinks, then reacts and reads his answer', async () => {
    await setup();
    await record();
    const reply = new Subject<TalkSession>();
    api.say.mockReturnValue(reply);
    await click('Gửi');
    expect(api.say).toHaveBeenCalledWith('s1', 'yes I like them', true);
    expect(text(el.querySelector('.me.sent'))).toContain('yes I like them');
    expect(el.querySelector('.bubble')).toBeNull();
    expect(el.querySelector('.think')).not.toBeNull();
    expect(el.querySelector('.leo')!.getAttribute('data-pose')).toBe('thinking');

    reply.next(answered('yes I like them', true));
    await settle();
    expect(el.querySelector('.me')).toBeNull();
    expect(text(el.querySelector('.face'))).toBe('😄');
    expect(text(el.querySelector('.line-text'))).toBe('Great! Who is your favorite singer?');
    expect(speech.last().text).toBe('Great! Who is your favorite singer?');
    expect(text(el.querySelector('.turns'))).toContain('1/20');
    expect(el.querySelector('.scenario')).toBeNull();
  });

  it('lets a misheard line be fixed on the keyboard, still sent as spoken', async () => {
    await setup();
    await record();
    await click('Sửa');
    expect(box()!.value).toBe('yes I like them');
    await typeLine('yes I like them a lot');
    api.say.mockReturnValue(of(answered('yes I like them a lot', true)));
    await click('Gửi');
    expect(api.say).toHaveBeenCalledWith('s1', 'yes I like them a lot', true);
  });

  it('types an answer, a tapped answer filling the box, and gets it back when the AI fails', async () => {
    await setup();
    await click('Gõ câu trả lời');
    expect(el.querySelector('.talk')!.classList).toContain('typing');
    expect(el.querySelector('.leo')).not.toBeNull();

    el.querySelectorAll<HTMLButtonElement>('.chip')[0].click();
    await settle();
    expect(box()!.value).toBe('Yes, I love them.');
    expect(speech.texts()).not.toContain('Yes, I love them.');

    api.say.mockReturnValue(
      throwError(
        () =>
          new ApiError('http', 429, {
            error: 'ai_quota',
            message: 'Đã hết lượt AI, vui lòng thử lại sau.',
          }),
      ),
    );
    await typeLine('Yes');
    await click('Gửi');
    expect(api.say).toHaveBeenCalledWith('s1', 'Yes', false);
    expect(text(el.querySelector('[role="alert"]'))).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
    expect(box()!.value).toBe('Yes');

    await click('Nói thay vì gõ');
    expect(el.querySelector('.talk-btn')).not.toBeNull();
  });

  it('asks before downloading the recogniser', async () => {
    await setup(open, 'off');
    expect(text(el.querySelector('.mic-label'))).toContain('khoảng 60MB');
    await click('Dùng micro để nói');
    expect(recognizer.load).toHaveBeenCalled();
  });

  it('stops reading Leo aloud once Tự đọc is off, and remembers it', async () => {
    await setup();
    await click('Tự đọc câu của Leo');
    expect(localStorage.getItem('lu.talk.autoRead')).toBe('off');
    const read = speech.spoken.length;

    await record();
    api.say.mockReturnValue(of(answered('yes I like them', true)));
    await click('Gửi');
    expect(speech.spoken.length).toBe(read);
  });

  it('shows the whole conversation in a dialog', async () => {
    await setup(answered('Yes I do', true));
    await click('Xem lại hội thoại');
    const dialog = el.querySelector('[role="dialog"]')!;
    expect(text(dialog)).toContain('Bạn gặp Sam ở buổi hoà nhạc.');
    expect(text(dialog)).toContain('Leo: Hi! Do you like this band?');
    expect(text(dialog)).toContain('Bạn: Yes I do nói');
    await click('Đóng hội thoại');
    expect(el.querySelector('[role="dialog"]')).toBeNull();
  });

  it('asks before hanging up early, then shows the review and offers another conversation', async () => {
    await setup(answered('I very like'));
    await click('Kết thúc cuộc trò chuyện');
    expect(text(el.querySelector('[role="alertdialog"]'))).toContain('Bạn mới nói 1/20 lượt');
    await click('Nói tiếp');
    expect(el.querySelector('[role="alertdialog"]')).toBeNull();
    expect(api.end).not.toHaveBeenCalled();

    api.end.mockReturnValue(
      of({
        ...answered('I very like'),
        status: 'ended',
        hints: [],
        review: {
          overallVi: 'Bạn nói khá tốt.',
          fixes: [
            { said: 'I very like', better: 'I really like it.', noteVi: 'Không nói very like.' },
          ],
          good: [{ said: 'Yes I love them', noteVi: 'Tự nhiên.' }],
          wordsUsed: ['song'],
          wordsUnused: ['guitar'],
        },
      } satisfies TalkSession),
    );
    await click('Kết thúc cuộc trò chuyện');
    await click('Kết thúc và xem nhận xét');
    expect(api.end).toHaveBeenCalledWith('s1');

    expect(text(el.querySelector('h1'))).toBe('Nhận xét của Leo');
    const review = text(el.querySelector('.review'));
    expect(review).toContain('Bạn nói khá tốt.');
    expect(review).toContain('1 câu đã nói');
    expect(review).toContain('I really like it.');
    expect(text(el.querySelector('.words.used'))).toBe('song');
    expect(text(el.querySelector('.words:not(.used)'))).toBe('guitar');
    expect(el.querySelector('.scene')).toBeNull();

    const router = TestBed.inject(Router);
    const go = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    api.start.mockReturnValue(of({ ...open, id: 's2' }));
    await click('Luyện lại chủ đề này');
    expect(api.start).toHaveBeenCalledWith('music');
    expect(go).toHaveBeenCalledWith(['/vocabulary/talk', 's2']);
    expect(text(el.querySelector('.line-text'))).toBe('Hi! Do you like this band?');
  });

  it('hangs up without asking when nothing was said', async () => {
    await setup();
    api.end.mockReturnValue(of({ ...open, status: 'ended', hints: [] }));
    await click('Kết thúc cuộc trò chuyện');
    expect(api.end).toHaveBeenCalledWith('s1');
    expect(text(el.querySelector('.review'))).toContain('chưa nói câu nào');
  });

  it('closes with a card after the last turn', async () => {
    const turns = Array.from({ length: 20 }, () => [
      { role: 'learner' as const, text: 'x', at: '2026-10-20T09:03:00Z' },
      { role: 'ai' as const, text: 'y', at: '2026-10-20T09:06:00Z' },
    ]).flat();
    await setup({ ...open, turns: [...open.turns, ...turns], hints: [] });
    expect(el.querySelector('.talk-btn')).toBeNull();
    expect(named('Kết thúc cuộc trò chuyện')).toBeUndefined();
    const card = text(el.querySelector('.finish'));
    expect(card).toContain('Cuộc trò chuyện xong rồi!');
    expect(card).toContain('20 câu đã nói');
    expect(card).toContain('6 phút');
    api.end.mockReturnValue(of({ ...open, status: 'ended', hints: [] }));
    await click('Xem Leo nhận xét');
    expect(api.end).toHaveBeenCalledWith('s1');
  });
});
