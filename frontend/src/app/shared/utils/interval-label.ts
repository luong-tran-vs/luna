const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Short Vietnamese label for a review interval in seconds ("10 phút", "3 ngày"). */
export function intervalLabel(seconds: number): string {
  if (seconds < HOUR) {
    return `${Math.max(1, Math.round(seconds / MINUTE))} phút`;
  }
  if (seconds < DAY) {
    return `${Math.round(seconds / HOUR)} giờ`;
  }
  const days = seconds / DAY;
  if (days < 30) {
    return `${Math.round(days)} ngày`;
  }
  if (days < 365) {
    return `${Math.round(days / 30)} tháng`;
  }
  return `${Math.round(days / 365)} năm`;
}
