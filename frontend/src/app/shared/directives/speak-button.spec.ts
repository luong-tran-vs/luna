import { ChangeDetectionStrategy, Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { FakeSpeech, provideFakeSpeech } from '../../core/services/speech.service.testing';

import { SpeakButton } from './speak-button';

@Component({
  imports: [SpeakButton],
  template: `
    <button type="button" id="hello" luSpeak="Hello." (click)="pressed = pressed + 1">Nghe</button>
    <button type="button" id="any" luSpeak (click)="pressed = pressed + 1">Nghe</button>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
class Host {
  pressed = 0;
}

describe('SpeakButton', () => {
  let speech: FakeSpeech;

  const setup = async () => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({ providers: [provideFakeSpeech(speech)] });
    const fixture = TestBed.createComponent(Host);
    await fixture.whenStable();
    const el = fixture.nativeElement as HTMLElement;
    return {
      fixture,
      hello: () => el.querySelector<HTMLButtonElement>('#hello')!,
      any: () => el.querySelector<HTMLButtonElement>('#any')!,
    };
  };

  it('shows preparing on the button whose text waits for the natural voice, or any text when left empty', async () => {
    const { fixture, hello, any } = await setup();
    expect(hello().classList.contains('preparing')).toBe(false);
    expect(hello().getAttribute('aria-busy')).toBeNull();

    speech.preparing.set('Other.');
    await fixture.whenStable();
    expect(hello().classList.contains('preparing')).toBe(false);
    expect(any().classList.contains('preparing')).toBe(true);

    speech.preparing.set('Hello.');
    await fixture.whenStable();
    expect(hello().classList.contains('preparing')).toBe(true);
    expect(hello().getAttribute('aria-busy')).toBe('true');
    expect(hello().getAttribute('title')).toBe('Đang chuẩn bị giọng đọc…');

    speech.preparing.set(null);
    await fixture.whenStable();
    expect(any().classList.contains('preparing')).toBe(false);
  });

  it('shows speaking on the button whose text is read, and stops it when pressed instead of reading again', async () => {
    const { fixture, hello, any } = await setup();
    hello().click();
    expect(fixture.componentInstance.pressed).toBe(1);

    speech.playing.set('Hello.');
    await fixture.whenStable();
    expect(hello().classList.contains('speaking')).toBe(true);
    expect(hello().getAttribute('aria-pressed')).toBe('true');
    expect(hello().getAttribute('title')).toBe('Dừng đọc');
    // A button without a text keeps its own playing state.
    expect(any().classList.contains('speaking')).toBe(false);

    hello().click();
    expect(fixture.componentInstance.pressed).toBe(1);
    expect(speech.stops).toBe(1);
    await fixture.whenStable();
    expect(hello().classList.contains('speaking')).toBe(false);
    expect(hello().getAttribute('aria-pressed')).toBeNull();

    hello().click();
    expect(fixture.componentInstance.pressed).toBe(2);
  });
});
