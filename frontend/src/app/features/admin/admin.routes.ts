import { Routes } from '@angular/router';

import { unsavedChangesGuard } from '../../core/guards/unsaved-changes.guard';

/** Admin area (F2, F14, F7). Guards are applied on the parent `admin` route in app.routes.ts. */
export const adminRoutes: Routes = [
  {
    path: '',
    title: 'Bài học · Quản trị · Luna',
    loadComponent: () => import('./lesson-list/lesson-list').then((m) => m.LessonList),
  },
  {
    path: 'lessons/new',
    title: 'Thêm bài · Quản trị · Luna',
    loadComponent: () => import('./lesson-form/lesson-form').then((m) => m.LessonForm),
  },
  {
    path: 'lessons/:id',
    title: 'Chi tiết bài · Quản trị · Luna',
    loadComponent: () => import('./lesson-detail/lesson-detail').then((m) => m.LessonDetail),
  },
  {
    path: 'lessons/:id/edit',
    title: 'Sửa bài · Quản trị · Luna',
    loadComponent: () => import('./lesson-form/lesson-form').then((m) => m.LessonForm),
  },
  {
    path: 'topics',
    title: 'Chủ đề · Quản trị · Luna',
    loadComponent: () => import('./topics/topics').then((m) => m.Topics),
  },
  {
    path: 'topics/:id/words',
    title: 'Từ vựng chủ đề · Quản trị · Luna',
    loadComponent: () => import('./topic-words/topic-words').then((m) => m.TopicWords),
  },
  {
    path: 'grammar',
    title: 'Ngữ pháp · Quản trị · Luna',
    loadComponent: () => import('./grammar-list/grammar-list').then((m) => m.GrammarList),
  },
  {
    path: 'grammar/:pointId',
    title: 'Bài ngữ pháp · Quản trị · Luna',
    loadComponent: () => import('./grammar-detail/grammar-detail').then((m) => m.GrammarDetail),
  },
  {
    path: 'roadmap',
    title: 'Lộ trình · Quản trị · Luna',
    canDeactivate: [unsavedChangesGuard],
    loadComponent: () => import('./roadmap/roadmap').then((m) => m.Roadmap),
  },
];
