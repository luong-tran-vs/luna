import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FakeSpeech, provideFakeSpeech } from '../../../core/services/speech.service.testing';
import { IRREGULAR_VERBS } from '../irregular-verbs.data';
import { TypedResult, VerbTypeRound } from './verb-type-round';

describe('VerbTypeRound', () => {
  let fixture: ComponentFixture<VerbTypeRound>;
  let el: HTMLElement;
  let speech: FakeSpeech;
  let finished: TypedResult[] | null;

  const verbs = ['go', 'buy'].map((b) => IRREGULAR_VERBS.find((v) => v.base === b)!);
  const text = (n: Element | null | undefined) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b).startsWith(label))!;
  const settle = async () => {
    await new Promise((r) => setTimeout(r));
    await fixture.whenStable();
  };
  const type = async (past: string, participle: string) => {
    const inputs = el.querySelectorAll<HTMLInputElement>('.field input');
    for (const [i, value] of [past, participle].entries()) {
      inputs[i].value = value;
      inputs[i].dispatchEvent(new Event('input'));
    }
    await settle();
  };

  beforeEach(async () => {
    speech = new FakeSpeech();
    TestBed.configureTestingModule({ imports: [VerbTypeRound], providers: [provideFakeSpeech(speech)] });
    fixture = TestBed.createComponent(VerbTypeRound);
    fixture.componentRef.setInput('verbs', verbs);
    fixture.componentRef.setInput('label', 'Bước 3 · Gõ');
    finished = null;
    fixture.componentInstance.finished.subscribe((r) => (finished = r));
    el = fixture.nativeElement;
    await settle();
  });

  it('grades each form, asks a wrong verb again once at the end, and gives the first answers', async () => {
    expect(text(el.querySelector('.eyebrow'))).toBe('Bước 3 · Gõ');
    expect(text(el.querySelector('#verb-base'))).toBe('go');
    expect(button('Kiểm tra').disabled).toBe(true);

    await type('went', 'goed');
    button('Kiểm tra').click();
    await settle();
    expect(el.querySelectorAll('.field.ok').length).toBe(1);
    expect(el.querySelectorAll('.field.bad').length).toBe(1);
    expect(text(el.querySelector('.feedback'))).toContain('hỏi lại cuối lượt');
    expect(text(el.querySelector('.count'))).toBe('1/3');
    button('Câu tiếp theo').click();
    await settle();

    await type('bought', 'bought');
    button('Kiểm tra').click();
    await settle();
    button('Câu tiếp theo').click();
    await settle();

    expect(text(el.querySelector('#verb-base'))).toBe('go');
    expect(el.querySelector('.retry')).not.toBeNull();
    await type('went', 'gone');
    button('Kiểm tra').click();
    await settle();
    button('Xong').click();
    await settle();

    expect(finished!.map((r) => [r.verb.base, r.right, r.past, r.participle])).toEqual([
      ['go', false, 'went', 'goed'],
      ['buy', true, 'bought', 'bought'],
    ]);
  });

  it('reads the verb and its three forms aloud', async () => {
    el.querySelector<HTMLButtonElement>('.verb-card .icon-btn')!.click();
    expect(speech.last()).toMatchObject({ text: 'go' });
    await type('went', 'gone');
    button('Kiểm tra').click();
    await settle();
    button('Nghe 3 dạng').click();
    expect(speech.last()?.text).toBe('go, went, gone');
  });
});
