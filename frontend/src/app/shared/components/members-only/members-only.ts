import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { Icon } from '../icon/icon';

/** What a guest sees in place of a members-only section (route data.title names the section). */
@Component({
  selector: 'lu-members-only',
  imports: [RouterLink, Icon],
  template: `
    <section class="screen">
      <header class="screen-bar">
        <h1 class="screen-title">{{ title }}</h1>
      </header>
      <div class="soft-card members-only" role="note">
        <span class="icon-chip tone-accent"><lu-icon name="lock" /></span>
        <p class="card-title">Chỉ dành cho thành viên</p>
        <p class="muted">
          Tài khoản khách chỉ học được bài đầu tiên của mỗi chủ đề. Liên hệ quản trị viên để được nâng lên
          thành viên và mở khoá phần này.
        </p>
        <a class="btn-primary" routerLink="/">Về trang chủ</a>
      </div>
    </section>
  `,
  styles: `
    .members-only {
      display: grid;
      gap: var(--space-3);
      justify-items: start;
    }
    .card-title,
    .muted {
      margin: 0;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MembersOnly {
  /** The section's name, from the route's data.title. */
  protected readonly title: string = inject(ActivatedRoute).snapshot.data['title'] ?? 'Luna';
}
