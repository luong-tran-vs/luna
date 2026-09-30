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
  audioStatus: 'done',
  annotationStatus: 'done',
  inRoadmap,
  createdAt: '2026-09-29T08:00:00Z',
});

const topic = (id: string, name: string, level: Topic['level'], roadmapCount: number): Topic => ({
  id, name, level, description: '', lessonCount: roadmapCount + 1, roadmapCount, remaining: roadmapCount,
  warning: roadmapCount < 3, createdAt: '',
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

  const setup = async (topicId: string | null) => {
    await TestBed.configureTestingModule({
      imports: [Roadmap],
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { queryParamMap: convertToParamMap(topicId ? { topicId } : {}) } },
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
      expect(el.textContent).toContain('Audio: Xong');
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

  it('warns that an empty roadmap has no lessons', async () => {
    await setup('t1');
    await flushTopic('t1', data([]), []);
    expect(text(el.querySelector('.selected-warning'))).toContain('Lộ trình chưa có bài');
  });
});
