import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Card } from '../../../core/models/vocab';
import { CardForm } from './card-form';

const went: Card = {
  id: 'c1', text: 'went', lemma: 'go', ipa: '/went/', meaningVi: 'đã đi', contextSentence: 'We went home.',
  lessonId: 'l1', source: 'ai', createdAt: '2026-09-29T01:00:00Z', due: '2026-09-30T17:00:00Z', reps: 2, state: 'review',
};

@Component({
  imports: [CardForm],
  template: `<lu-card-form [card]="card()" (saved)="saved.push($event)" (cancelled)="cancelled = cancelled + 1" />`,
})
class Host {
  readonly card = signal<Card | null>(null);
  readonly saved: Card[] = [];
  cancelled = 0;
}

describe('CardForm', () => {
  let fixture: ComponentFixture<Host>;
  let host: Host;
  let el: HTMLElement;
  let http: HttpTestingController;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const field = (id: string) => el.querySelector<HTMLInputElement | HTMLTextAreaElement>(`#${id}`)!;
  const type = (id: string, value: string) => {
    field(id).value = value;
    field(id).dispatchEvent(new Event('input'));
  };
  const submit = async () => {
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await settle();
  };
  const errorOf = (id: string) => el.querySelector(`#${id}-error`)?.textContent?.trim();

  const setup = async (card: Card | null = null) => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Host);
    host = fixture.componentInstance;
    host.card.set(card);
    el = fixture.nativeElement as HTMLElement;
    await settle();
  };

  afterEach(() => http.verify());

  describe('add', () => {
    it('requires the word and the meaning', async () => {
      await setup();
      await submit();
      expect(errorOf('card-text')).toBe('Vui lòng nhập từ');
      expect(errorOf('card-meaning')).toBe('Vui lòng nhập nghĩa');
      expect(errorOf('card-ipa')).toBeUndefined();
    });

    it('posts a manual card without a lesson', async () => {
      await setup();
      type('card-text', '  Serendipity ');
      type('card-meaning', 'sự tình cờ may mắn');
      await submit();
      const req = http.expectOne('/vocab/cards');
      expect(req.request.body).toEqual({
        text: 'Serendipity', lemma: 'serendipity', ipa: '', meaningVi: 'sự tình cờ may mắn', contextSentence: '',
        source: 'manual',
      });
      req.flush({ card: { ...went, id: 'c9', text: 'Serendipity' } }, { status: 201, statusText: 'Created' });
      await settle();
      expect(host.saved.map((c) => c.id)).toEqual(['c9']);
    });

    it('says when the word is already in the notebook', async () => {
      await setup();
      type('card-text', 'go');
      type('card-meaning', 'đi');
      await submit();
      http.expectOne('/vocab/cards').flush(
        { error: 'card_exists', message: 'Từ này đã có trong sổ', card: went },
        { status: 409, statusText: 'Conflict' },
      );
      await settle();
      expect(errorOf('card-text')).toBe('Từ này đã có trong sổ');
      expect(host.saved).toEqual([]);
    });

    it('shows field errors from the server', async () => {
      await setup();
      type('card-text', 'go');
      type('card-meaning', 'đi');
      await submit();
      http.expectOne('/vocab/cards').flush(
        { error: 'validation_failed', message: 'x', fields: { meaningVi: 'Nghĩa cần từ 1 đến 200 ký tự' } },
        { status: 400, statusText: 'Bad Request' },
      );
      await settle();
      expect(errorOf('card-meaning')).toBe('Nghĩa cần từ 1 đến 200 ký tự');
    });

    it('cancels', async () => {
      await setup();
      Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Huỷ')!.click();
      expect(host.cancelled).toBe(1);
    });
  });

  describe('edit', () => {
    it('shows the word read-only and patches only changed fields', async () => {
      await setup(went);
      expect(el.querySelector('#card-text')).toBeNull();
      expect(el.querySelector('.word')?.textContent?.trim()).toBe('went');
      expect(field('card-meaning').value).toBe('đã đi');

      type('card-meaning', 'đã đi (quá khứ của go)');
      await submit();
      const req = http.expectOne('/vocab/cards/c1');
      expect(req.request.method).toBe('PATCH');
      expect(req.request.body).toEqual({ meaningVi: 'đã đi (quá khứ của go)' });
      req.flush({ card: { ...went, meaningVi: 'đã đi (quá khứ của go)' } });
      await settle();
      expect(host.saved[0].meaningVi).toBe('đã đi (quá khứ của go)');
    });

    it('closes without a request when nothing changed', async () => {
      await setup(went);
      await submit();
      expect(host.saved).toEqual([went]);
    });
  });
});
