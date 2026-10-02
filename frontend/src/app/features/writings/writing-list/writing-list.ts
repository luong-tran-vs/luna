import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

import { formatScore, WritingSummary } from '../../../core/models/writing';
import { WritingApiService } from '../../../core/services/writing-api.service';
import { Icon } from '../../../shared/components/icon/icon';
import { Loading } from '../../../shared/components/loading/loading';

/** The learner's submitted writings (F8), newest first. */
@Component({
  selector: 'lu-writing-list',
  imports: [Loading, DatePipe, Icon, RouterLink],
  templateUrl: './writing-list.html',
  styleUrl: './writing-list.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WritingList {
  private readonly api = inject(WritingApiService);

  protected readonly formatScore = formatScore;
  protected readonly writings = signal<WritingSummary[] | null>(null);
  protected readonly loadError = signal(false);

  constructor() {
    void this.load();
  }

  protected async load(): Promise<void> {
    this.loadError.set(false);
    try {
      this.writings.set(await firstValueFrom(this.api.list()));
    } catch {
      this.loadError.set(true);
    }
  }
}
