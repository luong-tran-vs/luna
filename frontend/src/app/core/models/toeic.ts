/**
 * The TOEIC Listening & Reading practice (F27, specs/022-toeic-practice/contracts/api.md).
 */

export type ToeicPart = 1 | 2 | 3 | 4 | 5 | 6 | 7;
export type ToeicStatus = 'draft' | 'published' | 'hidden';
export type TestKind = 'full' | 'mini50' | 'mini100';
export type AttemptMode = 'practice' | 'test';
export type SourceKind = 'test' | 'random' | 'part' | 'tag' | 'mistakes' | 'retry';
export type Accent = 'us' | 'gb' | 'au' | 'ca';
export type Outcome = 'right' | 'wrong' | 'skipped';
export type Section = 'listening' | 'reading';

// --- content ---

export interface KV {
  k: string;
  v: string;
}

export interface DocTable {
  columns: string[];
  rows: string[][];
}

/** A paragraph, heading, list, table or chat message of a text; text may hold **bold** and [[n]]. */
export interface DocBlock {
  kind: 'p' | 'h' | 'list' | 'table' | 'msg';
  text?: string;
  items?: string[];
  table?: DocTable;
  who?: string;
  time?: string;
}

/** A text of Part 6/7 as the exam room shows it. */
export interface ToeicDoc {
  kind: string;
  title?: string;
  meta: KV[];
  body: DocBlock[];
  imageUrl: string | null;
  /** Only in the answer detail. */
  translateVi?: string;
}

export interface GraphicCell {
  label: string;
  row: number;
  col: number;
  span?: number;
}

export interface Graphic {
  kind: 'table' | 'schedule' | 'price-list' | 'floor-plan' | 'bar-chart';
  title?: string;
  columns?: string[];
  rows?: string[][];
  cells?: GraphicCell[];
  bars?: { label: string; value: number }[];
}

export interface Turn {
  speaker: string;
  accent: Accent;
  text: string;
  translateVi?: string;
}

/** Where the words that give an answer away are: a block of a text (doc ≥ 0) or a line (doc = -1). */
export interface Ref {
  doc: number;
  block: number;
  turn: number;
  text: string;
}

export interface VocabItem {
  word: string;
  ipa?: string;
  meaningVi: string;
}

/** A question of the exam room: no answer. */
export interface PaperQuestion {
  id: string;
  no: number;
  number: number;
  prompt?: string;
  options: string[];
}

/** A group as the exam room shows it. */
export interface PaperGroup {
  key: string;
  part: ToeicPart;
  number: number;
  imageUrl: string | null;
  docs?: ToeicDoc[];
  script?: Turn[];
  graphic: Graphic | null;
  questions: PaperQuestion[];
}

// --- library and tests ---

export interface TestState {
  status: 'new' | 'in_progress' | 'submitted';
  attemptId?: string;
  position?: number;
  total: number | null;
  correct: number | null;
  questions?: number;
  at: string | null;
}

export interface TestCard {
  key: string;
  title: string;
  kind: TestKind;
  questions: number;
  minutes: number;
  parts: number;
  complete: boolean;
  locked: boolean;
  state: TestState;
}

export interface ToeicGoal {
  target: number;
  examDate?: string;
  daysLeft: number | null;
}

export interface ToeicLibrary {
  tests: TestCard[];
  goal: ToeicGoal | null;
  latest: {
    attemptId: string;
    total: number;
    listening: number;
    reading: number;
    at: string;
  } | null;
  recent: { attemptId: string; title: string; correct: number; total: number; at: string }[];
  mistakes: number | null;
  /** Published questions of each part, index 1..7. */
  partCounts: number[];
  guest: boolean;
}

export interface PartInfo {
  part: ToeicPart;
  questions: number;
  tags: string[];
}

export interface AttemptLine {
  id: string;
  mode: AttemptMode;
  parts: ToeicPart[];
  correct: number;
  total: number;
  scaled: number | null;
  retake: boolean;
  durationMs: number;
  submittedAt: string;
}

