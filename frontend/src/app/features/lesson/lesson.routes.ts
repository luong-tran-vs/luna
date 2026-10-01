import { Routes } from '@angular/router';

/** Learner lesson pages. The parent `lessons` route in app.routes.ts requires login. */
export const lessonRoutes: Routes = [
  {
    path: '',
    title: 'Bài học · Luna',
    loadComponent: () => import('./my-lessons/my-lessons').then((m) => m.MyLessons),
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
