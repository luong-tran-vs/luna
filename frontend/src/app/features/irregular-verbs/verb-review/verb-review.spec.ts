import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { spellings } from '../irregular-verbs.logic';
import { VerbReview } from './verb-review';

describe('VerbReview', () => {
  let fixture: ComponentFixture<VerbReview>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!;
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const day = 24 * 60 * 60 * 1000;
  const learned = (base: string, dueInDays: number): VerbProgress => ({
    base,
    learnedAt: '2026-10-01T00:00:00Z',
    level: 1,
    dueAt: new Date(Date.now() + dueInDays * day).toISOString(),
    right: 0,
    wrong: 0,
  });

  const setup = async (verbs: VerbProgress[], query: Record<string, string> = {}) => {
    TestBed.configureTestingModule({
      imports: [VerbReview],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeSpeech(new FakeSpeech()),
        { provide: ActivatedRoute, useValue: { snapshot: { queryParamMap: convertToParamMap(query) } } },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbReview);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs });
    await settle();
  };

  afterEach(() => http.verify());

  it('lists the learned verbs, due ones first, and takes one out', async () => {
    await setup([learned('go', -2), learned('cut', 3), { ...learned('see', 0), learnedAt: null }]);
    expect(text(el.querySelector('.bar-title'))).toBe('Động từ đã học · 2');
    expect(text(el.querySelector('.start-title'))).toBe('1 từ đến hạn hôm nay');
    expect(Array.from(el.querySelectorAll('.group')).map((g) => text(g))).toEqual(['Đến hạn · 1', 'Sắp tới · 1']);
    const rows = Array.from(el.querySelectorAll('.row'));
    expect(text(rows[0].querySelector('.row-forms'))).toBe('go · went · gone');
    expect(text(rows[0].querySelector('.when'))).toBe('quá 2 ngày');
    expect(text(rows[1].querySelector('.when'))).toBe('sau 3 ngày');

    rows[1].querySelector<HTMLButtonElement>('.remove')!.click();
    const req = http.expectOne('/verbs/learned');
    expect(req.request.body).toEqual({ bases: ['cut'], learned: false });
    req.flush({ verbs: [{ base: 'cut', learnedAt: null, level: 0, dueAt: null, right: 0, wrong: 0 }] });
    await settle();
    expect(text(el.querySelector('.bar-title'))).toBe('Động từ đã học · 1');
  });

  it('reviews the due verbs and says when each comes back; a wrong one stays learned', async () => {
    await setup([learned('go', -1), learned('buy', 0), learned('cut', 5)], { start: '1' });
    // ?start=1 opens the round at once, with the due verbs only.
    const asked: string[] = [];
    for (let guard = 0; guard < 6 && el.querySelector('#verb-base'); guard++) {
      const v = IRREGULAR_VERBS.find((x) => x.base === text(el.querySelector('#verb-base')))!;
      const first = !asked.includes(v.base);
      asked.push(v.base);
      const inputs = el.querySelectorAll<HTMLInputElement>('.field input');
      inputs[0].value = spellings(v.past)[0];
      inputs[1].value = v.base === 'go' && first ? 'goed' : spellings(v.participle)[0];
      inputs.forEach((i) => i.dispatchEvent(new Event('input')));
      await settle();
      button('Kiểm tra').click();
      await settle();
      (button('Câu tiếp theo') ?? button('Xong')).click();
      await settle();
    }
    expect([...new Set(asked)].sort()).toEqual(['buy', 'go']);

    const req = http.expectOne('/verbs/results');
    expect(
      [...req.request.body.results].sort((a: { base: string }, b: { base: string }) => a.base.localeCompare(b.base)),
    ).toEqual([
      { base: 'buy', right: true },
      { base: 'go', right: false },
    ]);
    req.flush({ verbs: [learned('go', 1), learned('buy', 3)] });
    await settle();
    expect(text(el.querySelector('.summary-title'))).toBe('Ôn xong 2 từ');
    expect(text(el.querySelector('.summary'))).toContain('Nhớ 1/2');
    const wrong = el.querySelector('.row.is-wrong')!;
    expect(text(wrong)).toContain('vẫn là đã học');
    expect(text(wrong.querySelector('.when'))).toBe('hỏi lại ngày mai');
  });

  it('points to the lessons when nothing is learned', async () => {
    await setup([]);
    expect(text(el.querySelector('.empty'))).toContain('Chưa có động từ nào đã học');
    expect(el.querySelector('.empty a')?.getAttribute('href')).toBe('/irregular-verbs');
  });
});
