import { Routes } from '@angular/router';

import { adminGuard, authGuard, guestGuard, learnerGuard } from './core/guards/auth.guard';

/**
 * Three areas, each with its own layout:
 * - admin (/admin): content management only, for admins;
 * - learner: learning only; admins are sent to /admin;
 * - public: login, register and "no permission".
 */
export const routes: Routes = [
  {
    path: 'admin',
    canActivate: [authGuard, adminGuard],
    loadComponent: () => import('./layouts/admin-layout/admin-layout').then((m) => m.AdminLayout),
    loadChildren: () => import('./features/admin/admin.routes').then((m) => m.adminRoutes),
  },
  {
    path: '',
    canActivate: [authGuard, learnerGuard],
    loadComponent: () => import('./layouts/learner-layout/learner-layout').then((m) => m.LearnerLayout),
    children: [
      {
        path: '',
        loadComponent: () => import('./features/home/home').then((m) => m.Home),
      },
      {
        path: 'lessons',
        loadChildren: () => import('./features/lesson/lesson.routes').then((m) => m.lessonRoutes),
      },
      {
        path: 'today',
        title: 'Hôm nay · Luna',
        loadComponent: () => import('./features/lesson/today/today').then((m) => m.Today),
      },
      {
        path: 'goal',
        title: 'Mục tiêu · Luna',
        loadComponent: () => import('./features/lesson/goal/goal').then((m) => m.Goal),
      },
      {
        path: 'account',
        title: 'Tài khoản · Luna',
        loadComponent: () => import('./features/account/account').then((m) => m.Account),
      },
      {
        path: 'settings',
        title: 'Cài đặt · Luna',
        loadComponent: () => import('./features/settings/settings').then((m) => m.Settings),
      },
      {
        path: 'writings',
        title: 'Bài viết · Luna',
        loadComponent: () => import('./features/writings/writing-list/writing-list').then((m) => m.WritingList),
      },
      {
        path: 'writings/:id',
        title: 'Bài viết · Luna',
        loadComponent: () => import('./features/writings/writing-detail/writing-detail').then((m) => m.WritingDetail),
      },
      {
        path: 'stats',
        title: 'Thống kê · Luna',
        loadComponent: () => import('./features/stats/stats').then((m) => m.Stats),
      },
      {
        path: 'vocabulary',
        loadChildren: () => import('./features/vocabulary/vocabulary.routes').then((m) => m.vocabularyRoutes),
      },
    ],
  },
  // Last: as an empty-path parent it would also match "/" with no child.
  {
    path: '',
    loadComponent: () => import('./layouts/public-layout/public-layout').then((m) => m.PublicLayout),
    children: [
      {
        path: 'login',
        canActivate: [guestGuard],
        title: 'Đăng nhập · Luna',
        loadComponent: () => import('./features/auth/login/login').then((m) => m.Login),
      },
      {
        path: 'register',
        canActivate: [guestGuard],
        title: 'Đăng ký · Luna',
        loadComponent: () => import('./features/auth/register/register').then((m) => m.Register),
      },
      {
        path: 'forbidden',
        canActivate: [authGuard],
        title: 'Không có quyền · Luna',
        loadComponent: () => import('./features/forbidden/forbidden').then((m) => m.Forbidden),
      },
    ],
  },
  { path: '**', redirectTo: '' },
];
