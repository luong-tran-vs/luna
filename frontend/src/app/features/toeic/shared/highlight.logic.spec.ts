import {
  addHighlight,
  blockId,
  highlightsOf,
  parseHighlights,
  removeHighlight,
} from './highlight.logic';

describe('highlights', () => {
  const b = blockId('p7-a', 0, 1);

  it('adds trimmed words once per block', () => {
    let list = addHighlight([], { block: b, text: ' opens at 9 ' });
    list = addHighlight(list, { block: b, text: 'opens at 9' });
    list = addHighlight(list, { block: b, text: '   ' });
    expect(list).toEqual([{ block: 'p7-a/0/1', text: 'opens at 9' }]);
  });

  it('removes what overlaps the selection in that block only', () => {
    let list = addHighlight([], { block: b, text: 'opens at 9' });
    list = addHighlight(list, { block: 'other', text: 'opens at 9' });
    list = removeHighlight(list, { block: b, text: 'at' });
    expect(highlightsOf(list, b)).toEqual([]);
    expect(highlightsOf(list, 'other')).toEqual(['opens at 9']);
  });

  it('reads only well-formed lists', () => {
    expect(parseHighlights(null)).toEqual([]);
    expect(parseHighlights('nope')).toEqual([]);
    expect(parseHighlights('[{"block":"a","text":"b"},{"x":1}]')).toEqual([
      { block: 'a', text: 'b' },
    ]);
  });
});
