import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Level } from '../../../../core/models/lesson';
import { Topic, TopicLevel } from '../../../../core/models/topic';
import {
  suggestedLevel,
  SuggestedTopic,
  TOPIC_CURRICULUM,
  topicKey,
} from '../../../../core/models/topic-curriculum';
import { TopicSuggestions } from './topic-suggestions';

const lv = (level: Level, lessonCount: number, position = 0): TopicLevel => ({
  level,
  lessonCount,
  roadmapCount: lessonCount,
  remaining: lessonCount,
  warning: lessonCount < 3,
  position,
});

function topic(id: string, name: string, levels: TopicLevel[] = []): Topic {
  return {
    id,
    name,
    description: '',
    levels,
    lessonCount: levels.reduce((n, l) => n + l.lessonCount, 0),
    createdAt: '',
    wordCount: 0,
    usedWordCount: 0,
  };
}

/** The catalogue of the live site: four A1 topics have lessons, one of them better at B2. */
const catalogue = [
  topic('daily', 'Hoạt động hằng ngày', [lv('A1', 9)]),
  topic('shop', 'Mua sắm', [lv('A1', 10)]),
  topic('music', 'Âm nhạc', [lv('A1', 5)]),
  topic('news', 'Báo chí và truyền thông', [lv('A1', 1)]),
  topic('family', 'Gia đình'),
  topic('store', 'Cửa hàng'),
  topic('health', 'Sức khỏe'),
];

describe('topic curriculum', () => {
  it('suggests each topic at one level only, with no name twice', () => {
    const keys = Object.values(TOPIC_CURRICULUM)
      .flat()
      .flatMap((s) => [s.name, ...(s.aliases ?? [])].map(topicKey));
    expect(new Set(keys).size).toBe(keys.length);
  });

  it('matches names whatever the case, spacing or tone mark placement', () => {
    expect(topicKey('  Sức  KHOẺ ')).toBe(topicKey('sức khỏe'));
    expect(suggestedLevel('sức khỏe')).toBe('A2');
    expect(suggestedLevel('Báo chí và truyền thông')).toBe('B2');
    expect(suggestedLevel('Quân đội')).toBeNull();
  });
});

describe('TopicSuggestions', () => {
  let fixture: ComponentFixture<TopicSuggestions>;
  let el: HTMLElement;
  let applied: { level: Level; topicIds: string[] }[];
  let created: SuggestedTopic[];

  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const rows = () => Array.from(el.querySelectorAll<HTMLElement>('.row'));
  const row = (name: string) =>
    rows().find((r) => text(r.querySelector('.name')).startsWith(name))!;
  const button = (label: string, root: ParentNode = el) =>
    Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label);

  const setup = async (topics: Topic[] = catalogue, level: Level = 'A1') => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
    fixture = TestBed.createComponent(TopicSuggestions);
    fixture.componentRef.setInput('topics', topics);
    fixture.componentRef.setInput('level', level);
    applied = [];
    created = [];
    fixture.componentInstance.apply.subscribe((a) => applied.push(a));
    fixture.componentInstance.create.subscribe((s) => created.push(s));
    el = fixture.nativeElement as HTMLElement;
    await fixture.whenStable();
  };

  it('lists the suggested order of the level, starting point first, topics with lessons highlighted', async () => {
    await setup();
    expect(rows().length).toBe(TOPIC_CURRICULUM.A1.length);
    expect(text(rows()[0].querySelector('.name'))).toBe(
      'Chào hỏi và giới thiệu bản thân Bắt đầu từ đây',
    );
    expect(text(el.querySelector('.progress'))).toBe(
      `3/${TOPIC_CURRICULUM.A1.length} chủ đề gợi ý của A1 đã có bài.`,
    );

    expect(row('Hoạt động hằng ngày').dataset['status']).toBe('ready');
    expect(text(row('Hoạt động hằng ngày').querySelector('.badge'))).toBe('Đã có 9 bài');
    // Mua sắm, not its alias Cửa hàng, which is a topic too.
    expect(text(row('Mua sắm').querySelector('.badge'))).toBe('Đã có 10 bài');
    // A topic without lessons at the level links to the lesson generator.
    expect(row('Gia đình').dataset['status']).toBe('empty');
    expect(row('Gia đình').querySelector('a')!.getAttribute('href')).toBe(
      '/admin/roadmap?topicId=family&level=A1&generate=1',
    );
    // A topic the catalogue does not have offers to create it.
    expect(row('Nhà cửa và đồ đạc').dataset['status']).toBe('missing');
    button('Tạo chủ đề Nhà cửa và đồ đạc', row('Nhà cửa và đồ đạc'))!.click();
    expect(created.map((s) => s.name)).toEqual(['Nhà cửa và đồ đạc']);
  });

  it('names the topics with lessons that suit another level', async () => {
    await setup();
    const note = el.querySelector('.misplaced')!;
    expect(Array.from(note.querySelectorAll('li')).map((li) => text(li))).toEqual([
      'Báo chí và truyền thông: gợi ý cho B2',
    ]);
  });

  it('applies the suggested order: suggested topics with lessons first, then the others as they are', async () => {
    await setup();
    button('Áp dụng thứ tự gợi ý cho A1')!.click();
    // Hoạt động hằng ngày comes before Mua sắm in the curriculum; the rest keep their order (by name).
    expect(applied).toEqual([{ level: 'A1', topicIds: ['daily', 'shop', 'music', 'news'] }]);
  });

  it('says when the level already follows the suggestion', async () => {
    await setup([
      topic('daily', 'Hoạt động hằng ngày', [lv('A1', 9, 1)]),
      topic('shop', 'Mua sắm', [lv('A1', 10, 2)]),
    ]);
    expect(button('Áp dụng thứ tự gợi ý cho A1')!.disabled).toBe(true);
    expect(text(el.querySelector('.actions'))).toContain('đang theo thứ tự gợi ý');
  });

  it('shows another level on its own, without changing the page', async () => {
    await setup();
    button('B2')!.click();
    await fixture.whenStable();
    expect(text(rows()[0].querySelector('.name'))).toContain('Báo chí và truyền thông');
    // Báo chí has lessons at A1 only, not at B2.
    expect(row('Báo chí và truyền thông').dataset['status']).toBe('empty');
    expect(button('Áp dụng thứ tự gợi ý cho B2')!.disabled).toBe(true);
  });
});
