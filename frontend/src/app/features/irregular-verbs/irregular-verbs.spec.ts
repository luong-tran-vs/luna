import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { VerbProgress } from '../../core/models/verbs';
import { FakeSpeech, provideFakeSpeech } from '../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from './irregular-verbs.data';
import { IrregularVerbs } from './irregular-verbs';

describe('IrregularVerbs', () => {
  let fixture: ComponentFixture<IrregularVerbs>;
  let el: HTMLElement;
  let speech: FakeSpeech;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label));
  const rows = () => Array.from(el.querySelectorAll('tr.verb'));
  const bases = () =>
    rows().map((r) => text(r.querySelector('.v1')).replace(' (thông dụng)', '').replace(' (đã thuộc)', ''));
  const row = (base: string) =>
    rows().find((r) => text(r.querySelector('.v1')).startsWith(base + ' ') || text(r.querySelector('.v1')) === base)!;
  const search = async (q: string) => {
    const input = el.querySelector<HTMLInputElement>('#verb-search')!;
    input.value = q;
    input.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };

  /** Renders the tab; the learner's progress answers with `verbs` (or fails with null). */
  const render = async (supported = true, verbs: VerbProgress[] | null = []) => {
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
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('leads to the practice with how many common verbs are known, and ticks the known ones', async () => {
    const commonTotal = IRREGULAR_VERBS.filter((v) => v.common).length;
    await render(true, [
      { base: 'go', level: 2, dueAt: '2000-01-01T00:00:00Z', right: 2, wrong: 0 },
      { base: 'see', level: 1, dueAt: '2999-01-01T00:00:00Z', right: 1, wrong: 0 },
      { base: 'cut', level: 0, dueAt: null, right: 0, wrong: 1 },
    ]);
    const card = el.querySelector('.practice-card')!;
    expect(text(card.querySelector('.known-pill'))).toBe(`Đã thuộc 2/${commonTotal} thông dụng`);
    expect(text(card.querySelector('.due'))).toBe('1 động từ đến hạn ôn hôm nay');
    expect(card.querySelector('a.start')?.getAttribute('href')).toBe('/irregular-verbs/practice');
    expect(row('go').querySelector('.known')).not.toBeNull();
    expect(row('cut').querySelector('.known')).toBeNull();
  });

  it('still lists the verbs when the progress cannot be loaded', async () => {
    await render(true, null);
    expect(rows().length).toBe(IRREGULAR_VERBS.length);
    expect(el.querySelector('.known-pill')).toBeNull();
    expect(el.querySelector('.practice-card a.start')).not.toBeNull();
  });
  it('lists every verb by letter with its three forms and meaning', async () => {
    await render();
    expect(rows().length).toBe(IRREGULAR_VERBS.length);
    expect(text(el.querySelector('.result'))).toBe(`Hiện tất cả ${IRREGULAR_VERBS.length} động từ`);
    const go = row('go');
    expect(text(go.querySelector('.v2'))).toBe('went');
    expect(text(go.querySelector('.v3'))).toBe('gone');
    expect(text(go.querySelector('.vi'))).toBe('đi');
    expect(go.querySelector('.common')).not.toBeNull();
    expect(text(row('burn').querySelector('.v2'))).toBe('burnt / hoặc burned');
    expect(text(row('read').querySelector('.note'))).toContain('/red/');
    expect(el.querySelectorAll('.letter-row').length).toBe(el.querySelectorAll('.letters .letter').length);
  });

  it('searches by any form or by the Vietnamese meaning', async () => {
    await render();
    await search('went');
    expect(bases()).toEqual(['go']);
    expect(el.querySelector('.letter-row')).toBeNull();
    expect(el.querySelector('.letters')).toBeNull();
    expect(text(el.querySelector('.result'))).toBe(`Hiện 1 / ${IRREGULAR_VERBS.length} động từ`);

    await search('nhin thay');
    expect(bases()).toContain('see');

    await search('zzz');
    expect(rows().length).toBe(0);
    expect(text(el.querySelector('.empty'))).toContain('Không tìm thấy động từ nào.');

    button('')!.click(); // the clear button has only an icon
    await fixture.whenStable();
  });

  it('filters the common verbs and the patterns', async () => {
    await render();
    button('Thông dụng')!.click();
    await fixture.whenStable();
    expect(rows().length).toBe(IRREGULAR_VERBS.filter((v) => v.common).length);
    expect(button('Thông dụng')!.getAttribute('aria-pressed')).toBe('true');

    button('A – A – A')!.click();
    await fixture.whenStable();
    expect(bases()).toContain('cut');
    expect(bases()).not.toContain('go');
    for (const r of rows()) {
      expect(text(r.querySelector('.v2'))).toBe(text(r.querySelector('.v1')).replace(' (thông dụng)', ''));
    }
    // Tapping the chosen pattern again shows every pattern.
    expect(button('A – A – A')!.getAttribute('aria-pressed')).toBe('true');
    button('A – A – A')!.click();
    button('Tất cả')!.click();
    await fixture.whenStable();
    expect(rows().length).toBe(IRREGULAR_VERBS.length);
    expect(button('A – A – A')!.getAttribute('aria-pressed')).toBe('false');
  });

  it('reads a verb aloud and stops when pressed again', async () => {
    await render();
    const listen = row('go').querySelector<HTMLButtonElement>('.listen-btn')!;
    expect(listen.getAttribute('aria-label')).toBe('Nghe go, went, gone');
    listen.click();
    expect(speech.last()).toMatchObject({ text: 'go, went, gone' });
    speech.playing.set('go, went, gone');
    await fixture.whenStable();
    expect(row('go').classList).toContain('playing');
    listen.click();
    expect(speech.stops).toBe(1);
  });

  it('has no listen buttons without a voice', async () => {
    await render(false);
    expect(el.querySelector('.listen-btn')).toBeNull();
    expect(el.textContent).not.toContain('để nghe cả ba dạng');
  });
});
