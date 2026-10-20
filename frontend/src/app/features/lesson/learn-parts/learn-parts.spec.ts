import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { LearningPath, Milestone, PathRoadmap } from '../../../core/models/study';
import { LearnParts } from './learn-parts';

const roadmap = (topicId: string, completed: number, total: number): PathRoadmap => ({
  topicId,
  topicName: topicId,
  active: false,
  completed,
  lessons: Array.from({ length: total }, (_, i) => ({
    id: `${topicId}${i}`,
    title: 'Bài',
    state: i < completed ? 'done' : 'locked',
  })),
});

const part = (level: Milestone['level'], roadmaps: PathRoadmap[], active = false): Milestone => ({
  level,
  active,
  completed: roadmaps.reduce((n, r) => n + r.completed, 0),
  total: roadmaps.reduce((n, r) => n + r.lessons.length, 0),
  roadmaps,
});

const path: LearningPath = {
  topicId: 'family',
  topicName: 'family',
  streak: 3,
  milestones: [
    part('A1', [{ ...roadmap('family', 1, 4), active: true }], true),
    part('A2', [roadmap('travel', 0, 3), roadmap('work', 0, 2)]),
    part('B1', []),
  ],
};

describe('LearnParts', () => {
  let fixture: ComponentFixture<LearnParts>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

  const setup = async (data: LearningPath) => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(LearnParts);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/lessons/path').flush(data);
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };

  afterEach(() => http.verify());

  it('lists one card per part with lessons, each opening its path', async () => {
    await setup(path);

    expect(text(el.querySelector('h1'))).toBe('Chọn trình độ');
    expect(text(el.querySelector('.streak-chip'))).toBe('3 ngày liên tiếp');
    // "Trở về" leads home.
    expect(el.querySelector('a.back')?.getAttribute('href')).toBe('/');
    const parts = Array.from(el.querySelectorAll('.part'));
    expect(parts.map((p) => text(p.querySelector('h2')))).toEqual([
      'Trình độ A1 · Mới bắt đầu',
      'Trình độ A2 · Sơ cấp',
    ]);

    // The part being studied: its progress and "Tiếp tục".
    expect(parts[0].classList).toContain('is-active');
    expect(text(parts[0].querySelector('.chip'))).toBe('Đang học');
    expect(text(parts[0].querySelector('.badge'))).toBe('1/4 bài');
    expect(parts[0].querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe(
      '25',
    );
    expect(text(parts[0].querySelector('.percent'))).toBe('25%');
    const resume = parts[0].querySelector('a')!;
    expect(text(resume)).toBe('Tiếp tục học A1');
    expect(resume.getAttribute('href')).toBe('/lessons/levels/A1');

    // Another part: locked chip, lessons left, and a jump to it.
    expect(text(parts[1].querySelector('.chip'))).toBe('Chưa mở');
    expect(text(parts[1].querySelector('.badge'))).toBe('5 bài cần mở');
    expect(text(parts[1].querySelector('.part-meta'))).toBe(
      'i Học xong trình độ trước hoặc chuyển thẳng sang đây',
    );
    expect(text(parts[1].querySelector('a'))).toBe('Chuyển sang A2');
    expect(parts[1].querySelector('a')?.getAttribute('href')).toBe('/lessons/levels/A2');

    // The mascot, served from public/images.
    expect(parts[0].querySelector('img.mascot')?.getAttribute('src')).toContain(
      'images/mascot-cat.png',
    );

    // No lesson yet: no card, one small row in "Sắp có" with nothing to open.
    const soon = el.querySelector('.soon')!;
    expect(text(soon.querySelector('h2'))).toBe('Sắp có · 1 trình độ');
    const rows = Array.from(soon.querySelectorAll('li'));
    expect(rows.map((r) => text(r))).toEqual([
      'B1 Trung cấp Tự Tin Làm Chủ: Phản xạ nhanh & từ vựng nâng cao',
    ]);
    expect(soon.querySelector('a, img, .chip')).toBeNull();
    // The cards come first, then "Sắp có", then the daily streak card.
    const daily = el.querySelector('.daily')!;
    expect(
      el.querySelector('.list')!.compareDocumentPosition(soon) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(soon.compareDocumentPosition(daily) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(text(el.querySelector('.daily-title'))).toBe('Kiên trì mỗi ngày 3 ngày');
  });

  it('has no "Sắp có" box when every part has lessons', async () => {
    await setup({ ...path, milestones: path.milestones.slice(0, 2) });
    expect(el.querySelectorAll('.part')).toHaveLength(2);
    expect(el.querySelector('.soon')).toBeNull();
  });

  it('asks to pick a part when there is no goal', async () => {
    await setup({
      ...path,
      topicId: '',
      topicName: '',
      milestones: [part('A1', [roadmap('family', 0, 4)])],
    });

    expect(text(el.querySelector('[role="status"]'))).toContain('Bạn chưa chọn chủ đề nào');
    expect(text(el.querySelector('.part a'))).toBe('Chuyển sang A1');
    expect(el.querySelector('a.back')?.getAttribute('href')).toBe('/');
  });
});
