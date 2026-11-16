/**
 * The learner's games (F26, specs/021-learning-games/contracts/api.md): GET /api/games, the rounds
 * of each game, POST /api/games/{game}/results and the daily challenge.
 */

export type GameId = 'picture' | 'verbs' | 'letters' | 'memory' | 'grammar';

/** A game of the list with the learner's best. */
export interface GameSummary {
  id: GameId;
  best: number;
  /** When the best was set (RFC 3339); null when there is none. */
  bestAt: string | null;
  /** "Bắn lỗi" for a guest. */
  locked: boolean;
}

/** Today's challenge as the list and the home page show it. */
export interface DailyStatus {
  day: string;
  done: boolean;
  streak: number;
}

export interface GamesList {
  games: GameSummary[];
  daily: DailyStatus;
}

/** "Ảnh nhanh tay": a picture and four words; answer is the index of the right one. */
export interface PictureQuestion {
  lemma: string;
  imageUrl: string;
  meaningVi: string;
  options: string[];
  answer: number;
}

export interface PictureRound {
  questions: PictureQuestion[];
}

/** A word to spell ("Chữ rơi", "Ghép chữ"), or a pair of "Lật thẻ"; imageUrl may be "". */
export interface GameWord {
  lemma: string;
  meaningVi: string;
  imageUrl: string;
}

export interface LettersRound {
  words: GameWord[];
}

export interface MemoryRound {
  pairs: GameWord[];
}

/** Token range [from, to). */
export interface TokenSpan {
  from: number;
  to: number;
}

/** "Bắn lỗi": a sentence as tokens; wrong is null when the sentence is right (options then empty). */
export interface SpotQuestion {
  tokens: string[];
  wrong: TokenSpan | null;
  options: string[];
  answer: number;
  ruleVi: string;
  topicVi: string;
  pointId: string;
  source: 'choice' | 'fix' | 'mistake' | 'spot';
}

export interface GrammarRound {
  questions: SpotQuestion[];
}

/** A verb form typed wrong. */
export interface VerbMiss {
  base: string;
  form: 'past' | 'participle';
  typed: string;
}

export interface GameResultBody {
  score: number;
  /** False when the learner left halfway: the best stays, the misses still count. */
  completed: boolean;
  wrongWords: string[];
  wrongVerbs: VerbMiss[];
}

export interface GameResultReply {
  best: number;
  newBest: boolean;
  rescheduled: number;
}

export type DailyKind = 'picture' | 'verb' | 'letters' | 'grammar';

/** One question of the daily challenge: exactly one of the parts is set, by kind. */
export interface DailyQuestion {
  kind: DailyKind;
  picture?: PictureQuestion;
  /** base "" lets the page pick a common verb with the day's seed. */
  verb?: { base: string };
  letters?: GameWord;
  grammar?: SpotQuestion;
}

export interface DailyAnswer {
  kind: DailyKind;
  right: boolean;
}

export interface DailyStreak {
  current: number;
  longest: number;
}

export interface DailyCompare {
  yesterday: number | null;
  avg7: number | null;
  best: number;
}

export interface DailyResultView {
  score: number;
  answers: DailyAnswer[];
  finishedAt: string;
}

export interface DailyView {
  day: string;
  seed: number;
  done: boolean;
  streak: DailyStreak;
  /** Monday to Sunday of this week: whether the challenge was played. */
  week: boolean[];
  result: DailyResultView | null;
  compare: DailyCompare;
  questions: DailyQuestion[];
}

export interface DailyResultBody {
  day: string;
  score: number;
  answers: DailyAnswer[];
  wrongWords: string[];
  wrongVerbs: VerbMiss[];
}

export interface DailyResultReply {
  streak: DailyStreak;
  compare: DailyCompare;
  rescheduled: number;
}

/** Admin: how many "Bắn lỗi" sentences a published grammar point gives, by source. */
export interface SpotCount {
  pointId: string;
  level: string;
  titleVi: string;
  choice: number;
  fix: number;
  spot: number;
  draft: number;
  total: number;
}

export interface SpotCounts {
  points: SpotCount[];
}

/** Admin: an extra "Bắn lỗi" sentence. */
export interface SpotItem {
  id: string;
  pointId: string;
  level: string;
  sentence: string;
  wrong: string;
  right: string;
  distractors: string[];
  ruleVi: string;
  status: 'draft' | 'approved';
  createdAt: string;
  updatedAt: string;
}

export interface SpotItemEdit {
  sentence: string;
  wrong: string;
  right: string;
  distractors: string[];
  ruleVi: string;
}

export interface SpotGenerateReply {
  created: SpotItem[];
  dropped: number;
}

/** Admin: a "Bắn lỗi" sentence written outside the AI, in the shape the AI returns. */
export interface SpotImportItem {
  sentence: string;
  wrong: string;
  right: string;
  distractors: string[];
  ruleVi: string;
}

/** Admin: what became of one imported sentence. */
export interface SpotImportReport {
  index: number;
  sentence: string;
  kept: boolean;
  duplicate?: boolean;
  fields?: Record<string, string>;
}

export interface SpotImportReply {
  dryRun: boolean;
  created: SpotItem[];
  kept: number;
  dropped: number;
  reports: SpotImportReport[];
}
