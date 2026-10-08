import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { computed, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { Account, User } from '../../../core/models/user';
import { AuthService } from '../../../core/services/auth.service';
import { Accounts } from './accounts';

describe('Accounts', () => {
  let fixture: ComponentFixture<Accounts>;
  let el: HTMLElement;
  let http: HttpTestingController;

  const me: Account = { id: 'a1', email: 'admin@x.vn', role: 'admin', createdAt: '2026-09-28T08:00:00Z' };
  const guest: Account = { id: 'g1', email: 'khach@x.vn', role: 'guest', createdAt: '2026-10-01T08:00:00Z' };
  const member: Account = { id: 'm1', email: 'hoc@x.vn', role: 'member', createdAt: '2026-10-02T08:00:00Z' };

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) => node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string, root: ParentNode = el) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find(
      (b) => text(b).startsWith(label) || b.getAttribute('aria-label') === label,
    );
  const emails = () => Array.from(el.querySelectorAll('.account .email')).map((e) => text(e).replace(' Bạn', ''));
  const dialog = () => el.querySelector('lu-account-dialog dialog') as HTMLDialogElement;
  const type = (selector: string, value: string) => {
    const input = el.querySelector<HTMLInputElement>(selector)!;
    input.value = value;
    input.dispatchEvent(new Event('input'));
  };
  /** The role radios are in ROLE_INFO order (the forms directive keeps the value, not the DOM). */
  const radio = (role: string) =>
    dialog().querySelectorAll<HTMLInputElement>('input[type="radio"]')[['admin', 'member', 'guest'].indexOf(role)];
  const pickRole = (role: string) => radio(role).click();

  beforeEach(async () => {
    // jsdom has no <dialog> methods.
    HTMLDialogElement.prototype.showModal ??= function (this: HTMLDialogElement) {
      this.setAttribute('open', '');
    };
    HTMLDialogElement.prototype.close ??= function (this: HTMLDialogElement) {
      this.removeAttribute('open');
    };
    const user = signal<User | null>({ id: 'a1', email: 'admin@x.vn', role: 'admin', timezone: 'Asia/Ho_Chi_Minh' });
    await TestBed.configureTestingModule({
      imports: [Accounts],
      providers: [
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: { currentUser: user.asReadonly(), isAdmin: computed(() => true) } },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(Accounts);
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
    http.expectOne('/api/admin/users').flush({ users: [me, guest, member] });
    await settle();
  });

  afterEach(() => http.verify());

  it('lists accounts with a role badge and counts each role', () => {
    expect(emails()).toEqual(['admin@x.vn', 'khach@x.vn', 'hoc@x.vn']);
    const badges = Array.from(el.querySelectorAll('.account .badge')).map((b) => text(b));
    expect(badges).toEqual(['Quản trị', 'Khách', 'Thành viên']);
    const stats = Array.from(el.querySelectorAll('.stat')).map((s) => text(s));
    expect(stats).toEqual(['1 Quản trị', '1 Thành viên', '1 Khách']);
    expect(el.querySelector('select')).toBeNull();
  });

  it('filters by role and searches by email', async () => {
    button('Khách 1')!.click();
    await settle();
    expect(emails()).toEqual(['khach@x.vn']);
    expect(button('Khách 1')!.getAttribute('aria-pressed')).toBe('true');

    button('Tất cả')!.click();
    type('#account-search', 'HOC');
    await settle();
    expect(emails()).toEqual(['hoc@x.vn']);

    type('#account-search', 'zzz');
    await settle();
    expect(text(el.querySelector('.empty'))).toBe('Không có tài khoản nào khớp.');
  });

  it('marks your own account and does not let you delete it', () => {
    const mine = el.querySelector('.account')!;
    expect(text(mine)).toContain('Bạn');
    expect(button('Xoá admin@x.vn')).toBeUndefined();
    expect(button('Xoá khach@x.vn')).toBeDefined();
  });

  it('adds an account', async () => {
    button('Thêm tài khoản')!.click();
    await settle();
    expect(dialog().hasAttribute('open')).toBe(true);
    expect(text(dialog().querySelector('h2'))).toBe('Thêm tài khoản');

    button('Thêm tài khoản', dialog())!.click();
    await settle();
    expect(text(dialog())).toContain('Vui lòng nhập email');
    expect(text(dialog())).toContain('Vui lòng nhập mật khẩu');

    type('#account-email', 'moi@x.vn');
    type('#account-password', 'matkhau123');
    pickRole('member');
    button('Thêm tài khoản', dialog())!.click();
    await settle();
    const req = http.expectOne('/api/admin/users');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ email: 'moi@x.vn', password: 'matkhau123', role: 'member' });
    req.flush({ user: { id: 'n1', email: 'moi@x.vn', role: 'member', createdAt: '2026-10-08T08:00:00Z' } }, { status: 201, statusText: 'Created' });
    await settle();
    expect(dialog().hasAttribute('open')).toBe(false);
    expect(emails()).toContain('moi@x.vn');
    expect(text(el.querySelector('.notice'))).toBe('Đã thêm tài khoản moi@x.vn.');
  });

  it('shows the server field errors in the dialog', async () => {
    button('Thêm tài khoản')!.click();
    await settle();
    type('#account-email', 'x@x.vn');
    type('#account-password', 'short');
    button('Thêm tài khoản', dialog())!.click();
    await settle();
    http
      .expectOne('/api/admin/users')
      .flush({ error: 'validation', fields: { password: 'Mật khẩu cần ít nhất 8 ký tự' } }, { status: 400, statusText: 'Bad' });
    await settle();
    expect(dialog().hasAttribute('open')).toBe(true);
    expect(text(dialog().querySelector('#account-password-error'))).toContain('Mật khẩu cần ít nhất 8 ký tự');
    expect(dialog().querySelector('#account-password')!.getAttribute('aria-invalid')).toBe('true');
  });

  it('edits an account, keeping the password when left empty', async () => {
    button('Sửa khach@x.vn')!.click();
    await settle();
    expect(text(dialog().querySelector('h2'))).toBe('Sửa tài khoản');
    expect(dialog().querySelector<HTMLInputElement>('#account-email')!.value).toBe('khach@x.vn');
    expect(radio('guest').checked).toBe(true);

    pickRole('member');
    button('Lưu thay đổi', dialog())!.click();
    await settle();
    const req = http.expectOne('/api/admin/users/g1');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({ email: 'khach@x.vn', password: '', role: 'member' });
    req.flush({ user: { ...guest, role: 'member' } });
    await settle();
    expect(text(el.querySelectorAll('.account .badge')[1])).toBe('Thành viên');
    expect(text(el.querySelector('.notice'))).toBe('Đã lưu tài khoản khach@x.vn.');
  });

  it('does not let you change your own role', async () => {
    button('Sửa admin@x.vn')!.click();
    await settle();
    const radios = Array.from(dialog().querySelectorAll<HTMLInputElement>('input[type="radio"]'));
    expect(radios.every((r) => r.disabled)).toBe(true);
    expect(text(dialog())).toContain('Bạn không thể tự đổi quyền của chính mình.');
  });

  it('deletes an account after confirming', async () => {
    button('Xoá khach@x.vn')!.click();
    await settle();
    const confirm = el.querySelector('lu-confirm-dialog [role="alertdialog"]')!;
    expect(text(confirm)).toContain('khach@x.vn');
    expect(text(confirm)).toContain('Không thể hoàn tác');
    button('Xoá tài khoản', confirm)!.click();
    await settle();
    const req = http.expectOne('/api/admin/users/g1');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await settle();
    expect(emails()).toEqual(['admin@x.vn', 'hoc@x.vn']);
    expect(text(el.querySelector('.notice'))).toBe('Đã xoá tài khoản khach@x.vn.');
  });
});
