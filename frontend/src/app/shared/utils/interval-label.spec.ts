import { intervalLabel } from './interval-label';

describe('intervalLabel', () => {
  it.each([
    [30, '1 phút'],
    [60, '1 phút'],
    [600, '10 phút'],
    [3 * 3600, '3 giờ'],
    [2 * 86400, '2 ngày'],
    [45 * 86400, '2 tháng'],
    [400 * 86400, '1 năm'],
  ])('%i seconds → %s', (seconds, label) => {
    expect(intervalLabel(seconds)).toBe(label);
  });
});
