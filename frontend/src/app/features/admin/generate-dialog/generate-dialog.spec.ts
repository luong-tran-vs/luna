import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { TopicWord } from '../../../core/models/topic';
import { GenerateDialog, GenerateOptions, GenerateRequest, PLAN_DEBOUNCE_MS } from './generate-dialog';

describe('GenerateDialog', () => {
  let fixture: ComponentFixture<GenerateDialog>;
  let el: HTMLElement;
  let emitted: GenerateRequest[];
  let http: HttpTestingController;
  let closed: number;

  const options: GenerateOptions = {
    count: 3,
    words: 120,
    kind: 'reading',
    idea: '',
    perLesson: 8,
    images: false,
    imageStyle: 'flat illustration',
  };

  beforeEach(async () => {
    // jsdom has no showModal/close.
    HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
      this.setAttribute('open', '');
    });
    HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
      this.removeAttribute('open');
    });

    await TestBed.configureTestingModule({
      imports: [GenerateDialog],
      providers: [provideHttpClient(withInterceptors([errorInterceptor])), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(GenerateDialog);
    fixture.componentRef.setInput('topicLabel', 'A1 · Gia đình');
    fixture.componentRef.setInput('options', options);
    fixture.componentRef.setInput('open', true);
    emitted = [];
    closed = 0;
    fixture.componentInstance.generate.subscribe((v) => emitted.push(v));
    fixture.componentInstance.closed.subscribe(() => closed++);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  const input = (id: string) => el.querySelector<HTMLInputElement | HTMLTextAreaElement>(`#${id}`)!;
  const type = async (id: string, value: string) => {
    const field = input(id);
    field.value = value;
    field.dispatchEvent(new Event('input'));
    await fixture.whenStable();
  };
  const submit = async () => {
    el.querySelector<HTMLButtonElement>('button[type="submit"]')!.click();
    await fixture.whenStable();
  };
  const errorOf = (field: string) => el.querySelector(`#generate-${field}-error`)?.textContent?.trim();

  it('opens as a modal with the topic and the given options', () => {
    const dialog = el.querySelector('dialog')!;
    expect(dialog.showModal).toHaveBeenCalled();
    expect(dialog.hasAttribute('open')).toBe(true);
    expect(el.querySelector('#generate-title')?.textContent).toContain('Sinh bài bằng AI');
    expect(el.textContent).toContain('A1 · Gia đình');
    expect(input('generate-count').value).toBe('3');
    expect(input('generate-words').value).toBe('120');
    expect(el.querySelector<HTMLInputElement>('input[value="reading"]')!.checked).toBe(true);
    expect(input('generate-idea').value).toBe('');
    // A short hint: just the range, no explanation.
    expect(el.querySelector('#generate-words-hint')!.textContent!.trim()).toBe('Nên 96–144 từ.');
  });

  it('F23: shows the picture style only when pictures are on, and emits both', async () => {
    const toggle = el.querySelector<HTMLButtonElement>('button[role="switch"]')!;
    expect(toggle.getAttribute('aria-checked')).toBe('false');
    expect(toggle.textContent?.trim()).toBe('Sinh ảnh cho từ vựng');
    expect(el.querySelector('#generate-imageStyle')).toBeNull();
    toggle.click();
    await fixture.whenStable();
    expect(toggle.getAttribute('aria-checked')).toBe('true');
    expect(input('generate-imageStyle').value).toBe('flat illustration');
    await type('generate-imageStyle', '  tranh màu nước  ');
    await submit();
    expect(emitted[0].images).toBe(true);
    expect(emitted[0].imageStyle).toBe('tranh màu nước');
  });

  it('F23: refuses a picture style that is too long', async () => {
    el.querySelector<HTMLButtonElement>('button[role="switch"]')!.click();
    await fixture.whenStable();
    await type('generate-imageStyle', 'x'.repeat(501));
    await submit();
    expect(emitted).toEqual([]);
    expect(errorOf('imageStyle')).toBe('Mô tả ảnh tối đa 500 ký tự');
  });

  it('emits trimmed valid values', async () => {
    await type('generate-count', '2');
    await type('generate-words', '200');
    el.querySelector<HTMLInputElement>('input[value="dialogue"]')!.click();
    await type('generate-idea', '  một bữa tiệc  ');
    await submit();
    expect(emitted).toEqual([
      {
        level: 'A1',
        count: 2,
        words: 200,
        kind: 'dialogue',
        idea: 'một bữa tiệc',
        targetWords: [],
        grammarPointId: '',
        perLesson: 8,
        images: false,
        imageStyle: 'flat illustration',
      },
    ]);
    // A topic without words shows no target-word field and asks for no split.
    expect(el.querySelector('#generate-perLesson')).toBeNull();
  });

  for (const [field, value, message] of [
    ['count', '0', 'Số bài từ 1 đến 10'],
    ['count', '11', 'Số bài từ 1 đến 10'],
    ['count', '2.5', 'Số bài từ 1 đến 10'],
    ['words', '49', 'Độ dài từ 50 đến 800 từ'],
    ['words', '801', 'Độ dài từ 50 đến 800 từ'],
    ['idea', 'x'.repeat(501), 'Ý chính tối đa 500 ký tự'],
  ]) {
    it(`rejects ${field} = ${value.slice(0, 10)}`, async () => {
      await type(`generate-${field}`, value);
      await submit();
      expect(emitted).toEqual([]);
      expect(errorOf(field)).toBe(message);
      expect(input(`generate-${field}`).getAttribute('aria-invalid')).toBe('true');
      expect(input(`generate-${field}`).getAttribute('aria-describedby')).toContain(`generate-${field}-error`);
    });
  }

  it('locks everything while generating and ignores Escape', async () => {
    fixture.componentRef.setInput('busy', true);
    await fixture.whenStable();
    const button = el.querySelector<HTMLButtonElement>('button[type="submit"]')!;
    expect(button.textContent?.trim()).toBe('Đang sinh…');
    expect(button.disabled).toBe(true);
    expect(button.getAttribute('aria-busy')).toBe('true');
    expect(input('generate-count').disabled).toBe(true);
    expect(input('generate-idea').disabled).toBe(true);

    const cancel = new Event('cancel', { cancelable: true });
    el.querySelector('dialog')!.dispatchEvent(cancel);
    expect(cancel.defaultPrevented).toBe(true);
    expect(closed).toBe(0);
  });

  it('shows the error and keeps the values', async () => {
    await type('generate-idea', 'picnic');
    fixture.componentRef.setInput('busy', true);
    await fixture.whenStable();
    fixture.componentRef.setInput('busy', false);
    fixture.componentRef.setInput('error', 'Đã hết lượt AI, vui lòng thử lại sau.');
    await fixture.whenStable();
    expect(el.querySelector('[role="alert"]')?.textContent?.trim()).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
    expect(input('generate-idea').value).toBe('picnic');
    expect(input('generate-idea').disabled).toBe(false);
  });

  it('closes on Huỷ and on Escape', async () => {
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === 'Huỷ')!.click();
    const cancel = new Event('cancel', { cancelable: true });
    el.querySelector('dialog')!.dispatchEvent(cancel);
    expect(closed).toBe(2);

    fixture.componentRef.setInput('open', false);
    await fixture.whenStable();
    expect(el.querySelector('dialog')!.hasAttribute('open')).toBe(false);
  });

  it('resets to the options when opened again', async () => {
    await type('generate-count', '5');
    fixture.componentRef.setInput('open', false);
    await fixture.whenStable();
    fixture.componentRef.setInput('options', { count: 1, words: 300, kind: 'dialogue', idea: 'x', perLesson: 8 });
    fixture.componentRef.setInput('open', true);
    await fixture.whenStable();
    expect(input('generate-count').value).toBe('1');
    expect(input('generate-words').value).toBe('300');
    expect(el.querySelector<HTMLInputElement>('input[value="dialogue"]')!.checked).toBe(true);
  });

  describe('grammar point', () => {
    const point = (id: string, lessonCount: number) => ({
      id, level: 'A1', titleVi: `Điểm ${id}`, titleEn: id, pattern: `mẫu ${id}`, hintVi: `gợi ý ${id}.`, examples: [], lessonCount,
    });
    const grammarUrl = '/admin/grammar';
    const settle = async () => {
      await new Promise((resolve) => setTimeout(resolve));
      await fixture.whenStable();
    };
    const select = () => el.querySelector<HTMLSelectElement>('#generate-grammar');
    const reopen = async () => {
      fixture.componentRef.setInput('level', 'A1');
      fixture.componentRef.setInput('topicId', 't1');
      fixture.componentRef.setInput('open', false);
      await fixture.whenStable();
      fixture.componentRef.setInput('open', true);
      await fixture.whenStable();
    };
    const respond = async (points: object[]) => {
      const req = http.expectOne((r) => r.url === grammarUrl);
      expect(req.request.params.get('level')).toBe('A1');
      expect(req.request.params.get('topicId')).toBe('t1');
      req.flush({ points });
      await settle();
    };

    it('defaults to AI tự chọn, and shows the hint of a point once chosen', async () => {
      await reopen();
      await respond([point('g1', 2), point('g2', 1), point('g3', 1)]);
      expect(select()!.value).toBe('');
      expect(el.querySelector('#generate-grammar-hint')!.textContent).toContain('AI tự chọn');
      select()!.value = 'g2';
      select()!.dispatchEvent(new Event('change'));
      await settle();
      expect(el.querySelector('label[for="generate-grammar"]')?.textContent).toContain('Điểm ngữ pháp');
      expect(Array.from(select()!.options).map((o) => o.textContent?.trim())).toEqual([
        'AI tự chọn', 'Điểm g1 (2 bài)', 'Điểm g2 (1 bài)', 'Điểm g3 (1 bài)',
      ]);
      const hint = el.querySelector('#generate-grammar-hint')!.textContent!;
      expect(hint).toContain('mẫu g2');
      expect(hint).toContain('gợi ý g2.');
      expect(hint).toContain('mọi bài trong lượt sinh');
      expect(select()!.getAttribute('aria-describedby')).toBe('generate-grammar-hint');
    });

    it('emits the chosen point, or an empty one for AI tự chọn', async () => {
      await reopen();
      await respond([point('g1', 0), point('g2', 1)]);
      await submit();
      expect(emitted[0].grammarPointId).toBe('');
      select()!.value = 'g1';
      select()!.dispatchEvent(new Event('change'));
      await settle();
      await submit();
      expect(emitted[1].grammarPointId).toBe('g1');
    });

    it('still generates when the list cannot be loaded', async () => {
      await reopen();
      http.expectOne((r) => r.url === grammarUrl).flush({}, { status: 500, statusText: 'Error' });
      await settle();
      expect(select()).toBeNull();
      expect(el.textContent).toContain('Không tải được điểm ngữ pháp');
      await submit();
      expect(emitted.length).toBe(1);
      expect(emitted[0].grammarPointId).toBe('');
    });
  });

  describe('target words (F18)', () => {
    const topicWords = ['Family', 'Parents', 'cousin', 'take a shower'];
    /** Topic words for every level unless given as "word@B1". */
    const words = (...texts: string[]): TopicWord[] =>
      texts.map((t) => {
        const [text, level] = t.split('@');
        return { text, level: (level ?? '') as TopicWord['level'], used: false, lessonCount: 0 };
      });
    const planUrl = '/admin/topics/t1/word-plan';
    const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
    const settle = async () => {
      await wait(0);
      await fixture.whenStable();
    };
    const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
    const byLabel = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
    const groups = () =>
      Array.from(el.querySelectorAll('.group')).map((g) =>
        Array.from(g.querySelectorAll('.chip span')).map((c) => text(c)),
      );
    const expectPlan = async (count: number, perLesson: number, respond: string[][] | 'error', shortage = 0) => {
      await settle(); // the request leaves on the next tick (timer)
      const req = http.expectOne((r) => r.url === planUrl);
      expect(req.request.params.get('count')).toBe(String(count));
      expect(req.request.params.get('perLesson')).toBe(String(perLesson));
      if (respond === 'error') {
        req.flush({ error: 'internal_error' }, { status: 500, statusText: 'Error' });
      } else {
        req.flush({ groups: respond, shortage });
      }
      await settle();
    };
    const addTo = async (group: number, value: string, how: 'enter' | 'button' = 'enter') => {
      const box = input(`generate-add-${group}`) as HTMLInputElement;
      box.value = value;
      if (how === 'enter') {
        box.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', cancelable: true }));
      } else {
        Array.from(box.parentElement!.querySelectorAll('button'))
          .find((b) => text(b).startsWith('Thêm'))!
          .click();
      }
      await settle();
    };

    beforeEach(async () => {
      fixture.componentRef.setInput('open', false);
      await fixture.whenStable();
      fixture.componentRef.setInput('topicId', 't1');
      fixture.componentRef.setInput('topicWords', words(...topicWords));
      fixture.componentRef.setInput('options', { ...options, count: 2 });
      fixture.componentRef.setInput('open', true);
      await settle();
      // With a topic the grammar points of the level are offered too.
      http.expectOne((r) => r.url === '/admin/grammar').flush({ points: [] });
      await settle();
    });

    it('shows the words per lesson and the suggested groups', async () => {
      expect(input('generate-perLesson').value).toBe('8');
      expect(text(el.querySelector('.targets lu-loading'))).toBe('Đang chia từ…');
      await expectPlan(2, 8, [['Family', 'Parents'], ['cousin']]);
      expect(Array.from(el.querySelectorAll('.group legend')).map((l) => text(l))).toEqual([
        'Bài 1 (2 từ)',
        'Bài 2 (1 từ)',
      ]);
      expect(groups()).toEqual([['Family', 'Parents'], ['cousin']]);
      // The add box offers the topic words not yet in the group.
      const offered = Array.from(el.querySelectorAll<HTMLOptionElement>('#generate-add-1-options option')).map(
        (o) => o.value,
      );
      expect(offered).toEqual(['Family', 'Parents', 'take a shower']);

      await submit();
      expect(emitted[0].targetWords).toEqual([['Family', 'Parents'], ['cousin']]);
      expect(emitted[0].perLesson).toBe(8);
    });

    it('removes and adds words, rejecting unknown and duplicate ones', async () => {
      await expectPlan(2, 8, [['Family', 'Parents'], ['cousin']]);
      byLabel('Bỏ Parents khỏi bài 1')!.click();
      await settle();
      expect(groups()[0]).toEqual(['Family']);
      expect(document.activeElement?.id).toBe('generate-add-0');

      await addTo(0, 'grandma');
      expect(text(el.querySelector('#generate-add-0-error'))).toBe('Từ không có trong danh sách');
      expect(input('generate-add-0').getAttribute('aria-invalid')).toBe('true');
      expect(emitted).toEqual([]);

      await addTo(0, ' family ');
      expect(text(el.querySelector('#generate-add-0-error'))).toBe('Từ đã có trong bài này');

      await addTo(0, 'TAKE A SHOWER');
      expect(el.querySelector('#generate-add-0-error')).toBeNull();
      expect(groups()[0]).toEqual(['Family', 'take a shower']);
      expect(input('generate-add-0').value).toBe('');

      await addTo(1, 'parents', 'button');
      expect(groups()[1]).toEqual(['cousin', 'Parents']);

      await submit();
      expect(emitted[0].targetWords).toEqual([['Family', 'take a shower'], ['cousin', 'Parents']]);
    });

    it('limits a lesson to 15 words', async () => {
      const many = Array.from({ length: 16 }, (_, i) => `word${String.fromCharCode(97 + i)}`);
      fixture.componentRef.setInput('topicWords', words(...many));
      await expectPlan(2, 8, [many.slice(0, 15), []]);
      await addTo(0, many[15]);
      expect(text(el.querySelector('#generate-add-0-error'))).toBe('Tối đa 15 từ mỗi bài');
    });

    it('asks for a new split when the counts change, keeping only the latest answer', async () => {
      await expectPlan(2, 8, [['Family'], ['Parents']]);
      await type('generate-count', '3');
      await type('generate-count', '4');
      await wait(PLAN_DEBOUNCE_MS + 50);
      await fixture.whenStable();
      // Debounced: one request for the last value only.
      await expectPlan(4, 8, [['Family'], ['Parents'], ['cousin'], ['take a shower']]);
      expect(groups()).toEqual([['Family'], ['Parents'], ['cousin'], ['take a shower']]);

      await type('generate-perLesson', '2');
      await wait(PLAN_DEBOUNCE_MS + 50);
      const stale = http.expectOne((r) => r.url === planUrl);
      await type('generate-perLesson', '1');
      await wait(PLAN_DEBOUNCE_MS + 50);
      expect(stale.cancelled).toBe(true);
      await expectPlan(4, 1, [['Family'], ['Parents'], ['cousin'], ['take a shower']]);
    });

    it('with 0 words per lesson shows no groups and sends no target words', async () => {
      await expectPlan(2, 8, [['Family'], ['Parents']]);
      await type('generate-perLesson', '0');
      await wait(PLAN_DEBOUNCE_MS + 50);
      await fixture.whenStable();
      expect(el.querySelectorAll('.group').length).toBe(0);
      await submit();
      expect(emitted[0].targetWords).toEqual([]);
      expect(emitted[0].perLesson).toBe(0);
    });

    it('rejects words per lesson out of range', async () => {
      await expectPlan(2, 8, [['Family'], ['Parents']]);
      await type('generate-perLesson', '16');
      await submit();
      expect(emitted).toEqual([]);
      expect(text(el.querySelector('#generate-perLesson-error'))).toBe('Số từ mục tiêu từ 0 đến 15');
    });

    it('offers to retry when the split fails', async () => {
      await expectPlan(2, 8, 'error');
      expect(text(el.querySelector('.plan-error'))).toContain('Không chia được từ, thử lại.');
      el.querySelector<HTMLButtonElement>('.plan-error button')!.click();
      await settle();
      await expectPlan(2, 8, [['Family'], ['Parents']]);
      expect(groups()).toEqual([['Family'], ['Parents']]);
    });

    it('offers to add the missing words with AI, then splits again', async () => {
      const changed: string[][] = [];
      fixture.componentInstance.wordsChanged.subscribe((w) => changed.push(w.map((x) => x.text)));
      await expectPlan(2, 8, [['Family', 'Parents'], ['cousin']], 12);
      expect(text(el.querySelector('.shortage'))).toContain('Chủ đề thiếu 12 từ chưa dùng');

      el.querySelector<HTMLButtonElement>('.shortage button')!.click();
      await settle();
      const req = http.expectOne('/admin/topics/t1/words/suggest');
      expect(req.request.body).toEqual({ count: 12, level: 'A1' });
      expect(text(el.querySelector('.shortage button'))).toBe('Đang bổ sung…');
      req.flush({ added: ['aunt', 'uncle'], words: words(...topicWords, 'aunt@A1', 'uncle@A1') });
      await settle();

      expect(changed).toEqual([[...topicWords, 'aunt', 'uncle']]);
      expect(text(el.querySelector('.suggest-note'))).toBe('Đã thêm 2 từ: aunt, uncle.');
      await expectPlan(2, 8, [['aunt', 'Family'], ['uncle', 'cousin']]);
      expect(groups()).toEqual([['aunt', 'Family'], ['uncle', 'cousin']]);
      expect(el.querySelector('.shortage')).toBeNull();
    });

    it('shows the server message when the AI cannot add words', async () => {
      await expectPlan(2, 8, [['Family'], ['Parents']], 3);
      el.querySelector<HTMLButtonElement>('.shortage button')!.click();
      await settle();
      http
        .expectOne('/admin/topics/t1/words/suggest')
        .flush({ error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' }, { status: 429, statusText: 'Too Many' });
      await settle();
      expect(text(el.querySelector('.suggest-note.error'))).toBe('Đã hết lượt AI, vui lòng thử lại sau.');
      expect(el.querySelector<HTMLButtonElement>('.shortage button')!.disabled).toBe(false);
    });

    it('keeps the edited groups after a generation error', async () => {
      await expectPlan(2, 8, [['Family', 'Parents'], ['cousin']]);
      byLabel('Bỏ Parents khỏi bài 1')!.click();
      await settle();
      await submit();
      expect(emitted.length).toBe(1);
      fixture.componentRef.setInput('busy', true);
      await fixture.whenStable();
      expect(byLabel('Bỏ Family khỏi bài 1')!.disabled).toBe(true);
      fixture.componentRef.setInput('busy', false);
      fixture.componentRef.setInput('error', 'Đã hết lượt AI, vui lòng thử lại sau.');
      await wait(PLAN_DEBOUNCE_MS + 50);
      await fixture.whenStable();
      // No new split was asked for (http.verify), and the edits are still there.
      expect(groups()).toEqual([['Family'], ['cousin']]);
    });
  });
});
