import { CdkDragDrop } from '@angular/cdk/drag-drop';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { LessonSummary } from '../../../core/models/lesson';
import { Topic, TopicRoadmap } from '../../../core/models/topic';
import { Roadmap } from './roadmap';

const item = (id: string, title: string, inRoadmap = true): LessonSummary => ({
  id,
  title,
  level: 'A1',
  topicId: 't1',
  topicName: 'Gia đình',
  annotationStatus: 'done',
  inRoadmap,
  createdAt: '2026-09-29T08:00:00Z',
});

const topic = (id: string, name: string, level: Topic['level'], roadmapCount: number): Topic => ({
  id, name, level, description: '', lessonCount: roadmapCount + 1, roadmapCount, remaining: roadmapCount,
  warning: roadmapCount < 3, createdAt: '', wordCount: 0, usedWordCount: 0,
});

const topics = [topic('t1', 'Gia đình', 'A1', 3), topic('t2', 'Mua sắm', 'A1', 0), topic('t3', 'Công việc', 'B1', 2)];

const data = (ids: string[]): TopicRoadmap => ({
  topic: topic('t1', 'Gia đình', 'A1', ids.length),
  lessons: ids.map((id) => item(id, `Bài ${id}`)),
  remaining: ids.length,
  warning: ids.length < 3,
});