export interface TestDetail {
  key: string;
  title: string;
  kind: TestKind;
  minutes: number;
  questions: number;
  complete: boolean;
  locked: boolean;
  parts: PartInfo[];
  attempts: AttemptLine[];
  inProgress: string | null;
}

// --- attempts ---

export interface AttemptSource {
  kind: SourceKind;
  testKey?: string;
  part?: ToeicPart;
  tag?: string;
  fromAttempt?: string;
  size?: number;
  count?: number;
}

export interface StartAttempt {
  source: AttemptSource;
  mode?: AttemptMode;
  parts?: ToeicPart[];
  timeLimitMin?: number;
}

export interface AttemptView {
  id: string;
  title: string;
  mode: AttemptMode;
  status: 'in_progress' | 'submitted';
  source: AttemptSource;
  phase?: Section;
  parts: ToeicPart[];
  timeLimitMs: number;
  remainingMs: number;
  elapsedMs: number;
  listeningSegment: number;
  position: string;
  answers: Record<string, number>;
  flags: string[];
  paper: PaperGroup[];
}

/** What the exam room saves; every field may be left out. Answers are merged, -1 clears one. */
export interface SaveProgress {
  answers?: Record<string, number>;
  flags?: string[];
  position?: string;
  remainingMs?: number;
  elapsedMs?: number;
  phase?: Section;
  listeningSegment?: number;
}

export interface SectionScore {
  correct: number;
  total: number;
  scaled: number;
}

export interface NumberRef {
  number: number;
  id: string;
  state: Outcome;
}

export interface TagRow {
  tag: string;
  labelVi: string;
  right: number;
  wrong: number;
  skipped: number;
  questions: NumberRef[];
}

export interface AnswerRow {
  number: number;
  id: string;
  part: ToeicPart;
  correct: string;
  chosen: string;
  state: Outcome;
}

export interface ResultView {
  id: string;
  title: string;
  mode: AttemptMode;
  source: AttemptSource;
  parts: ToeicPart[];
  retake: boolean;
  counted: boolean;
  correct: number;
  wrong: number;
  skipped: number;
  questions: number;
  accuracy: number;
  durationMs: number;
  listening: SectionScore | null;
  reading: SectionScore | null;
  total: number | null;
  goal: number | null;
  byPart: { part: ToeicPart; tags: TagRow[] }[];
  answers: AnswerRow[];
  submittedAt: string;
}

export interface TagRef {
  tag: string;
  labelVi: string;
  grammarPointId?: string;
}

export interface QuestionAnswer {
  id: string;
  no: number;
  number: number;
  prompt?: string;
  options: string[];
  answer: number;
  chosen: number;
  state: Outcome;
  evidence: Ref[];
  explainVi: string;
  translateVi?: string;
  tags: TagRef[];
  vocab: VocabItem[];
}

export interface QuestionDetail {
  attemptId: string;
  title: string;
  group: {
    key: string;
    part: ToeicPart;
    imageUrl: string | null;
    docs?: ToeicDoc[];
    script?: Turn[];
    graphic: Graphic | null;
  };
  question: QuestionAnswer;
  prev?: string;
  next?: string;
}

// --- mistakes ---

export interface MistakeItem {
  questionId: string;
  groupKey: string;
  part: ToeicPart;
  tags: string[];
  streak: number;
  prompt: string;
}

export interface MistakesView {
  items: MistakeItem[];
  counts: Record<string, number>;
  need: number;
}

export interface RoundItem {
  questionId: string;
  streak: number;
  group: PaperGroup;
}

export interface CheckReply {
  right: boolean;
  answer: number;
  streak: number;
  removed: boolean;
  explainVi: string;
  evidence: Ref[];
  translateVi?: string;
}

// --- admin ---

export interface GroupReport {
  key: string;
  part: ToeicPart;
  questions: number;
  status: 'ok' | 'duplicate' | 'replace' | 'error';
  fields?: Record<string, string>;
}

