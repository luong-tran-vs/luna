import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SpeechService } from '../../../core/services/speech.service';
import { splitSentences, TtsLab } from './tts-lab';

describe('splitSentences', () => {
  it('splits after . ! and ? and drops empty parts', () => {
    expect(splitSentences('Hi there. How are you?  Fine!\n')).toEqual(['Hi there.', 'How are you?', 'Fine!']);
    expect(splitSentences('   ')).toEqual([]);
  });
});

describe('TtsLab', () => {
  let fixture: ComponentFixture<TtsLab>;
  let el: HTMLElement;
  const speak = vi.fn();

  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === label)!;

  beforeEach(async () => {
    speak.mockClear();
    TestBed.configureTestingModule({
      providers: [{ provide: SpeechService, useValue: { supported: true, speak, stop: vi.fn() } }],
    });
    fixture = TestBed.createComponent(TtsLab);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  it('asks to load the voice before reading with Piper', () => {
    expect(button('Tải giọng').disabled).toBe(false);
    expect(button('Đọc bằng Piper').disabled).toBe(true);
    expect(el.textContent).toContain('Tải giọng ở bước 1');
    expect(el.querySelector<HTMLSelectElement>('#lab-voice')!.value).toBe('en_US-hfc_female-medium');
  });

  it('reads the text with the browser voice for comparison', () => {
    button('Đọc bằng giọng trình duyệt').click();
    expect(speak).toHaveBeenCalledWith(expect.stringContaining('On Sunday'), 1, expect.any(Object));
  });
});
