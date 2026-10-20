import { Routes } from '@angular/router';

export const vocabularyRoutes: Routes = [
  {
    path: '',
    title: 'Sổ từ · Luna',
    loadComponent: () => import('./notebook/notebook').then((m) => m.Notebook),
  },
  {
    path: 'review',
    title: 'Ôn tập · Luna',
    loadComponent: () => import('./review/review').then((m) => m.Review),
  },
  {
    path: 'review/lessons',
    title: 'Bài đã học · Luna',
    loadComponent: () => import('./all-lessons/all-lessons').then((m) => m.AllLessons),
  },
  {
    path: 'review/lessons/:id',
    title: 'Ôn bài · Luna',
    loadComponent: () => import('./lesson-review/lesson-review').then((m) => m.LessonReview),
  },
  // Talk with the AI (added 2026-10-20).
  {
    path: 'talk',
    title: 'Luyện nói với AI · Luna',
    loadComponent: () => import('./talk/talk-topics/talk-topics').then((m) => m.TalkTopics),
  },
  {
    path: 'talk/:id',
    title: 'Luyện nói với AI · Luna',
    loadComponent: () => import('./talk/talk-session/talk-session').then((m) => m.TalkSessionPage),
  },
];
