import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { GoalView } from '../../core/models/study';
import { NaturalVoiceService } from '../../core/natural-voice/natural-voice.service';
import { AuthService } from '../../core/services/auth.service';
import { InstallService } from '../../core/services/install.service';
import { PALETTES, PaletteService } from '../../core/services/palette.service';
import { ResetPart, SettingsApiService } from '../../core/services/settings-api.service';
import { StudyApiService } from '../../core/services/study-api.service';
import { ThemeService } from '../../core/services/theme.service';
import { WritingNotifier } from '../../core/services/writing-notifier.service';
import { ConfirmDialog } from '../../shared/components/confirm-dialog/confirm-dialog';
import { Icon, IconName } from '../../shared/components/icon/icon';

interface AccountLink {
  path: string;
  label: string;
  hint: string;
  icon: IconName;
}

/** The parts the learner may start over, with what resetting each removes (added 2026-11-23). */
const RESET_PARTS: readonly { value: ResetPart; label: string; hint: string }[] = [
  {
    value: 'lessons',
    label: 'Bài học theo chủ đề',
    hint: 'Bước đã học, bài đã xong, câu trả lời phần Đọc, nghe chép, kết quả ôn theo bài',
  },
  { value: 'grammar', label: 'Ngữ pháp', hint: 'Bài đã học, đã thuộc và điểm luyện tập' },
  {
    value: 'vocab',
    label: 'Ôn từ vựng',
    hint: 'Thẻ ôn trở về từ mới, xoá lịch sử ôn; các từ đã lưu vẫn giữ',
  },
  { value: 'toeic', label: 'TOEIC', hint: 'Bài làm, kết quả và sổ lỗi sai; điểm mục tiêu vẫn giữ' },
];

const LINKS: readonly AccountLink[] = [
  {
    path: '/stats',
    label: 'Thống kê & huy hiệu',
    hint: 'Theo tuần, tháng và huy hiệu',
    icon: 'chart',
  },
  { path: '/writings', label: 'Bài viết', hint: 'Kết quả chấm bài viết', icon: 'pencil' },
  {
    path: '/settings',
    label: 'Cài đặt học tập',
    hint: 'Số thẻ ôn, múi giờ, xuất dữ liệu',
    icon: 'sliders',
  },
];

/**
 * The account tab (client sketch, screen 11): who is logged in, links to the other pages, dark mode,
 * a link to the color page, installing the app (PWA), logout.
 */
@Component({
  selector: 'lu-account',
  imports: [ConfirmDialog, Icon, RouterLink],
  templateUrl: './account.html',
  styleUrl: './account.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Account {
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);
  protected readonly notifier = inject(WritingNotifier);
  private readonly palette = inject(PaletteService);
  protected readonly voice = inject(NaturalVoiceService);
  protected readonly installer = inject(InstallService);
  private readonly router = inject(Router);

  protected readonly links = LINKS;
  protected readonly goal = signal<GoalView | null>(null);
  /** iPhone/iPad: whether the "Share → Add to Home Screen" steps are shown. */
  protected readonly iosHelp = signal(false);
  protected readonly dark = computed(() => this.theme.resolved() === 'dark');
  protected readonly paletteName = computed(
    () => PALETTES.find((p) => p.id === this.palette.palette())?.name ?? '',
  );
  protected readonly voiceHint = computed(() => {
    switch (this.voice.state()) {
      case 'loading':
        return this.voice.starting()
          ? 'Đang khởi động giọng đọc…'
          : `Đang tải giọng đọc… ${this.voice.percent()}%`;
      case 'ready':
        return this.voice.only()
          ? 'Đang dùng'
          : 'Đang dùng. Câu chưa chuẩn bị kịp sẽ tạm đọc bằng giọng của trình duyệt';
      case 'error':
        return this.voice.only()
          ? 'Chưa tải được giọng đọc. Tắt rồi bật lại để thử lại.'
          : 'Chưa tải được, đang dùng giọng của trình duyệt. Tắt rồi bật lại để thử lại.';
      default:
        return 'Tải khoảng 60MB một lần, nên dùng Wi-Fi';
    }
  });
  protected readonly machineHint = computed(() =>
    this.voice.only()
      ? 'Đã tắt: chỉ dùng giọng tự nhiên, câu chưa sẵn sàng sẽ chờ tải xong rồi phát'
      : 'Câu giọng tự nhiên chưa chuẩn bị kịp sẽ tạm đọc bằng giọng máy',
  );
  protected readonly initial = computed(() => this.auth.currentUser()?.email.charAt(0) ?? '?');

  private readonly settingsApi = inject(SettingsApiService);
  protected readonly resetParts = RESET_PARTS;
  /** "Đặt lại tiến độ học": the dialog is open, the ticked parts, sending, and the outcome. */
  protected readonly resetting = signal(false);
  protected readonly resetPicked = signal<ReadonlySet<ResetPart>>(new Set());
  protected readonly resetBusy = signal(false);
  protected readonly resetNote = signal('');
  protected readonly resetError = signal('');
  protected readonly resetAll = computed(() => this.resetPicked().size === RESET_PARTS.length);
  /** One text whatever is ticked: a text that changed with the ticks resized the centred dialog. */
  protected readonly resetMessage =
    'Tích những phần muốn học lại từ đầu. Tiến độ của phần đã chọn sẽ bị xoá và không lấy lại được; chuỗi ngày học, cài đặt và các từ đã lưu luôn được giữ.';

  constructor() {
    inject(StudyApiService)
      .goals()
      // The level line is extra information: without it the page still works.
      .subscribe({ next: (g) => this.goal.set(g.active), error: () => undefined });
  }

  protected openReset(): void {
    this.resetPicked.set(new Set());
    this.resetNote.set('');
    this.resetError.set('');
    this.resetting.set(true);
  }

  protected pickReset(part: ResetPart, on: boolean): void {
    this.resetPicked.update((s) => {
      const next = new Set(s);
      if (on) {
        next.add(part);
      } else {
        next.delete(part);
      }
      return next;
    });
  }

  protected pickAllReset(on: boolean): void {
    this.resetPicked.set(new Set(on ? RESET_PARTS.map((p) => p.value) : []));
  }

  protected confirmReset(): void {
    const parts = RESET_PARTS.filter((p) => this.resetPicked().has(p.value));
    if (parts.length === 0 || this.resetBusy()) {
      return;
    }
    this.resetting.set(false);
    this.resetBusy.set(true);
    this.settingsApi.resetProgress(parts.map((p) => p.value)).subscribe({
      next: () => {
        this.resetBusy.set(false);
        this.resetNote.set(
          parts.length === RESET_PARTS.length
            ? 'Đã đặt lại toàn bộ tiến độ học.'
            : `Đã đặt lại: ${parts.map((p) => p.label).join(', ')}.`,
        );
      },
      error: () => {
        this.resetBusy.set(false);
        this.resetError.set('Không đặt lại được, vui lòng thử lại.');
      },
    });
  }

  protected toggleDark(): void {
    this.theme.set(this.dark() ? 'light' : 'dark');
  }

  protected toggleNaturalVoice(): void {
    if (this.voice.enabled()) {
      this.voice.disable();
    } else {
      this.voice.enable();
    }
  }

  protected async install(): Promise<void> {
    await this.installer.install();
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }
}
