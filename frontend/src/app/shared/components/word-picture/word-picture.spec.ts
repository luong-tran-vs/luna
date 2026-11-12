import { ComponentFixture, TestBed } from '@angular/core/testing';

import { serverUrl } from '../../../core/api-url';
import { WordPicture } from './word-picture';

describe('WordPicture', () => {
  let fixture: ComponentFixture<WordPicture>;
  let el: HTMLElement;

  const open = async (url: string | null | undefined) => {
    fixture = TestBed.createComponent(WordPicture);
    fixture.componentRef.setInput('url', url);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows the picture from the server, as decoration', async () => {
    await open('/api/words/go/image?v=1');
    const img = el.querySelector('img')!;
    expect(img.getAttribute('src')).toBe(serverUrl('/api/words/go/image?v=1'));
    expect(img.getAttribute('alt')).toBe('');
    img.dispatchEvent(new Event('load'));
    await fixture.whenStable();
    expect(img.classList.contains('loaded')).toBe(true);
  });

  it('shows nothing without a picture, or when it fails to load, until the next word', async () => {
    await open('');
    expect(el.querySelector('img')).toBeNull();

    await open('/api/words/go/image?v=1');
    el.querySelector('img')!.dispatchEvent(new Event('error'));
    await fixture.whenStable();
    expect(el.querySelector('img')).toBeNull();

    fixture.componentRef.setInput('url', '/api/words/run/image?v=1');
    await fixture.whenStable();
    const img = el.querySelector('img')!;
    expect(img.getAttribute('src')).toBe(serverUrl('/api/words/run/image?v=1'));
    expect(img.classList.contains('loaded')).toBe(false);
  });
});
