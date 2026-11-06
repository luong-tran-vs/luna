import { BankImageBatch } from '../../../core/models/word-bank';
import { batchScope } from './image-batch.service';

describe('batchScope', () => {
  const b = (over: Partial<BankImageBatch>): BankImageBatch => ({
    running: true, stopping: false, stopped: false, engine: 'gemini', ticked: false, topics: [], total: 5, done: 0,
    drawing: [], saved: [], failures: [], ...over,
  });

  it('names the filter topic', () => {
    expect(batchScope(b({ topicName: 'Nhà cửa', topics: ['Chào hỏi', 'Nhà cửa'] }))).toBe('các từ thiếu ảnh của chủ đề "Nhà cửa"');
  });

  it('names the topics of every missing word, or of the ticked ones', () => {
    expect(batchScope(b({ search: 'ho', topics: ['Chào hỏi', 'Nhà cửa'] }))).toBe(
      'mọi từ thiếu ảnh khớp "ho" (chủ đề: Chào hỏi, Nhà cửa)',
    );
    expect(batchScope(b({ ticked: true, search: 'ignored', topics: ['A', 'B', 'C', 'D', 'E'] }))).toBe(
      '5 từ đã chọn (5 chủ đề: A, B, C và 2 chủ đề khác)',
    );
    expect(batchScope(b({ ticked: true, total: 2 }))).toBe('2 từ đã chọn');
    expect(batchScope(b({ ticked: true, total: 12, topicName: 'Tính cách' }))).toBe('12 từ đã chọn của chủ đề "Tính cách"');
  });
});
