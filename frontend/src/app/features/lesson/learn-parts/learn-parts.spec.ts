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

  it('lists one card per part, each opening its path', async () => {
    await setup(path);

    expect(text(el.querySelector('h1'))).toBe('Chọn chặng học');
    expect(text(el.querySelector('.streak-chip'))).toBe('3 ngày liên tiếp');
    // "Trở về" leads home.
    expect(el.querySelector('a.back')?.getAttribute('href')).toBe('/');
    const parts = Array.from(el.querySelectorAll('.part'));
    expect(parts.map((p) => text(p.querySelector('h2')))).toEqual([
      'Phần 1: Khởi Động · A1 Mới bắt đầu',
      'Phần 2: Giao Tiếp · A2 Sơ cấp',
      'Phần 3: Tự Tin Làm Chủ · B1 Trung cấp',
    ]);

    // The part being studied: its progress and "Tiếp tục".
    expect(parts[0].classList).toContain('is-active');
    expect(text(parts[0].querySelector('.chip'))).toBe('Đang học · Chặng 1/3');
    expect(text(parts[0].querySelector('.badge'))).toBe('1/4 bài');
    expect(parts[0].querySelector('[role="progressbar"]')?.getAttribute('aria-valuenow')).toBe(
      '25',
    );
    expect(text(parts[0].querySelector('.percent'))).toBe('25%');
    const resume = parts[0].querySelector('a')!;
    expect(text(resume)).toBe('Tiếp tục học phần 1');
    expect(resume.getAttribute('href')).toBe('/lessons/levels/A1');

    // Another part: locked chip, lessons left, and a jump to it.
    expect(text(parts[1].querySelector('.chip'))).toBe('Khóa');
    expect(text(parts[1].querySelector('.stage'))).toBe('Chặng 2 / 3');
    expect(text(parts[1].querySelector('.badge'))).toBe('5 bài cần mở');
    expect(text(parts[1].querySelector('.part-meta'))).toBe(
      'i Hoàn thành Phần 1 hoặc nhảy thẳng tới đây',
    );
    expect(text(parts[1].querySelector('a'))).toBe('Nhảy tới phần 2');
    expect(parts[1].querySelector('a')?.getAttribute('href')).toBe('/lessons/levels/A2');

    // The mascot, served from public/images.
    expect(parts[0].querySelector('img.mascot')?.getAttribute('src')).toContain(
      'images/mascot-cat.png',
    );

    // No lesson yet: locked, nothing to open.
    expect(parts[2].classList).toContain('is-locked');
    expect(text(parts[2].querySelector('.part-meta'))).toBe('i Phần này chưa có bài học');
    expect(text(parts[2].querySelector('.badge'))).toBe('Sắp có');

    // The daily streak card.
    expect(text(el.querySelector('.daily-title'))).toBe('Kiên trì mỗi ngày 3 ngày');
    expect(parts[2].querySelector('a')).toBeNull();
  });

  it('asks to pick a part when there is no goal', async () => {
    await setup({
      ...path,
      topicId: '',
      topicName: '',
      milestones: [part('A1', [roadmap('family', 0, 4)])],
    });

    expect(text(el.querySelector('[role="status"]'))).toContain('Bạn chưa chọn lộ trình nào');
    expect(text(el.querySelector('.part a'))).toBe('Nhảy tới phần 1');
    expect(el.querySelector('a.back')?.getAttribute('href')).toBe('/');
  });
});
