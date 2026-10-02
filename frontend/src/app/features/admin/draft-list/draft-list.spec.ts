import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DraftChange, DraftList, DraftState } from './draft-list';

const draft = (key: number, patch: Partial<DraftState> = {}): DraftState => ({
  key,
  title: `Bài ${key}`,
  content: 'We went to the park. It was fun.',
  targetWords: [],
  missingWords: [],
  saving: false,
  error: null,
  fields: {},
  ...patch,
});

describe('DraftList', () => {
  let fixture: ComponentFixture<DraftList>;
  let el: HTMLElement;
  let changes: DraftChange[];
  let saved: number[];
  let discarded: number[];
  let savedAll: number;

  const render = async (drafts: DraftState[], busy = false) => {
    fixture.componentRef.setInput('drafts', drafts);
    fixture.componentRef.setInput('busy', busy);
    await fixture.whenStable();
  };
  const button = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  const byText = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => b.textContent?.trim() === label);

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [DraftList] }).compileComponents();
    fixture = TestBed.createComponent(DraftList);
    el = fixture.nativeElement;
    changes = [];
    saved = [];
    discarded = [];
    savedAll = 0;
    const c = fixture.componentInstance;
    c.changed.subscribe((v) => changes.push(v));
    c.save.subscribe((k) => saved.push(k));
    c.discard.subscribe((k) => discarded.push(k));
    c.saveAll.subscribe(() => savedAll++);
  });

  it('renders nothing without drafts', async () => {
    await render([]);
    expect(el.querySelector('section')).toBeNull();
  });

  it('shows how many target words each draft uses and which are missing (F18)', async () => {
    await render([
      draft(1, { targetWords: ['park', 'fun', 'picnic', 'cousin'], missingWords: ['picnic', 'cousin'] }),
      draft(2, { targetWords: ['park'], missingWords: [] }),
      draft(3),
    ]);
    const items = Array.from(el.querySelectorAll('li.draft'));
    const text = (n: Element | null) => n?.textContent?.replace(/\s+/g, ' ').trim() ?? null;
    expect(text(items[0].querySelector('.targets-used'))).toBe('Dùng 2/4 từ mục tiêu');
    expect(text(items[0].querySelector('.targets-missing'))).toBe('Còn thiếu: picnic, cousin');
    expect(text(items[1].querySelector('.targets-used'))).toBe('Dùng 1/1 từ mục tiêu');
    expect(items[1].querySelector('.targets-missing')).toBeNull();
    expect(items[2].querySelector('.targets')).toBeNull();
  });

  it('shows editable title and content with labels and word count', async () => {
    await render([draft(1), draft(2, { content: 'Anna: Hi!\nBen: Hello there.' })]);
    const title = el.querySelector<HTMLInputElement>('#draft-1-title')!;
    expect(el.querySelector('label[for="draft-1-title"]')?.textContent?.trim()).toBe('Tiêu đề bản nháp 1');
    expect(el.querySelector('label[for="draft-2-content"]')?.textContent?.trim()).toBe('Nội dung bản nháp 2');
    expect(title.value).toBe('Bài 1');
    expect(el.querySelector('#draft-1-words')?.textContent?.trim()).toBe('8 từ');
    expect(el.querySelector('#draft-2-words')?.textContent?.trim()).toBe('5 từ');
    expect(el.querySelector('#draft-1-content')?.getAttribute('aria-describedby')).toBe('draft-1-words');

    title.value = 'Mới';
    title.dispatchEvent(new Event('input'));
    const content = el.querySelector<HTMLTextAreaElement>('#draft-2-content')!;
    content.value = 'One two';
    content.dispatchEvent(new Event('input'));
    expect(changes).toEqual([
      { key: 1, title: 'Mới' },
      { key: 2, content: 'One two' },
    ]);
  });

  it('emits Lưu, Bỏ and Lưu tất cả', async () => {
    await render([draft(1), draft(7)]);
    button('Lưu bản nháp 2').click();
    button('Bỏ bản nháp 1').click();
    byText('Lưu tất cả')!.click();
    expect(saved).toEqual([7]);
    expect(discarded).toEqual([1]);
    expect(savedAll).toBe(1);
  });

  it('offers Lưu tất cả only with two drafts or more', async () => {
    await render([draft(1)]);
    expect(byText('Lưu tất cả')).toBeUndefined();
  });

  it('locks a draft while it is saving', async () => {
    await render([draft(1, { saving: true }), draft(2)]);
    const saving = button('Đang lưu bản nháp 1');
    expect(saving.textContent?.trim()).toBe('Đang lưu…');
    expect(saving.disabled).toBe(true);
    expect(saving.getAttribute('aria-busy')).toBe('true');
    expect(button('Bỏ bản nháp 1').disabled).toBe(true);
    expect(el.querySelector<HTMLInputElement>('#draft-1-title')!.disabled).toBe(true);
    expect(button('Lưu bản nháp 2').disabled).toBe(false);
  });

  it('disables saving while another save is running', async () => {
    await render([draft(1), draft(2)], true);
    expect(button('Lưu bản nháp 2').disabled).toBe(true);
    expect(byText('Lưu tất cả')!.disabled).toBe(true);
  });

  it('shows field and save errors', async () => {
    await render([
      draft(1, { fields: { title: 'Vui lòng nhập tiêu đề' } }),
      draft(2, { error: 'Không lưu được, vui lòng thử lại.' }),
    ]);
    const title = el.querySelector('#draft-1-title')!;
    expect(title.getAttribute('aria-invalid')).toBe('true');
    expect(title.getAttribute('aria-describedby')).toBe('draft-1-title-error');
    expect(el.querySelector('#draft-1-title-error')?.textContent?.trim()).toBe('Vui lòng nhập tiêu đề');
    const alerts = el.querySelectorAll('[role="alert"]');
    expect(alerts.length).toBe(1);
    expect(alerts[0].textContent?.trim()).toBe('Không lưu được, vui lòng thử lại.');
  });
});
