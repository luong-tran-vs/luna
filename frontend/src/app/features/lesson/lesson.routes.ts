import { Routes } from '@angular/router';

/** Learner lesson pages. The parent `lessons` route in app.routes.ts requires login. */
export const lessonRoutes: Routes = [
  {
    path: '',
    title: 'Bài học · Luna',
    loadComponent: () => import('./learn-parts/learn-parts').then((m) => m.LearnParts),
  },
  {
    path: 'levels/:level',
    title: 'Trình độ · Luna',
    loadComponent: () => import('./learn-path/learn-path').then((m) => m.LearnPath),
  },
  {
    path: ':id',
    title: 'Bài học · Luna',
    loadComponent: () => import('./lesson-detail/lesson-detail').then((m) => m.LessonDetail),
  },
  {
    path: ':id/read',
    title: 'Đọc · Luna',
    loadComponent: () => import('./reading/reading').then((m) => m.Reading),
  },
  {
    path: ':id/listen',
    title: 'Nghe · Luna',
    loadComponent: () => import('./listening/listening').then((m) => m.Listening),
  },
  {
    path: ':id/write',
    title: 'Viết · Luna',
    loadComponent: () => import('./writing/writing').then((m) => m.Writing),
  },
];
