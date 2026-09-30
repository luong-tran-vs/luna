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
];
