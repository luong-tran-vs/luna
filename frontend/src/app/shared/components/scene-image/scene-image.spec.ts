import { ComponentFixture, TestBed } from '@angular/core/testing';

import { serverUrl } from '../../../core/api-url';
import { SceneImage } from './scene-image';

describe('SceneImage', () => {
  let fixture: ComponentFixture<SceneImage>;
  let el: HTMLElement;

  const open = async (url: string | null | undefined) => {
    fixture = TestBed.createComponent(SceneImage);
    fixture.componentRef.setInput('url', url);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('shows the picture from the server, as decoration', async () => {
    await open('/api/lessons/l1/images/_scene');
    const img = el.querySelector('img')!;
    expect(img.getAttribute('src')).toBe(serverUrl('/api/lessons/l1/images/_scene'));
    expect(img.getAttribute('alt')).toBe('');
    expect(img.classList.contains('loaded')).toBe(false);
    img.dispatchEvent(new Event('load'));
    await fixture.whenStable();
    expect(img.classList.contains('loaded')).toBe(true);
  });

  it('shows nothing without a picture, or when it fails to load', async () => {
    await open('');
    expect(el.querySelector('img')).toBeNull();
    await open(undefined);
    expect(el.querySelector('img')).toBeNull();

    await open('/api/lessons/l1/images/_scene');
    el.querySelector('img')!.dispatchEvent(new Event('error'));
    await fixture.whenStable();
    expect(el.querySelector('img')).toBeNull();
  });
});
