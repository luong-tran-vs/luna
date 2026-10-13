import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  linkedSignal,
  output,
} from '@angular/core';
import { RouterLink } from '@angular/router';

import { Level, LEVELS } from '../../../../core/models/lesson';
import { sortTopicsAt, Topic, topicLevel } from '../../../../core/models/topic';
import {
  matchesSuggestion,
  SuggestedTopic,
  suggestedLevel,
  TOPIC_CURRICULUM,
  topicKey,
} from '../../../../core/models/topic-curriculum';

/** A suggested topic of the level, and how far the catalogue is with it. */
export interface SuggestionRow {
  suggestion: SuggestedTopic;
  /** The catalogue topic it matches, or null when there is none yet. */
  topic: Topic | null;
  /** Lessons of the matched topic at the level. */
  lessons: number;
  status: 'ready' | 'empty' | 'missing';
}

/** A topic with lessons at the level that the curriculum puts at another level. */
export interface MisplacedTopic {
  topic: Topic;
  level: Level;
}

/**
 * The suggested order of the topics of a level (TOPIC_CURRICULUM), set against the catalogue: the
 * topics that have lessons there stand out, the others say what is missing (lessons, or the topic
 * itself), and the topics with lessons that suit another level are listed. "Áp dụng" orders the
 * level as suggested.
 */
@Component({
  selector: 'lu-topic-suggestions',
  imports: [RouterLink],
  templateUrl: './topic-suggestions.html',
  styleUrl: './topic-suggestions.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TopicSuggestions {
  readonly topics = input.required<Topic[]>();
  /** The level the page is showing; the panel opens on it. */
  readonly level = input<Level>('A1');
  readonly busy = input(false);

  /** Áp dụng: the topic ids of the level in the suggested order. */
  readonly apply = output<{ level: Level; topicIds: string[] }>();
  /** Tạo chủ đề: a suggested topic the catalogue does not have yet. */
  readonly create = output<SuggestedTopic>();

  protected readonly levels = LEVELS;
  protected readonly shownLevel = linkedSignal(() => this.level());

  protected readonly rows = computed<SuggestionRow[]>(() => {
    const level = this.shownLevel();
    const topics = this.topics();
    return TOPIC_CURRICULUM[level].map((suggestion) => {
      // The topic of that very name first: "Cửa hàng" is an alias of Mua sắm but a topic too.
      const topic =
        topics.find((t) => topicKey(t.name) === topicKey(suggestion.name)) ??
        topics.find((t) => matchesSuggestion(t.name, suggestion)) ??
        null;
      const lessons = topic ? topicLevel(topic, level).lessonCount : 0;
      const status = !topic ? 'missing' : lessons > 0 ? 'ready' : 'empty';
      return { suggestion, topic, lessons, status };
    });
  });

  protected readonly readyCount = computed(
    () => this.rows().filter((r) => r.status === 'ready').length,
  );

  /** Topics with lessons at the level that the curriculum suggests for another level. */
  protected readonly misplaced = computed<MisplacedTopic[]>(() => {
    const level = this.shownLevel();
    return this.topics()
      .filter((t) => topicLevel(t, level).lessonCount > 0)
      .flatMap((topic) => {
        const suggested = suggestedLevel(topic.name);
        return suggested && suggested !== level ? [{ topic, level: suggested }] : [];
      });
  });

  /**
   * The level's topics with lessons in the suggested order: the suggested ones first, then the
   * others in their current order.
   */
  protected readonly suggestedOrder = computed(() => {
    const level = this.shownLevel();
    const first = this.rows()
      .filter((r) => r.status === 'ready')
      .map((r) => r.topic!.id);
    const rest = sortTopicsAt(this.topics(), level)
      .filter((t) => topicLevel(t, level).lessonCount > 0 && !first.includes(t.id))
      .map((t) => t.id);
    return [...first, ...rest];
  });

  /** Whether the level is already in the suggested order. */
  protected readonly applied = computed(() => {
    const level = this.shownLevel();
    const current = sortTopicsAt(this.topics(), level)
      .filter((t) => topicLevel(t, level).lessonCount > 0)
      .map((t) => t.id);
    const order = this.suggestedOrder();
    return (
      order.length > 0 &&
      current.join() === order.join() &&
      this.topics().some((t) => topicLevel(t, level).position > 0)
    );
  });

  protected onApply(): void {
    this.apply.emit({ level: this.shownLevel(), topicIds: this.suggestedOrder() });
  }
}
