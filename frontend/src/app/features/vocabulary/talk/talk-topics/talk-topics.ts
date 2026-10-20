import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';

import { TalkSummary, TalkTopic } from '../../../../core/models/talk';
import { TalkApiService } from '../../../../core/services/talk-api.service';
import { Icon } from '../../../../shared/components/icon/icon';
import { Loading } from '../../../../shared/components/loading/loading';
import { talkError } from '../talk-error';

/** How many past sessions the page lists. */
const HISTORY = 10;

/**
 * Luyện nói với AI (added 2026-10-20): the topics the learner studied, each with a button that
 * starts a conversation about it, and the past conversations to read again.
 */
@Component({
  selector: 'lu-talk-topics',
  imports: [DatePipe, Icon, Loading, RouterLink],
  templateUrl: './talk-topics.html',
  styleUrl: './talk-topics.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TalkTopics {
  private readonly api = inject(TalkApiService);
  private readonly router = inject(Router);

  /** Null while loading. */
  protected readonly topics = signal<TalkTopic[] | null>(null);
  protected readonly history = signal<TalkSummary[]>([]);
  protected readonly loadError = signal(false);
  /** The topic whose conversation is being started. */
  protected readonly starting = signal<string | null>(null);
  protected readonly startError = signal<{ topicId: string; message: string } | null>(null);

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.topics.set(null);
    forkJoin({ topics: this.api.topics(), sessions: this.api.sessions() }).subscribe({
      next: ({ topics, sessions }) => {
        this.topics.set(topics);
        this.history.set(sessions.slice(0, HISTORY));
      },
      error: () => this.loadError.set(true),
    });
  }

  protected start(topic: TalkTopic): void {
    if (this.starting()) {
      return;
    }
    this.starting.set(topic.id);
    this.startError.set(null);
    this.api.start(topic.id).subscribe({
      next: (s) => void this.router.navigate(['/vocabulary/talk', s.id]),
      error: (err: unknown) => {
        this.starting.set(null);
        this.startError.set({
          topicId: topic.id,
          message: talkError(err, 'Chưa bắt đầu được, vui lòng thử lại.'),
        });
      },
    });
  }
}
