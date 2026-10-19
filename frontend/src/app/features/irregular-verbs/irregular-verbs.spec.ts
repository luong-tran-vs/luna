import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { VerbProgress } from '../../core/models/verbs';
import { FakeSpeech, provideFakeSpeech } from '../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from './irregular-verbs.data';
import { IrregularVerbs } from './irregular-verbs';
import { VERB_LESSONS } from './verb-lessons.data';

describe('IrregularVerbs', () => {
  let fixture: ComponentFixture<IrregularVerbs>;
  let el: HTMLElement;
  let speech: FakeSpeech;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label));
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const rows = () => Array.from(el.querySelectorAll('tr.verb'));
  const bases = () => rows().map((r) => text(r.querySelector('.open')));
  const row = (base: string) => rows().find((r) => text(r.querySelector('.open')) === base)!;
  const search = async (q: string) => {
    const input = el.querySelector<HTMLInputElement>('#verb-search')!;
    input.value = q;
    input.dispatchEvent(new Event('input'));
    await settle();
  };
  const learned = (base: string, dueAt = '2999-01-01T00:00:00Z'): VerbProgress => ({
    base,
    learnedAt: '2026-10-01T00:00:00Z',
    level: 1,
    dueAt,
    right: 0,
    wrong: 0,
  });

  /** Renders the tab; the learner's progress answers with `verbs` (or fails with null). */
  const render = async (verbs: VerbProgress[] | null = [], supported = true) => {
    sessionStorage.clear();
    speech = new FakeSpeech();
    speech.supported = supported;
    await TestBed.configureTestingModule({
      imports: [IrregularVerbs],
      providers: [provideFakeSpeech(speech), provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(IrregularVerbs);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    const req = http.expectOne('/verbs/progress');
    if (verbs) {
      req.flush({ verbs });
    } else {
      req.flush({}, { status: 500, statusText: 'x' });
    }
    await settle();
  };
  const lookup = async () => {
    button('Tra cứu')!.click();
    await settle();
  };

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  describe('Học', () => {
    it('shows the learned common verbs, the next lesson with its pattern and the way to review', async () => {
      await render([...VERB_LESSONS[0].bases.map((b) => learned(b)), learned('go', '2000-01-01T00:00:00Z')]);
      expect(button('Học')!.getAttribute('aria-pressed')).toBe('true');
      const hero = el.querySelector('.hero')!;
      expect(text(hero.querySelector('.ring-inner'))).toBe('7/104');
      expect(text(hero.querySelector('.hero-eyebrow'))).toBe(`Học tiếp · Bài 2/${VERB_LESSONS.length}`);
      expect(text(hero.querySelector('.hero-title'))).toBe(VERB_LESSONS[1].title);
      expect(text(hero.querySelector('.hero-model'))).toBe('bet → bet → bet');
      const [learn, review] = Array.from(hero.querySelectorAll('a.hero-btn'));
      expect(learn.getAttribute('href')).toBe('/irregular-verbs/lessons/2');
      expect(review.getAttribute('href')).toBe('/irregular-verbs/review?start=1');
      expect(text(review.querySelector('.badge'))).toBe('1 từ đến hạn');
    });

    it('lists the lessons with a model of three forms each, done and next marked', async () => {
      await render([...VERB_LESSONS[0].bases.map((b) => learned(b)), learned('send')]);
      const lessons = Array.from(el.querySelectorAll('a.lesson'));
      expect(lessons.length).toBe(VERB_LESSONS.length);
      expect(lessons[0].classList).toContain('is-done');
      expect(text(lessons[0].querySelector('.model'))).toBe('cut → cut → cut');
      expect(text(lessons[0].querySelector('.others'))).toBe('put · let · hit · set · cost');
      expect(lessons[1].classList).toContain('is-current');
      expect(text(lessons[3].querySelector('.model'))).toBe('send → sent → sent');
      expect(text(lessons[3].querySelector('.state'))).toBe('1/5');
      expect(lessons[3].getAttribute('href')).toBe('/irregular-verbs/lessons/4');
    });

    it('shows the path in two parts and moves on to "Nâng cao" once the common verbs are learned', async () => {
      const common = VERB_LESSONS.filter((l) => !l.advanced);
      await render(common.flatMap((l) => l.bases.map((b) => learned(b))));
      expect(Array.from(el.querySelectorAll('.group-title')).map((t) => text(t))).toEqual([
        `Thông dụng · ${common.length} bài`,
        `Nâng cao · ${VERB_LESSONS.length - common.length} bài`,
      ]);
      expect(text(el.querySelector('.hero-eyebrow'))).toBe(`Học tiếp · Bài 19/${VERB_LESSONS.length}`);
      expect(text(el.querySelector('.ring-inner'))).toBe(`104/${IRREGULAR_VERBS.length}`);
      expect(el.querySelector('.ring')?.getAttribute('aria-label')).toBe('Động từ đã học');
    });

    it('starts at lesson 1 with no badge when nothing is learned', async () => {
      await render();
      expect(text(el.querySelector('.hero-eyebrow'))).toBe(`Học tiếp · Bài 1/${VERB_LESSONS.length}`);
      expect(el.querySelector('.badge')).toBeNull();
      expect(el.querySelector('a.hero-btn.review')?.getAttribute('href')).toBe('/irregular-verbs/review');
    });
  });

  describe('Tra cứu', () => {
    it('lists every verb in three aligned columns by letter, the learned ones ticked', async () => {
      await render([learned('go')]);
      await lookup();
      expect(sessionStorage.getItem('luna.verbs.view')).toBe('lookup');
      expect(rows().length).toBe(IRREGULAR_VERBS.length);
      expect(Array.from(el.querySelectorAll('thead th')).map((t) => text(t))).toEqual([
        'Đã học',
        'V1',
        'V2 · quá khứ',
        'V3 · phân từ',
        'Nghĩa',
      ]);
      const go = row('go');
      expect(text(go.querySelector('.v2'))).toBe('went');
      expect(text(go.querySelector('.v3'))).toBe('gone');
      expect(text(go.querySelector('.vi'))).toBe('đi');
      expect(text(go.querySelector('.c'))).toBe('Đã học');
      expect(text(row('see').querySelector('.c'))).toBe('');
      expect(text(row('burn').querySelector('.v2'))).toBe('burnt / hoặc burned');
      expect(el.querySelectorAll('.letter-row').length).toBeGreaterThan(10);
    });

    it('searches by any form or by the Vietnamese meaning', async () => {
      await render();
      await lookup();
      await search('went');
      expect(bases()).toEqual(['go']);
      expect(el.querySelector('.letter-row')).toBeNull();
      expect(text(el.querySelector('.result'))).toBe(`Hiện 1 / ${IRREGULAR_VERBS.length} động từ`);
      await search('nhin thay');
      expect(bases()).toContain('see');
      await search('zzz');
      expect(rows().length).toBe(0);
      expect(text(el.querySelector('.empty'))).toContain('Không tìm thấy động từ nào.');
    });

    it('filters the common verbs, the learned ones and a pattern in plain words', async () => {
      await render([learned('go'), learned('cut')]);
      await lookup();
      button('Thông dụng')!.click();
      await settle();
      expect(rows().length).toBe(IRREGULAR_VERBS.filter((v) => v.common).length);

      button('Đã học')!.click();
      await settle();
      expect(bases()).toEqual(['cut', 'go']);
      button('Đã học')!.click();
      await settle();

      expect(text(button('Không đổi'))).toContain('cut');
      button('Không đổi')!.click();
      await settle();
      expect(bases()).toContain('cut');
      expect(bases()).not.toContain('go');
      button('Không đổi')!.click();
      button('Tất cả')!.click();
      await settle();
      expect(rows().length).toBe(IRREGULAR_VERBS.length);
    });

    it('opens a verb from its row: forms with how they are said, the tip, the example, and marks it learned', async () => {
      await render();
      await lookup();
      row('read').querySelector<HTMLButtonElement>('.open')!.click();
      await settle();
      const sheet = el.querySelector('.sheet')!;
      expect(sheet.getAttribute('role')).toBe('dialog');
      expect(text(sheet.querySelector('.title'))).toBe('read');
      expect(Array.from(sheet.querySelectorAll('.ipa')).map((i) => text(i))).toEqual(['/riːd/', '/red/', '/red/']);
      expect(text(sheet.querySelector('.tip'))).toContain('/red/');
      expect(text(sheet.querySelector('.verb-mark'))).toBe('read');
      expect(sheet.querySelector('a.btn')?.getAttribute('href')).toBe('/irregular-verbs/lessons/9');
      expect(document.activeElement?.getAttribute('aria-label')).toBe('Đóng');

      el.querySelector<HTMLButtonElement>('.sheet .mark')!.click();
      const req = http.expectOne('/verbs/learned');
      expect(req.request.body).toEqual({ bases: ['read'], learned: true });
      req.flush({ verbs: [learned('read')] });
      await settle();
      expect(el.querySelector('.sheet .mark')?.getAttribute('aria-pressed')).toBe('true');
      expect(text(el.querySelector('.mark-note'))).toContain('Đã đánh dấu read là đã học');

      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
      await settle();
      expect(el.querySelector('.sheet')).toBeNull();
      expect(text(row('read').querySelector('.c'))).toBe('Đã học');

      button('Hoàn tác')!.click();
      const undo = http.expectOne('/verbs/learned');
      expect(undo.request.body).toEqual({ bases: ['read'], learned: false });
      undo.flush({ verbs: [{ ...learned('read'), learnedAt: null, level: 0, dueAt: null }] });
      await settle();
      expect(text(row('read').querySelector('.c'))).toBe('');
    });

    it('reads a verb aloud from its details', async () => {
      await render();
      await lookup();
      row('go').querySelector<HTMLButtonElement>('.open')!.click();
      await settle();
      el.querySelector<HTMLButtonElement>('.sheet .round')!.click();
      expect(speech.last()).toMatchObject({ text: 'go, went, gone' });
    });

    it('still lists the verbs, without "Đã học", when the progress cannot be loaded', async () => {
      await render(null, false);
      await lookup();
      expect(rows().length).toBe(IRREGULAR_VERBS.length);
      expect(button('Đã học')).toBeUndefined();
      row('go').querySelector<HTMLButtonElement>('.open')!.click();
      await settle();
      expect(el.querySelector('.sheet .mark')).toBeNull();
      expect(el.querySelector('.sheet .round:not(.plain)')).toBeNull();
    });
  });
});
