import { Routes } from '@angular/router';

import { unsavedChangesGuard } from '../../core/guards/unsaved-changes.guard';

/** Admin area (F2, F14, F7). Guards are applied on the parent `admin` route in app.routes.ts. */
export const adminRoutes: Routes = [
  {
    path: '',
    title: 'Tổng quan · Quản trị · Luna',
    loadComponent: () => import('./dashboard/dashboard').then((m) => m.Dashboard),
  },
  {
    path: 'lessons',
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
    path: 'words',
    title: 'Kho từ vựng · Quản trị · Luna',
    loadComponent: () => import('./word-bank/word-bank').then((m) => m.WordBank),
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
  {
    path: 'accounts',
    title: 'Tài khoản · Quản trị · Luna',
    loadComponent: () => import('./accounts/accounts').then((m) => m.Accounts),
  },
  {
    path: 'ai-usage',
    title: 'Sử dụng AI · Quản trị · Luna',
    loadComponent: () => import('./ai-usage/ai-usage').then((m) => m.AiUsage),
  },
  {
    path: 'appearance',
    title: 'Giao diện · Quản trị · Luna',
    loadComponent: () => import('./appearance/appearance').then((m) => m.Appearance),
  },
  {
    path: 'tts-lab',
    title: 'Thử giọng đọc · Quản trị · Luna',
    loadComponent: () => import('./tts-lab/tts-lab').then((m) => m.TtsLab),
  },
  {
    path: 'stt-lab',
    title: 'Thử nhận dạng giọng nói · Quản trị · Luna',
    loadComponent: () => import('./stt-lab/stt-lab').then((m) => m.SttLab),
  },
];
