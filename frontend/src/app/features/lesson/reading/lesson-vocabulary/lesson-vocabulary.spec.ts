import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../../core/interceptors/error-interceptor';
import { VocabItem } from '../../../../core/models/vocab';
import { LessonVocabulary } from './lesson-vocabulary';

const items: VocabItem[] = [
  { lemma: 'go', text: 'went', meaningVi: 'đi', ipa: '/ɡəʊ/', sentenceIndex: 0, sentence: 'We went to the park.' },
  { lemma: 'give up', text: 'gave up', meaningVi: 'từ bỏ', ipa: '', sentenceIndex: 1, sentence: 'He gave up.' },
  { lemma: 'park', text: 'park', meaningVi: 'công viên', ipa: '/pɑːk/', sentenceIndex: 0, sentence: 'We went to the park.' },
];

@Component({
  imports: [LessonVocabulary],
  template: `<lu-lesson-vocabulary lessonId="l1" [saved]="saved()" (added)="onAdded($event)" />`,
})
class Host {
  readonly saved = signal<ReadonlySet<string>>(new Set(['go']));
  onAdded(lemmas: string[]): void {
    this.saved.update((s) => new Set([...s, ...lemmas]));
  }
}

describe('LessonVocabulary', () => {
  let fixture: ComponentFixture<Host>;
  let host: Host;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) => Array.from(el.querySelectorAll('button')).find((b) => text(b) === label);
  const row = (lemma: string) =>
    Array.from(el.querySelectorAll('.vocab-item')).find((r) => text(r.querySelector('.lemma')) === lemma)!;

  const open = async (available = true) => {
    const toggle = el.querySelector<HTMLButtonElement>('.toggle')!;
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    toggle.click();
    await settle();
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    http.expectOne('/api/lessons/l1/vocabulary').flush({ available, items: available ? items : [] });
    await settle();
  };

  beforeEach(async () => {
    vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined);
    vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => undefined);
    TestBed.configureTestingModule({
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Host);
    host = fixture.componentInstance;
    el = fixture.nativeElement as HTMLElement;
    await settle();
  });

  afterEach(() => {
    http.verify();
    vi.restoreAllMocks();
  });

  it('loads nothing until opened', () => {
    expect(text(el.querySelector('.toggle'))).toBe('Từ vựng');
    expect(el.querySelector('.vocab-item')).toBeNull();
  });

  it('lists the annotated words with saved ones marked', async () => {
    await open();
    expect(text(row('go').querySelector('.ipa'))).toBe('/ɡəʊ/');
    expect(text(row('go').querySelector('.meaning'))).toBe('đi');
    expect(text(row('go'))).toContain('✓ Đã lưu');
    expect(row('go').querySelector('button.save')).toBeNull();
    expect(text(row('give up').querySelector('button.save'))).toBe('Lưu give up');
  });

  it('plays a word', async () => {
    await open();
    row('park').querySelector<HTMLButtonElement>('button.listen')!.click();
    await settle();
    expect(el.querySelector('audio')!.getAttribute('src')).toBe('/api/tts/word?text=park');
  });

  it('saves one word', async () => {
    await open();
    row('park').querySelector<HTMLButtonElement>('button.save')!.click();
    await settle();
    const req = http.expectOne('/api/vocab/cards/bulk');
    expect(req.request.body).toEqual({ lessonId: 'l1', lemmas: ['park'] });
    req.flush({ added: 1, cards: [] });
    await settle();
    expect(host.saved().has('park')).toBe(true);
    expect(text(row('park'))).toContain('✓ Đã lưu');
  });

  it('saves all unsaved words once', async () => {
    await open();
    button('Lưu tất cả')!.click();
    await settle();
    const req = http.expectOne('/api/vocab/cards/bulk');
    expect(req.request.body).toEqual({ lessonId: 'l1', lemmas: ['give up', 'park'] });
    req.flush({ added: 2, cards: [] });
    await settle();
    expect(text(el.querySelector('.status'))).toBe('Đã lưu 2 từ');
    expect(el.querySelectorAll('button.save')).toHaveLength(0);
    expect(button('Lưu tất cả')).toBeUndefined();
  });

  it('shows an error when saving fails', async () => {
    await open();
    button('Lưu tất cả')!.click();
    await settle();
    http.expectOne('/api/vocab/cards/bulk').flush({ error: 'x', message: 'x' }, { status: 500, statusText: 'Error' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toBe('Không lưu được, vui lòng thử lại.');
    expect(button('Lưu tất cả')).toBeTruthy();
  });

  it('says when the lesson has no vocabulary yet', async () => {
    await open(false);
    expect(el.textContent).toContain('Chưa có danh sách từ vựng');
    expect(button('Lưu tất cả')).toBeUndefined();
  });

  it('follows words saved elsewhere on the page', async () => {
    await open();
    host.saved.set(new Set(['go', 'park']));
    await settle();
    expect(text(row('park'))).toContain('✓ Đã lưu');
  });
});
