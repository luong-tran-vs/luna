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
      at: '',
    },
  ],
  hint: 'Yes, I love them.',
  hintVi: 'Có, mình rất thích.',
  status: 'open',
  maxTurns: 20,
  createdAt: '2026-10-20T09:00:00Z',
};

class FakeRecorder implements Recorder {
  finish!: (r: Recording) => void;
  record(): Promise<Recording> {
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
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);
  const bubbles = () => Array.from(el.querySelectorAll<HTMLElement>('.bubble'));
  const typeLine = async (value: string) => {
    const box = el.querySelector<HTMLTextAreaElement>('#talk-text')!;
    box.value = value;
    box.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };

  const setup = async (session: TalkSession = open, recognizerState: RecognizerState = 'off') => {
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

  it('shows the situation and reads the AI line aloud, the Vietnamese on demand', async () => {
    await setup();
    expect(api.get).toHaveBeenCalledWith('s1');
    expect(text(el.querySelector('.scenario'))).toContain('Bạn gặp Sam ở buổi hoà nhạc.');
    expect(speech.last().text).toBe('Hi! Do you like this band?');
    expect(text(bubbles()[0])).not.toContain('Chào!');

    el.querySelector<HTMLButtonElement>('.meaning-switch')!.click();
    await fixture.whenStable();
    expect(text(bubbles()[0])).toContain('Chào! Bạn thích ban nhạc này không?');
    expect(text(el.querySelector('.composer'))).toContain('Lượt 1/20');
  });

  it('shows the suggested answer with Gợi ý', async () => {
    await setup();
    expect(el.querySelector('.hint')).toBeNull();
    named('Gợi ý')!.click();
    await fixture.whenStable();
    expect(text(el.querySelector('.hint'))).toContain('Yes, I love them.');
    expect(text(el.querySelector('.hint'))).toContain('Có, mình rất thích.');
  });

  it('sends a typed line and shows the answer, which is read aloud', async () => {
    await setup();
    const reply = new Subject<TalkSession>();
    api.say.mockReturnValue(reply);
    await typeLine('Yes I love them');
    named('Gửi')!.click();
    await fixture.whenStable();

    expect(api.say).toHaveBeenCalledWith('s1', 'Yes I love them', false);
    expect(text(el.querySelector('.typing'))).toBe('AI đang trả lời...');

    reply.next({
      ...open,
      turns: [
        ...open.turns,
        { role: 'learner', text: 'Yes I love them', at: '' },
        { role: 'ai', text: 'Great! Who is your favorite singer?', at: '' },
      ],
    });
    await settle();
    expect(bubbles().length).toBe(3);
    expect(speech.last().text).toBe('Great! Who is your favorite singer?');
    expect(text(el.querySelector('.composer'))).toContain('Lượt 2/20');
  });

  it('puts the line back in the box when the AI fails', async () => {
    await setup();
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
    named('Gửi')!.click();
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
    expect(el.querySelector<HTMLTextAreaElement>('#talk-text')!.value).toBe('Yes');
    expect(bubbles().length).toBe(1);
  });

  it('asks before downloading the recogniser, then puts what was heard in the box', async () => {
    await setup();
    expect(el.querySelector('.mic')).toBeNull();
    expect(text(el.querySelector('.mic-area'))).toContain('khoảng 60MB');
    named('Dùng micro')!.click();
    expect(recognizer.load).toHaveBeenCalled();

    state.set('ready');
    await fixture.whenStable();
    el.querySelector<HTMLButtonElement>('.mic')!.click();
    await fixture.whenStable();
    recorder.finish({ blob: new Blob(), audio: new Float32Array(16000), seconds: 1 });
    await settle();
    expect(el.querySelector<HTMLTextAreaElement>('#talk-text')!.value).toBe('yes I like them');

    api.say.mockReturnValue(of(open));
    named('Gửi')!.click();
    expect(api.say).toHaveBeenCalledWith('s1', 'yes I like them', true);
  });

  it('ends with the review and offers another conversation', async () => {
    await setup();
    api.end.mockReturnValue(
      of({
        ...open,
        status: 'ended',
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
    named('Kết thúc và xem nhận xét')!.click();
    await settle();

    const review = text(el.querySelector('.review'));
    expect(review).toContain('Bạn nói khá tốt.');
    expect(review).toContain('I very like');
    expect(review).toContain('I really like it.');
    expect(review).toContain('Đã dùng: song');
    expect(review).toContain('Lần sau thử dùng: guitar');
    expect(el.querySelector('.composer')).toBeNull();

    const router = TestBed.inject(Router);
    const go = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    api.start.mockReturnValue(of({ ...open, id: 's2' }));
    named('Luyện lại chủ đề này')!.click();
    await settle();
    expect(api.start).toHaveBeenCalledWith('music');
    expect(go).toHaveBeenCalledWith(['/vocabulary/talk', 's2']);
  });

  it('stops taking lines after the last turn', async () => {
    const turns = Array.from({ length: 20 }, () => [
      { role: 'learner' as const, text: 'x', at: '' },
      { role: 'ai' as const, text: 'y', at: '' },
    ]).flat();
    await setup({ ...open, turns: [...open.turns, ...turns], hint: '' });
    expect(el.querySelector('#talk-text')).toBeNull();
    expect(el.textContent).toContain('Bạn đã nói đủ 20 lượt');
    expect(named('Kết thúc và xem nhận xét')!.classList).toContain('btn-primary');
  });
});