export interface ImportReport {
  dryRun: boolean;
  created: string[];
  replaced: string[];
  duplicates: string[];
  rescored: number;
  listeningMinutes: number | null;
  reports: GroupReport[];
}

export interface GroupAdmin {
  key: string;
  part: ToeicPart;
  level: number;
  questions: number;
  tags: string[];
  status: ToeicStatus;
  source: string;
  version: number;
  needsPhoto: boolean;
  hasPhoto: boolean;
  /** How many of the group's picture slots have a picture (Part 1, 6, 7). */
  pictures: number;
  pictureSlots: number;
  tests: string[];
  title: string;
}

/** A whole group (admin preview), as imported. */
export interface GroupFull {
  key: string;
  part: ToeicPart;
  level: number;
  status: ToeicStatus;
  source?: string;
  version: number;
  scene?: string;
  script?: Turn[];
  graphic?: Graphic | null;
  docs?: (Omit<ToeicDoc, 'imageUrl' | 'meta'> & { meta?: KV[]; imageSlot?: string })[];
  questions: {
    no: number;
    prompt?: string;
    options: string[];
    answer: number;
    evidence?: Ref[];
    explainVi: string;
    translateVi?: string;
    tags: string[];
    vocab?: VocabItem[];
  }[];
  updatedAt: string;
  imageUrl: string | null;
}

export interface TestCheck {
  parts: { part: ToeicPart; have: number; need: number }[];
  missingGroups: string[];
  draftGroups: string[];
  missingPhotos: { groupKey: string; number: number }[];
  graphics: number;
  docs: number;
  listeningMinutes: number;
  complete: boolean;
  ready: boolean;
}

export interface TestAdmin {
  key: string;
  title: string;
  kind: TestKind;
  order: number;
  groups: string[];
  status: ToeicStatus;
  updatedAt: string;
  check: TestCheck;
  groupRows?: GroupAdmin[];
}

export interface TestInput {
  key: string;
  title: string;
  kind: TestKind;
  order: number;
  groups: string[];
}

/** "Ghép đề tự động": what the app takes for one part. */
export interface ComposePart {
  part: ToeicPart;
  need: number;
  have: number;
  groups: number;
  reused: number;
  available: number;
  filled: boolean;
}

/** The test the app would make from the published groups. */
export interface ComposePlan {
  key: string;
  title: string;
  kind: TestKind;
  order: number;
  parts: ComposePart[];
  groups: string[];
  complete: boolean;
}

export interface ComposeInput {
  kind: TestKind;
  title?: string;
  partial?: boolean;
  dryRun?: boolean;
}

/** "Nhờ AI soạn": the drafts kept and the groups dropped with why. */
export interface DraftReport {
  created: string[];
  dropped: {
    key: string;
    part: ToeicPart;
    questions: number;
    status: string;
    fields?: Record<string, string>;
  }[];
}

export interface DraftInput {
  part: ToeicPart;
  level: number;
  topic: string;
}

/** A picture slot of a group: the Part 1 photo or the picture of a Part 6/7 text. */
export interface ImageSlot {
  slot: string;
  label: string;
  has: boolean;
  url: string;
}

export interface GroupImages {
  slots: ImageSlot[];
  codex: boolean;
}

/** The batch of missing Part 1 photos. */
export interface PhotoBatch {
  running: boolean;
  stopping: boolean;
  engine: string;
  testKey: string;
  /** The part of the bank drawn (1, 6, 7); 0 for a test's batch. */
  part: number;
  /** The groups of a batch started for some groups (after "Nhờ AI soạn"). */
  keys: string[];
  total: number;
  drawing: string;
  saved: string[];
  failed: Record<string, string>;
  stopReason: string;
  startedAt: string;
}

/** What "Xoá" did: the groups removed, the ones kept because a test or a learner's attempt uses them, and unknown keys. */
export interface DeleteGroupsReport {
  deleted: string[];
  blocked: { key: string; reason: string }[];
  missing: string[];
}
