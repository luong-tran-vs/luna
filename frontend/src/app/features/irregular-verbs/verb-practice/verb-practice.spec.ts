import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { VerbProgress } from '../../../core/models/verbs';
import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { spellings } from '../irregular-verbs.logic';
import { inSet } from './verb-practice.logic';
import { VerbPractice } from './verb-practice';

describe('VerbPractice', () => {
  let fixture: ComponentFixture<VerbPractice>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let speech: FakeSpeech;

  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!;
  const verbOnScreen = () => IRREGULAR_VERBS.find((v) => v.base === text(el.querySelector('#verb-base')))!;
  const type = (index: number, value: string) => {
    const input = el.querySelectorAll<HTMLInputElement>('.field input')[index];
    input.value = value;
    input.dispatchEvent(new Event('input'));
  };
  /** Answers the verb on screen, right or with a wrong V3, and checks. */
  const answer = async (right: boolean) => {
    const v = verbOnScreen();
    type(0, spellings(v.past)[0]);
    type(1, right ? spellings(v.participle)[0] : 'nope');
    await settle();
    button('Kiểm tra').click();
    await settle();
    return v;
  };
  const choose = async (key: string) => {
    el.querySelector<HTMLInputElement>(`input[name="verb-set"][value="${key}"]`)!.click();
    await settle();
  };

  const setup = async (verbs: VerbProgress[] = []) => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({
      imports: [VerbPractice],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting(), provideFakeSpeech(speech)],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbPractice);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({ verbs });
    await settle();
  };

  afterEach(() => http.verify());

  it('offers the sets with how much of each is known', async () => {
    await setup([
      { base: 'go', level: 1, dueAt: '2000-01-01T00:00:00Z', right: 1, wrong: 0 },
      { base: 'see', level: 1, dueAt: '2999-01-01T00:00:00Z', right: 1, wrong: 0 },
    ]);
    const common = inSet(IRREGULAR_VERBS, 'common').length;
    const sets = Array.from(el.querySelectorAll('.set'));
    expect(sets.map((s) => text(s.querySelector('.set-title')))).toEqual([
      'Thông dụng',
      'A – A – A',
      'A – B – B',
      'A – B – A',
      'A – B – C',
      'Tất cả',
    ]);
    expect(text(sets[0].querySelector('.set-count'))).toBe(`2/${common} đã thuộc 1 đến hạn`);
    expect(sets[0].classList).toContain('is-chosen');
    expect(text(el.querySelector('.bottom-bar'))).toContain('Bắt đầu 10 động từ');
    expect(el.querySelector('.legend')).not.toBeNull();
  });

  it('runs a round, asks a wrong verb again at the end, and records the first answers', async () => {
    await setup();
    await choose('ABA');
    const size = inSet(IRREGULAR_VERBS, 'ABA').length;
    expect(text(el.querySelector('.bottom-bar'))).toContain(`Bắt đầu ${size} động từ`);
    button('Bắt đầu').click();
    await settle();

    // A Kiểm tra that waits for both forms.
    expect(button('Kiểm tra').disabled).toBe(true);
    const missed = await answer(false);
    expect(el.querySelectorAll('.field.ok').length).toBe(1);
    expect(el.querySelectorAll('.field.bad').length).toBe(1);
    expect(text(el.querySelector('.feedback'))).toContain('Chưa đúng.');
    expect(text(el.querySelector('.feedback'))).toContain('hỏi lại cuối lượt');
    expect(text(el.querySelector('.count'))).toBe(`1/${size + 1}`);
    button('Câu tiếp theo').click();
    await settle();

    for (let i = 1; i < size; i++) {
      await answer(true);
      expect(text(el.querySelector('.feedback'))).toContain('Đúng rồi!');
      button('Câu tiếp theo').click();
      await settle();
    }
    // The wrong verb comes back, marked; right this time, it is not asked a third time.
    expect(verbOnScreen().base).toBe(missed.base);
    expect(el.querySelector('.retry')).not.toBeNull();
    await answer(true);
    button('Xem kết quả').click();
    await settle();

    const req = http.expectOne('/verbs/results');
    const results = req.request.body.results as { base: string; right: boolean }[];
    expect(results.length).toBe(size);
    expect(results.find((r) => r.base === missed.base)?.right).toBe(false);
    expect(results.filter((r) => r.right).length).toBe(size - 1);
    req.flush({
      verbs: results.map((r) => ({
        base: r.base,
        level: r.right ? 1 : 0,
        dueAt: null,
        right: r.right ? 1 : 0,
        wrong: r.right ? 0 : 1,
      })),
    });
    await settle();

    expect(text(el.querySelector('.summary-title'))).toBe(`Xong ${size} động từ`);
    expect(text(el.querySelector('.summary'))).toContain(`Đúng ngay lần đầu ${size - 1}/${size}`);
    expect(text(el.querySelector('.gain'))).toBe(`+${size - 1} từ thuộc · Đã thuộc ${size - 1}/${size}`);
    const misses = Array.from(el.querySelectorAll('.miss'));
    expect(misses.length).toBe(1);
    expect(text(misses[0])).toContain('Bạn gõ:');
    expect(text(misses[0])).toContain('nope');
  });

  it('keeps the round when saving fails and sends it again', async () => {
    await setup();
    await choose('ABA');
    button('Bắt đầu').click();
    await settle();
    const size = inSet(IRREGULAR_VERBS, 'ABA').length;
    for (let i = 0; i < size; i++) {
      await answer(true);
      button(i === size - 1 ? 'Xem kết quả' : 'Câu tiếp theo').click();
      await settle();
    }
    http.expectOne('/verbs/results').flush({}, { status: 500, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không lưu được kết quả');
    button('Gửi lại').click();
    http.expectOne('/verbs/results').flush({ verbs: [] });
    await settle();
    expect(el.querySelector('[role="alert"]')).toBeNull();
  });

  it('reads the verb and its three forms aloud', async () => {
    await setup();
    await choose('ABA');
    button('Bắt đầu').click();
    await settle();
    const v = verbOnScreen();
    el.querySelector<HTMLButtonElement>('.verb-card .icon-btn')!.click();
    expect(speech.last()).toMatchObject({ text: v.base });
    await answer(true);
    button('Nghe 3 dạng').click();
    expect(speech.last()?.text).toContain(v.base + ', ');
  });

  it('shows an error with a retry when the progress cannot be loaded', async () => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({
      imports: [VerbPractice],
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting(), provideFakeSpeech(speech)],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(VerbPractice);
    el = fixture.nativeElement;
    await fixture.whenStable();
    http.expectOne('/verbs/progress').flush({}, { status: 500, statusText: 'x' });
    await settle();
    expect(text(el.querySelector('[role="alert"]'))).toContain('Không tải được tiến độ');
    button('Thử lại').click();
    http.expectOne('/verbs/progress').flush({ verbs: [] });
    await settle();
    expect(el.querySelectorAll('.set').length).toBe(6);
  });
});
