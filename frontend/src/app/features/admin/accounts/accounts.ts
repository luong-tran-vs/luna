import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { Account, Role } from '../../../core/models/user';
import { AuthService } from '../../../core/services/auth.service';
import { ConfirmDialog } from '../../../shared/components/confirm-dialog/confirm-dialog';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';
import { AdminApiService } from '../admin-api.service';
import { AccountDialog } from './account-dialog/account-dialog';
import { ROLE_INFO } from './roles';

/**
 * Account management: add, edit (email, password, role) and delete accounts. Roles are admin,
 * member (every lesson) and guest (only the first lesson of each roadmap).
 */
@Component({
  selector: 'lu-accounts',
  imports: [DatePipe, Loading, Icon, ConfirmDialog, AccountDialog],
  templateUrl: './accounts.html',
  styleUrl: './accounts.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Accounts {
  private readonly api = inject(AdminApiService);
  private readonly auth = inject(AuthService);

  protected readonly roles = ROLE_INFO;
  protected readonly roleInfo = Object.fromEntries(ROLE_INFO.map((r) => [r.role, r])) as Record<
    Role,
    (typeof ROLE_INFO)[number]
  >;
  protected readonly myId = computed(() => this.auth.currentUser()?.id ?? '');

  protected readonly accounts = signal<Account[] | null>(null);
  protected readonly failed = signal(false);
  protected readonly query = signal('');
  /** The role shown, null for every role. */
  protected readonly filter = signal<Role | null>(null);
  protected readonly notice = signal<string | null>(null);
  protected readonly error = signal<string | null>(null);

  /** The dialog: open, and the account being edited (null to add one). */
  protected readonly dialogOpen = signal(false);
  protected readonly editing = signal<Account | null>(null);

  protected readonly deleting = signal<Account | null>(null);
  protected readonly deleteBusy = signal(false);

  protected readonly counts = computed(() => {
    const out: Record<Role, number> = { admin: 0, member: 0, guest: 0 };
    for (const a of this.accounts() ?? []) {
      out[a.role]++;
    }
    return out;
  });

  protected readonly shown = computed(() => {
    const q = this.query().trim().toLowerCase();
    const role = this.filter();
    return (this.accounts() ?? []).filter((a) => (!role || a.role === role) && (!q || a.email.includes(q)));
  });

  protected readonly deleteMessage = computed(() => {
    const a = this.deleting();
    return a
      ? `Tài khoản ${a.email} cùng toàn bộ tiến độ học, thẻ từ vựng và bài viết của tài khoản này sẽ bị xoá. Không thể hoàn tác.`
      : '';
  });

  constructor() {
    void this.load();
  }

  protected async load(): Promise<void> {
    this.failed.set(false);
    try {
      this.accounts.set(await firstValueFrom(this.api.accounts()));
    } catch {
      this.failed.set(true);
    }
  }

  protected initial(a: Account): string {
    return a.email.charAt(0).toUpperCase();
  }

  protected onSearch(event: Event): void {
    this.query.set((event.target as HTMLInputElement).value);
  }

  protected openAdd(): void {
    this.editing.set(null);
    this.dialogOpen.set(true);
  }

  protected openEdit(a: Account): void {
    this.editing.set(a);
    this.dialogOpen.set(true);
  }

  protected onSaved(saved: Account): void {
    const isNew = !this.editing();
    this.accounts.update((list) =>
      isNew ? [...(list ?? []), saved] : (list ?? []).map((a) => (a.id === saved.id ? saved : a)),
    );
    this.dialogOpen.set(false);
    this.error.set(null);
    this.notice.set(isNew ? `Đã thêm tài khoản ${saved.email}.` : `Đã lưu tài khoản ${saved.email}.`);
  }

  protected async confirmDelete(): Promise<void> {
    const a = this.deleting();
    if (!a || this.deleteBusy()) {
      return;
    }
    this.deleteBusy.set(true);
    this.notice.set(null);
    this.error.set(null);
    try {
      await firstValueFrom(this.api.deleteAccount(a.id));
      this.accounts.update((list) => (list ?? []).filter((x) => x.id !== a.id));
      this.notice.set(`Đã xoá tài khoản ${a.email}.`);
    } catch (err) {
      const body = err instanceof ApiError ? (err.body as { message?: string } | null) : null;
      this.error.set(body?.message ?? 'Không xoá được tài khoản, vui lòng thử lại.');
    } finally {
      this.deleteBusy.set(false);
      this.deleting.set(null);
    }
  }
}
