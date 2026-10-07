import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SpeechService } from '../../../core/services/speech.service';
import { micError, SttLab } from './stt-lab';

describe('micError', () => {
  it('explains a refused permission and a missing microphone', () => {
    expect(micError(new DOMException('x', 'NotAllowedError'))).toContain('Chưa cho phép dùng micro');
    expect(micError(new DOMException('x', 'NotFoundError'))).toBe('Không tìm thấy micro trên máy này.');
    expect(micError(new Error('boom'))).toContain('Không ghi âm được');
  });
});

describe('SttLab', () => {
  let fixture: ComponentFixture<SttLab>;
  let el: HTMLElement;
  const speak = vi.fn();

  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === label)!;

  beforeEach(async () => {
    speak.mockClear();
    TestBed.configureTestingModule({
      providers: [{ provide: SpeechService, useValue: { supported: true, speak, stop: vi.fn() } }],
    });
    fixture = TestBed.createComponent(SttLab);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  it('offers the models, Moonshine tiny first, on the CPU', () => {
    const options = Array.from(el.querySelectorAll<HTMLOptionElement>('#stt-model option')).map((o) => o.textContent?.trim());
    expect(options).toEqual([
      'Moonshine tiny (khoảng 28MB)',
      'Moonshine base (khoảng 62MB)',
      'Whisper tiny.en (khoảng 41MB)',
      'Whisper base.en (khoảng 77MB)',
    ]);
    expect(el.querySelector<HTMLSelectElement>('#stt-device')!.value).toBe('wasm');
    expect(el.textContent).toContain('Tải model ở bước 1');
  });

  it('says the microphone needs HTTPS when the page cannot record', () => {
    // jsdom has no MediaRecorder.
    expect(button('Ghi âm').disabled).toBe(true);
    expect(el.textContent).toContain('chỉ cho dùng micro khi trang mở qua HTTPS');
  });

  it('cycles through the sample sentences and reads one aloud', async () => {
    const input = () => el.querySelector<HTMLInputElement>('#stt-sentence')!;
    expect(input().value).toBe('I usually drink coffee in the morning.');
    button('Câu khác').click();
    await fixture.whenStable();
    expect(input().value).toBe('Could you tell me where the station is?');
    button('Nghe mẫu').click();
    expect(speak).toHaveBeenCalledWith('Could you tell me where the station is?', 1);
  });
});
