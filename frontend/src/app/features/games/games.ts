import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { finalize, switchMap } from 'rxjs';

import { GameId, GamesList } from '../../core/models/games';
import { GamesApiService } from '../../core/services/games-api.service';
import { Icon, IconName } from '../../shared/components/icon/icon';
import { Loading } from '../../shared/components/loading/loading';
import { formatScore } from './shared/game-score.logic';

/** How each game is shown in the list (FR-001). */
export const GAME_INFO: Readonly<
  Record<GameId, { name: string; line: string; icon: IconName; bestUnit: string }>
> = {
  picture: { name: 'Ảnh nhanh tay', line: 'Nhìn ảnh, chọn đúng từ trong 60 giây', icon: 'eye', bestUnit: 'điểm' },
  verbs: { name: 'Đấu động từ', line: 'Gõ nhanh V2, V3 với 3 mạng', icon: 'keyboard', bestUnit: 'động từ liền' },
  letters: { name: 'Chữ rơi', line: 'Chạm chữ cái theo thứ tự để ghép từ', icon: 'layers', bestUnit: 'điểm' },
  memory: { name: 'Lật thẻ', line: 'Ghép ảnh hoặc nghĩa với từ tiếng Anh', icon: 'cards', bestUnit: 'điểm' },
  grammar: { name: 'Bắn lỗi ngữ pháp', line: 'Tìm chữ sai trong câu và sửa', icon: 'target', bestUnit: 'điểm' },
};

/**
 * The games page (/games, FR-001): today's challenge first, then each released game with its best.
 * Results kept on this device while offline are sent first.
 */
@Component({
  selector: 'lu-games',
  imports: [RouterLink, Icon, Loading],
  templateUrl: './games.html',
  styleUrl: './games.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Games {
  private readonly api = inject(GamesApiService);

  protected readonly data = signal<GamesList | null>(null);
  protected readonly failed = signal(false);
  protected readonly loading = signal(true);
  protected readonly info = GAME_INFO;
  protected readonly format = formatScore;

  protected readonly games = computed(() =>
    (this.data()?.games ?? []).filter((g) => g.id in GAME_INFO),
  );

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loading.set(true);
    this.failed.set(false);
    this.api
      .flushPending()
      .pipe(
        switchMap(() => this.api.list()),
        finalize(() => this.loading.set(false)),
      )
      .subscribe({
        next: (d) => this.data.set(d),
        error: () => this.failed.set(true),
      });
  }
}