describe('Roadmap', () => {
  let fixture: ComponentFixture<Roadmap>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let router: Router;

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const titles = () => Array.from(el.querySelectorAll('.roadmap-list .title')).map((t) => text(t));
  const byLabel = (label: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  const flushTopic = async (id: string, roadmap: TopicRoadmap, lessons: LessonSummary[]) => {
    http.expectOne(`/api/admin/topics/${id}/roadmap`).flush(roadmap);
    const list = http.expectOne((r) => r.url === '/api/admin/lessons');
    expect(list.request.params.get('topicId')).toBe(id);
    list.flush({ lessons });
    await settle();
  };
  const expectSave = async (ids: string[], respond: TopicRoadmap | 'error' = data(ids)) => {
    const req = http.expectOne('/api/admin/topics/t1/roadmap');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ lessonIds: ids });
    if (respond === 'error') {
      req.flush({ error: 'internal_error', message: 'Có lỗi xảy ra' }, { status: 500, statusText: 'Error' });
    } else {
      req.flush(respond);
    }
    await settle();
  };

  const setup = async (topicId: string | null, extra: Record<string, string> = {}) => {
    await TestBed.configureTestingModule({
      imports: [Roadmap],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { queryParamMap: convertToParamMap(topicId ? { topicId, ...extra } : extra) } },
        },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture = TestBed.createComponent(Roadmap);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/admin/topics').flush({ topics });
    await settle();
  };

  afterEach(() => http.verify());

  describe('choosing a topic', () => {
    beforeEach(() => setup(null));

    it('offers the topics grouped by level and lists every topic running low', () => {
      const groups = Array.from(el.querySelectorAll<HTMLOptGroupElement>('select[name="topicId"] optgroup'));
      expect(groups.map((g) => g.label)).toEqual(['A1', 'B1']);
      expect(Array.from(el.querySelectorAll('.topic-warnings button')).map((b) => text(b))).toEqual([
        'A1 · Mua sắm: lộ trình chưa có bài',
        'B1 · Công việc: còn 2 bài chưa học',
      ]);
      expect(el.textContent).toContain('Chọn một chủ đề');
    });

    it('loads the roadmap of the chosen topic and remembers it in the URL', async () => {
      const select = el.querySelector<HTMLSelectElement>('select[name="topicId"]')!;
      select.value = 't1';
      select.dispatchEvent(new Event('change'));
      await settle();
      expect(router.navigate).toHaveBeenCalledWith([], expect.objectContaining({ queryParams: { topicId: 't1' } }));
      await flushTopic('t1', data(['a']), [item('a', 'Bài a'), item('x', 'Bài x', false)]);
      expect(titles()).toEqual(['Bài a']);
    });

    it('opens a topic from the warning list', async () => {
      Array.from(el.querySelectorAll<HTMLButtonElement>('.topic-warnings button'))[1].click();
      await settle();
      http.expectOne('/api/admin/topics/t3/roadmap').flush({ ...data([]), topic: topics[2] });
      http.expectOne((r) => r.url === '/api/admin/lessons').flush({ lessons: [] });
      await settle();
      expect(el.querySelector<HTMLSelectElement>('select[name="topicId"]')!.value).toBe('t3');
    });
  });

  describe('editing a roadmap', () => {
    beforeEach(async () => {
      await setup('t1');
      await flushTopic('t1', data(['a', 'b', 'c']), [
        item('a', 'Bài a'), item('b', 'Bài b'), item('c', 'Bài c'), item('x', 'Bài x', false),
      ]);
    });

    it('lists lessons in order with positions and status chips', () => {
      expect(titles()).toEqual(['Bài a', 'Bài b', 'Bài c']);
      expect(el.querySelector('.position')?.textContent?.trim()).toBe('1');
      expect(el.textContent).toContain('Chú thích: Xong');
      expect(el.textContent).not.toContain('Audio');
      expect(el.querySelector('.selected-warning')).toBeNull();
    });

    it('disables moving the first item up and the last item down', () => {
      expect(byLabel('Lên: Bài a').disabled).toBe(true);
      expect(byLabel('Xuống: Bài c').disabled).toBe(true);
      expect(byLabel('Lên: Bài b').disabled).toBe(false);
    });

    it('moves an item with the Up and Down buttons and saves', async () => {
      byLabel('Lên: Bài c').click();
      await settle();
      expect(titles()).toEqual(['Bài a', 'Bài c', 'Bài b']);
      await expectSave(['a', 'c', 'b']);

      byLabel('Xuống: Bài a').click();
      await settle();
      await expectSave(['c', 'a', 'b']);
    });

    it('reorders on drop and saves', async () => {
      const cmp = fixture.componentInstance as unknown as { drop(e: CdkDragDrop<LessonSummary[]>): void };
      cmp.drop({ previousIndex: 2, currentIndex: 0 } as CdkDragDrop<LessonSummary[]>);
      await settle();
      expect(titles()).toEqual(['Bài c', 'Bài a', 'Bài b']);
      await expectSave(['c', 'a', 'b']);
    });

    it('adds a lesson of the topic that is not in the roadmap yet', async () => {
      expect(Array.from(el.querySelectorAll('.available .title')).map((t) => text(t))).toEqual(['Bài x']);
      byLabel('Thêm vào lộ trình: Bài x').click();
      await settle();
      await expectSave(['a', 'b', 'c', 'x']);
      expect(titles()).toEqual(['Bài a', 'Bài b', 'Bài c', 'Bài x']);
      expect(el.querySelector('.available .title')).toBeNull();
    });

    it('removes an item and shows the warning from the response', async () => {
      byLabel('Gỡ khỏi lộ trình: Bài b').click();
      await settle();
      await expectSave(['a', 'c']);
      expect(titles()).toEqual(['Bài a', 'Bài c']);
      expect(text(el.querySelector('.selected-warning'))).toContain('Lộ trình chỉ còn 2 bài chưa học');
      expect(Array.from(el.querySelectorAll('.topic-warnings button')).map((b) => text(b))).toContain(
        'A1 · Gia đình: còn 2 bài chưa học',
      );
      expect(Array.from(el.querySelectorAll('.available .title')).map((t) => text(t))).toEqual(['Bài b', 'Bài x']);
    });

    it('reverts the order and shows an alert when saving fails', async () => {
      byLabel('Lên: Bài b').click();
      await settle();
      expect(titles()).toEqual(['Bài b', 'Bài a', 'Bài c']);
      await expectSave(['b', 'a', 'c'], 'error');
      expect(titles()).toEqual(['Bài a', 'Bài b', 'Bài c']);
      expect(el.querySelector('[role="alert"]')?.textContent).toContain('Không lưu được lộ trình');
    });

    it('keeps focus on the moved item for keyboard users', async () => {
      byLabel('Xuống: Bài a').click();
      await settle();
      await expectSave(['b', 'a', 'c']);
      expect(document.activeElement?.getAttribute('aria-label')).toBe('Xuống: Bài a');

      // At the bottom the Down button is disabled, so focus moves to Up.
      byLabel('Xuống: Bài a').click();
      await settle();
      await expectSave(['b', 'c', 'a']);
      expect(document.activeElement?.getAttribute('aria-label')).toBe('Lên: Bài a');
    });
  });

  it('opens Sinh bài bằng AI right away when asked from the Chủ đề page', async () => {
    HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
      this.setAttribute('open', '');
    });
    await setup('t1', { generate: '1' });
    await flushTopic('t1', data(['a']), [item('a', 'Bài a')]);
    http.expectOne('/api/admin/topics/t1/words').flush({ words: [] });
    await settle();
    http.expectOne((r) => r.url === '/api/admin/grammar').flush({ points: [] });
    await settle();
    expect(el.querySelector('lu-generate-dialog dialog')!.hasAttribute('open')).toBe(true);
  });

  it('warns that an empty roadmap has no lessons', async () => {
    await setup('t1');
    await flushTopic('t1', data([]), []);
    expect(text(el.querySelector('.selected-warning'))).toContain('Lộ trình chưa có bài');
  });

  describe('AI drafts (F7)', () => {
    const generateUrl = '/api/admin/topics/t1/generate';
    const words = (n: number) => Array.from({ length: n }, (_, i) => `w${i}`).join(' ');
    const generated = (...titles: string[]) => ({
      drafts: titles.map((title) => ({ title, content: words(120), words: 120, targetWords: [], missingWords: [] })),
      requested: titles.length,
      dropped: 0,
    });
    const button = (label: string, root: ParentNode = el) =>
      Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);
    const dialog = () => el.querySelector('lu-generate-dialog')!;
    const confirm = () => el.querySelector('lu-confirm-dialog')!;
    const draftTitles = () =>
      Array.from(el.querySelectorAll<HTMLInputElement>('lu-draft-list input[type="text"]')).map((i) => i.value);
    const roadmap = () => fixture.componentInstance as unknown as { canLeave(): boolean | Promise<boolean> };

    const wordsUrl = '/api/admin/topics/t1/words';
    /** Opening loads the topic's words first (F18); by default the topic has none. */
    const gpoint = (id: string, lessonCount: number) => ({
      id, level: 'A1', titleVi: `Điểm ${id}`, titleEn: id, pattern: 'pat', hintVi: 'gợi ý', examples: [], lessonCount,
    });
    const openDialog = async (topicWords: string[] = [], points: object[] = []) => {
      button('Sinh bài bằng AI')!.click();
      await settle();
      http.expectOne(wordsUrl).flush({ words: topicWords.map((text) => ({ text, used: false, lessonCount: 0 })) });
      await settle();
      const grammar = http.expectOne((r) => r.url === '/api/admin/grammar');
      expect(grammar.request.params.get('level')).toBe('A1');
      expect(grammar.request.params.get('topicId')).toBe('t1');
      grammar.flush({ points });
      await settle();
    };
    const submitDialog = async () => {
      dialog().querySelector<HTMLButtonElement>('button[type="submit"]')!.click();
      await settle();
    };
    const generate = async (result: object) => {
      await openDialog();
      await submitDialog();
      http.expectOne(generateUrl).flush(result);
      await settle();
    };
    const expectReload = async (ids: string[]) => {
      await flushTopic('t1', data(ids), ids.map((id) => item(id, `Bài ${id}`)));
    };

    beforeEach(async () => {
      HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
        this.setAttribute('open', '');
      });
      HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
        this.removeAttribute('open');
      });
      await setup('t1');
      await flushTopic('t1', data(['a']), [item('a', 'Bài a')]);
    });

    it('opens the dialog with the topic and the default length of its level', async () => {
      await openDialog();
      expect(dialog().querySelector('dialog')!.hasAttribute('open')).toBe(true);
      expect(text(dialog())).toContain('A1 · Gia đình');
      expect(dialog().querySelector<HTMLInputElement>('#generate-words')!.value).toBe('120');
      expect(dialog().querySelector<HTMLInputElement>('#generate-count')!.value).toBe('3');
    });

    it('generates drafts without touching the roadmap', async () => {
      await openDialog();
      await submitDialog();
      const req = http.expectOne(generateUrl);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ count: 3, words: 120, kind: 'reading', idea: '', targetWords: [], grammarPointId: '' });
      expect(text(button('Đang sinh…', dialog()))).toBe('Đang sinh…');

      req.flush({
        ...generated('Sunday Lunch', 'The Picnic'),
        requested: 3,
        dropped: 1,
        dropReasons: { duplicateTitle: 1, empty: 0, tooLong: 0 },
      });
      await settle();
      expect(dialog().querySelector('dialog')!.hasAttribute('open')).toBe(false);
      expect(draftTitles()).toEqual(['Sunday Lunch', 'The Picnic']);
      expect(text(el.querySelector('.generate-note'))).toBe(
        'Đã thêm 2 bản nháp. Đã loại 1 bản: 1 trùng tiêu đề.',
      );
      expect(titles()).toEqual(['Bài a']);
    });

    it('F23: saves the picture settings chosen in the dialog with each draft', async () => {
      await openDialog();
      dialog().querySelector<HTMLButtonElement>('button[role="switch"]')!.click();
      await settle();
      const style = dialog().querySelector<HTMLTextAreaElement>('#generate-imageStyle')!;
      style.value = 'watercolor';
      style.dispatchEvent(new Event('input'));
      await submitDialog();
      const gen = http.expectOne(generateUrl);
      // The picture settings stay on the page: the generation itself draws nothing.
      expect(gen.request.body.images).toBeUndefined();
      gen.flush(generated('Sunday Lunch'));
      await settle();
      expect(text(el.querySelector('lu-draft-list .images-note'))).toBe('Sẽ sinh ảnh cho từ vựng sau khi lưu.');

      el.querySelector<HTMLButtonElement>('button[aria-label="Lưu bản nháp 1"]')!.click();
      await settle();
      const req = http.expectOne('/api/admin/lessons');
      expect(req.request.body.images).toEqual({ enabled: true, style: 'watercolor' });
      req.flush({ lesson: { id: 'n1' } });
      await settle();
      await expectReload(['a', 'n1']);
    });

    it('saves a draft to the end of the roadmap', async () => {
      await generate(generated('Sunday Lunch', 'The Picnic'));
      const title = el.querySelector<HTMLInputElement>('lu-draft-list input[type="text"]')!;
      title.value = 'Sunday Lunch at Home';
      title.dispatchEvent(new Event('input'));
      await settle();

      el.querySelector<HTMLButtonElement>('button[aria-label="Lưu bản nháp 1"]')!.click();
      await settle();
      const req = http.expectOne('/api/admin/lessons');
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({
        title: 'Sunday Lunch at Home',
        content: words(120),
        topicId: 't1',
        source: 'AI sinh',
        license: 'Nội dung do AI tạo',
        appendToRoadmap: true,
        grammarPointId: '',
      });
      req.flush({ lesson: { id: 'n1' } });
      await settle();
      await expectReload(['a', 'n1']);
      expect(draftTitles()).toEqual(['The Picnic']);
      expect(titles()).toEqual(['Bài a', 'Bài n1']);
    });

    it('generates with the chosen grammar point and saves drafts with it', async () => {
      await openDialog([], [gpoint('g1', 2), gpoint('g2', 0), gpoint('g3', 0)]);
      const select = dialog().querySelector<HTMLSelectElement>('#generate-grammar')!;
      expect(select.value).toBe('');
      select.value = 'g2';
      select.dispatchEvent(new Event('change'));
      await settle();
      await submitDialog();
      const gen = http.expectOne(generateUrl);
      expect(gen.request.body.grammarPointId).toBe('g2');
      gen.flush({
        drafts: [{ title: 'One', content: words(120), words: 120, targetWords: [], missingWords: [], grammarPointId: 'g2' }],
        requested: 1,
        dropped: 0,
      });
      await settle();
      el.querySelector<HTMLButtonElement>('button[aria-label="Lưu bản nháp 1"]')!.click();
      await settle();
      const req = http.expectOne('/api/admin/lessons');
      expect(req.request.body.grammarPointId).toBe('g2');
      req.flush({ lesson: { id: 'n1' } });
      await settle();
      await expectReload(['a', 'n1']);
    });

    it('discards a draft without any request', async () => {
      await generate(generated('Sunday Lunch', 'The Picnic'));
      el.querySelector<HTMLButtonElement>('button[aria-label="Bỏ bản nháp 2"]')!.click();
      await settle();
      expect(draftTitles()).toEqual(['Sunday Lunch']);
    });

    it('saves all drafts one after another and keeps the failed ones', async () => {
      await generate(generated('One', 'Two', 'Three'));
      button('Lưu tất cả')!.click();
      await settle();

      const first = http.expectOne('/api/admin/lessons');
      expect(first.request.body.title).toBe('One');
      first.flush({ lesson: { id: 'n1' } });
      await settle();

      const second = http.expectOne('/api/admin/lessons');
      expect(second.request.body.title).toBe('Two');
      second.flush(
        { error: 'validation_failed', message: 'Dữ liệu không hợp lệ', fields: { title: 'Tối đa 200 ký tự' } },
        { status: 400, statusText: 'Bad Request' },
      );
      await settle();

      const third = http.expectOne('/api/admin/lessons');
      expect(third.request.body.title).toBe('Three');
      third.error(new ProgressEvent('error'));
      await settle();

      await expectReload(['a', 'n1']);
      expect(draftTitles()).toEqual(['Two', 'Three']);
      expect(text(el.querySelector('lu-draft-list .field-error'))).toBe('Tối đa 200 ký tự');
      expect(text(el.querySelector('lu-draft-list [role="alert"]'))).toBe('Không lưu được, vui lòng thử lại.');
    });

    it('sends one request when Lưu is pressed twice', async () => {
      await generate(generated('One'));
      const save = el.querySelector<HTMLButtonElement>('button[aria-label="Lưu bản nháp 1"]')!;
      save.click();
      save.click();
      await settle();
      http.expectOne('/api/admin/lessons').flush({ lesson: { id: 'n1' } });
      await settle();
      await expectReload(['a', 'n1']);
    });

    for (const [name, respond, message] of [
      [
        'quota',
        { status: 429, body: { error: 'ai_quota', message: 'Đã hết lượt AI, vui lòng thử lại sau.' } },
        'Đã hết lượt AI, vui lòng thử lại sau.',
      ],
      [
        'not configured',
        { status: 503, body: { error: 'ai_not_configured', message: 'AI chưa được cấu hình. Liên hệ người vận hành.' } },
        'AI chưa được cấu hình. Liên hệ người vận hành.',
      ],
      [
        'unusable',
        { status: 502, body: { error: 'ai_unusable', message: 'AI trả về nội dung không dùng được, vui lòng thử lại.' } },
        'AI trả về nội dung không dùng được, vui lòng thử lại.',
      ],
      ['network', 'network', 'Sinh bài thất bại, vui lòng thử lại.'],
    ] as const) {
      it(`keeps drafts and options when the AI fails (${name})`, async () => {
        await generate(generated('Kept'));
        const title = el.querySelector<HTMLInputElement>('lu-draft-list input[type="text"]')!;
        title.value = 'Kept and edited';
        title.dispatchEvent(new Event('input'));

        await openDialog();
        const idea = dialog().querySelector<HTMLTextAreaElement>('#generate-idea')!;
        idea.value = 'a picnic';
        idea.dispatchEvent(new Event('input'));
        await submitDialog();
        const req = http.expectOne(generateUrl);
        if (respond === 'network') {
          req.error(new ProgressEvent('error'));
        } else {
          req.flush(respond.body, { status: respond.status, statusText: 'Error' });
        }
        await settle();

        expect(dialog().querySelector('dialog')!.hasAttribute('open')).toBe(true);
        expect(text(dialog().querySelector('[role="alert"]'))).toBe(message);
        expect(dialog().querySelector<HTMLTextAreaElement>('#generate-idea')!.value).toBe('a picnic');
        expect(draftTitles()).toEqual(['Kept and edited']);
      });
    }

    it('passes the topic words to the dialog and sends the suggested target words (F18)', async () => {
      await openDialog(['Family', 'Parents', 'cousin']);
      await settle();
      const plan = http.expectOne((r) => r.url === '/api/admin/topics/t1/word-plan');
      expect(plan.request.params.get('count')).toBe('3');
      expect(plan.request.params.get('perLesson')).toBe('8');
      plan.flush({ groups: [['Family'], ['Parents'], ['cousin']] });
      await settle();
      expect(dialog().querySelector<HTMLInputElement>('#generate-perLesson')!.value).toBe('8');
      expect(dialog().querySelector('button[aria-label="Bỏ Parents khỏi bài 2"]')).toBeTruthy();

      await submitDialog();
      const req = http.expectOne(generateUrl);
      expect(req.request.body.targetWords).toEqual([['Family'], ['Parents'], ['cousin']]);
      req.flush({
        drafts: [
          { title: 'One', content: words(120), words: 120, targetWords: ['Family'], missingWords: [] },
          { title: 'Two', content: words(120), words: 120, targetWords: ['Parents'], missingWords: ['Parents'] },
        ],
        requested: 3,
        dropped: 1,
      });
      await settle();
      const drafts = Array.from(el.querySelectorAll('lu-draft-list li.draft'));
      expect(text(drafts[0].querySelector('.targets-used'))).toBe('Dùng 1/1 từ mục tiêu');
      expect(text(drafts[1].querySelector('.targets-missing'))).toBe('Còn thiếu: Parents');
    });

    it('still opens the dialog, without target words, when the topic words fail to load', async () => {
      button('Sinh bài bằng AI')!.click();
      await settle();
      http.expectOne(wordsUrl).flush({ error: 'internal_error' }, { status: 500, statusText: 'Error' });
      await settle();
      http.expectOne((r) => r.url === '/api/admin/grammar').flush({ points: [] });
      await settle();
      expect(dialog().querySelector('dialog')!.hasAttribute('open')).toBe(true);
      expect(dialog().querySelector('#generate-perLesson')).toBeNull();
    });

    it('appends a new batch after the old drafts', async () => {
      await generate(generated('First'));
      await generate(generated('Second', 'Third'));
      expect(draftTitles()).toEqual(['First', 'Second', 'Third']);
    });

    it('keeps the last options when the dialog opens again', async () => {
      await openDialog();
      const count = dialog().querySelector<HTMLInputElement>('#generate-count')!;
      count.value = '1';
      count.dispatchEvent(new Event('input'));
      await submitDialog();
      http.expectOne(generateUrl).flush(generated('One'));
      await settle();
      await openDialog();
      expect(dialog().querySelector<HTMLInputElement>('#generate-count')!.value).toBe('1');
    });

    it('lets the page go when there are no drafts', () => {
      expect(roadmap().canLeave()).toBe(true);
    });

    it('asks before leaving with drafts', async () => {
      await generate(generated('One'));
      const stay = roadmap().canLeave() as Promise<boolean>;
      await settle();
      expect(text(confirm())).toContain('Các bản nháp chưa lưu sẽ mất.');
      button('Huỷ', confirm())!.click();
      await expect(stay).resolves.toBe(false);
      await settle();
      expect(draftTitles()).toEqual(['One']);

      const leave = roadmap().canLeave() as Promise<boolean>;
      await settle();
      button('Rời trang', confirm())!.click();
      await expect(leave).resolves.toBe(true);
    });

    it('asks before switching topic with drafts', async () => {
      await generate(generated('One'));
      const select = el.querySelector<HTMLSelectElement>('select[name="topicId"]')!;
      select.value = 't3';
      select.dispatchEvent(new Event('change'));
      await settle();
      button('Huỷ', confirm())!.click();
      await settle();
      expect(select.value).toBe('t1');
      expect(draftTitles()).toEqual(['One']);

      select.value = 't3';
      select.dispatchEvent(new Event('change'));
      await settle();
      button('Rời trang', confirm())!.click();
      await settle();
      http.expectOne('/api/admin/topics/t3/roadmap').flush({ ...data([]), topic: topics[2] });
      http.expectOne((r) => r.url === '/api/admin/lessons').flush({ lessons: [] });
      await settle();
      expect(draftTitles()).toEqual([]);
    });

    it('asks the browser before unloading only with drafts', async () => {
      const before = new Event('beforeunload', { cancelable: true });
      window.dispatchEvent(before);
      expect(before.defaultPrevented).toBe(false);

      await generate(generated('One'));
      const after = new Event('beforeunload', { cancelable: true });
      window.dispatchEvent(after);
      expect(after.defaultPrevented).toBe(true);
    });
  });
});
