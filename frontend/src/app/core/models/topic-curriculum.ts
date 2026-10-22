import { Level } from './lesson';

/** One topic of the suggested curriculum of a level, and why it comes at that place. */
export interface SuggestedTopic {
  /** Name as in the catalogue (the seeded topics), so it matches an existing topic. */
  name: string;
  /** Other names an admin may have given the same topic. */
  aliases?: string[];
  /** Why it comes here: what it builds on or prepares, and the grammar it suits. */
  why: string;
  /** Description used when the admin creates the topic from the suggestion. */
  description: string;
}

/**
 * Suggested order of the topics per level, from the CEFR descriptors (A1 is the fixed course of the
 * backend, backend/internal/curriculum/a1.json, since 2026-10-22): A1–A2 move from the learner
 * outwards (self, family, home, day, food, town), each topic reusing the words and grammar of the
 * ones before; B1 turns to experiences and opinions; B2 and above to abstract and public topics.
 * Grammar in "why" follows the order of the app's grammar syllabus.
 */
export const TOPIC_CURRICULUM: Record<Level, SuggestedTopic[]> = {
  A1: [
    {
      name: 'Chào hỏi và giới thiệu bản thân',
      why: 'Câu đầu tiên ai cũng cần: tên, tuổi, quê. Đi cùng động từ to be.',
      description: 'Chào hỏi, tên, tuổi, quê quán',
    },
    {
      name: 'Gia đình',
      aliases: ['Người thân'],
      why: 'Từ vựng gần gũi nhất; luyện have/has và tính từ sở hữu.',
      description: 'Người thân trong gia đình',
    },
    {
      name: 'Thời gian và ngày tháng',
      why: 'Giờ, thứ, tháng với at / on / in và từ để hỏi: cần trước khi kể thói quen.',
      description: 'Giờ giấc, thứ, ngày, tháng',
    },
    {
      name: 'Nhà cửa và đồ đạc',
      aliases: ['Nhà cửa'],
      why: 'There is / there are và giới từ chỉ nơi chốn.',
      description: 'Các phòng và đồ đạc trong nhà',
    },
    {
      name: 'Hoạt động hằng ngày',
      aliases: ['Thói quen hằng ngày'],
      why: 'Hiện tại đơn và trạng từ tần suất, dùng lại giờ giấc vừa học.',
      description: 'Thói quen từ sáng đến tối',
    },
    {
      name: 'Đồ ăn',
      aliases: ['Đồ ăn và đồ uống', 'Ăn uống'],
      why: 'A / an / the và danh từ số nhiều; nói ăn gì, gọi món.',
      description: 'Các món ăn quen thuộc',
    },
    {
      name: 'Mua sắm',
      aliases: ['Cửa hàng'],
      why: 'This / that / these / those, hỏi giá: dùng lại đồ ăn và số đếm.',
      description: 'Đi chợ, siêu thị, hỏi giá',
    },
    {
      name: 'Trường học',
      why: 'Hiện tại đơn: phủ định và câu hỏi (I don’t…, Do you…?).',
      description: 'Lớp học, môn học, thời khoá biểu',
    },
    {
      name: 'Âm nhạc',
      why: 'Like / love + V-ing và and / but / because; dễ gắn với bài hát.',
      description: 'Nhạc cụ, thể loại nhạc, buổi hoà nhạc',
    },
    {
      name: 'Thời tiết',
      why: 'Ôn hiện tại đơn và trạng từ tần suất; mở đầu cho chuyện mùa, du lịch ở A2.',
      description: 'Thời tiết và các mùa',
    },
    {
      name: 'Con vật',
      why: 'Can / can’t chỉ khả năng; từ vựng dễ nhớ, có hình.',
      description: 'Con vật nuôi và hoang dã',
    },
    {
      name: 'Thành phố và chỉ đường',
      why: 'Câu mệnh lệnh và giới từ nơi chốn: hỏi và chỉ đường.',
      description: 'Địa điểm trong thành phố, hỏi đường',
    },
  ],
  A2: [
    {
      name: 'Số đếm',
      why: 'Nền cho tuổi, giờ, giá tiền, số điện thoại ở các chủ đề sau.',
      description: 'Số đếm, tuổi, số điện thoại',
    },
    {
      name: 'Quần áo',
      why: 'This / these, màu sắc, kích cỡ: chuẩn bị cho Mua sắm.',
      description: 'Quần áo, màu sắc, kích cỡ',
    },
    {
      name: 'Sở thích và thời gian rảnh',
      aliases: ['Sở thích'],
      why: 'Like + V-ing, trạng từ tần suất; chủ đề dễ nói về mình.',
      description: 'Sở thích, việc làm lúc rảnh',
    },
    {
      name: 'Bạn bè và các mối quan hệ',
      why: 'Kể về người khác; luyện quá khứ đơn qua chuyện gặp gỡ.',
      description: 'Bạn bè, làm quen, giữ liên lạc',
    },
    {
      name: 'Miêu tả người',
      why: 'Tính từ và so sánh hơn, so sánh nhất.',
      description: 'Ngoại hình và tính cách',
    },
    {
      name: 'Giao thông',
      why: 'Phương tiện, by bus / on foot; nối tiếp chỉ đường.',
      description: 'Phương tiện đi lại',
    },
    {
      name: 'Nhà hàng và gọi món',
      why: 'Would like, câu hỏi lịch sự; dùng lại từ vựng đồ ăn A1.',
      description: 'Đặt bàn, gọi món, thanh toán',
    },
    {
      name: 'Nghề nghiệp',
      why: 'Nói về công việc của người thân và ước mơ sau này.',
      description: 'Các nghề và nơi làm việc',
    },
    {
      name: 'Sức khoẻ',
      why: 'Should / shouldn’t để khuyên; bộ phận cơ thể, triệu chứng.',
      description: 'Ốm đau, khám bệnh, lời khuyên',
    },
    {
      name: 'Thể thao',
      why: 'Quá khứ đơn khi kể trận đấu; play / do / go.',
      description: 'Các môn thể thao, luyện tập',
    },
    {
      name: 'Du lịch',
      why: 'Be going to / will cho kế hoạch; kể chuyến đi bằng quá khứ đơn.',
      description: 'Kỳ nghỉ, chuẩn bị chuyến đi',
    },
    {
      name: 'Tết',
      aliases: ['Lễ hội', 'Lễ kỷ niệm'],
      why: 'Văn hoá quen thuộc nên dễ diễn đạt; mô tả phong tục.',
      description: 'Phong tục ngày Tết',
    },
    {
      name: 'Cảm xúc',
      why: 'Tính từ -ed / -ing; chuẩn bị cho việc nêu ý kiến ở B1.',
      description: 'Vui, buồn, lo lắng, hào hứng',
    },
  ],
  B1: [
    {
      name: 'Công việc',
      aliases: ['Công việc và sự nghiệp'],
      why: 'Hiện tại hoàn thành để kể kinh nghiệm; phỏng vấn, CV.',
      description: 'Đi làm, phỏng vấn, đồng nghiệp',
    },
    {
      name: 'Khách sạn và sân bay',
      why: 'Tình huống thực tế khi đi xa; câu hỏi gián tiếp lịch sự.',
      description: 'Làm thủ tục, đặt phòng, xử lý sự cố',
    },
    {
      name: 'Công nghệ, internet và mạng xã hội',
      why: 'Chủ đề gần gũi để bắt đầu nêu ưu và nhược điểm.',
      description: 'Điện thoại thông minh, mạng xã hội',
    },
    {
      name: 'Phim ảnh',
      why: 'Kể lại nội dung, nêu cảm nhận; mệnh đề quan hệ.',
      description: 'Thể loại phim, review phim',
    },
    {
      name: 'Sách và đọc',
      why: 'Kể chuyện bằng quá khứ tiếp diễn và quá khứ đơn.',
      description: 'Thể loại sách, thói quen đọc',
    },
    {
      name: 'Tính cách',
      why: 'Từ vựng sắc thái hơn Miêu tả người; nêu ví dụ minh hoạ.',
      description: 'Tính cách và cách cư xử',
    },
    {
      name: 'Thiên nhiên và địa lý',
      why: 'Câu bị động đơn giản; so sánh vùng miền, quốc gia.',
      description: 'Núi, sông, khí hậu, vùng miền',
    },
    {
      name: 'Môi trường',
      why: 'Câu điều kiện loại 1, nêu giải pháp; mở đường cho B2.',
      description: 'Ô nhiễm, rác thải, bảo vệ môi trường',
    },
    {
      name: 'Điện thoại và giao tiếp',
      why: 'Gọi điện, để lại lời nhắn; câu tường thuật đơn giản.',
      description: 'Gọi điện, nhắn tin, email',
    },
    {
      name: 'Nấu ăn',
      why: 'Mô tả quy trình theo bước; từ nối first, then, finally.',
      description: 'Công thức, cách chế biến',
    },
  ],
  B2: [
    {
      name: 'Báo chí và truyền thông',
      why: 'Phân biệt sự thật và ý kiến; câu bị động, câu tường thuật.',
      description: 'Tin tức, báo chí, truyền hình, quảng cáo',
    },
    {
      name: 'Giáo dục đại học và du học',
      why: 'Lập luận ưu nhược điểm; câu điều kiện loại 2.',
      description: 'Đại học, học bổng, du học',
    },
    {
      name: 'Kinh doanh và văn phòng',
      why: 'Ngôn ngữ trang trọng: email, họp, đàm phán.',
      description: 'Công ty, họp hành, đàm phán',
    },
    {
      name: 'Khoa học và phát minh',
      why: 'Giải thích quy trình, nguyên nhân – kết quả.',
      description: 'Phát minh, nghiên cứu khoa học',
    },
    {
      name: 'Nghệ thuật và văn hoá',
      why: 'Bày tỏ quan điểm có lý lẽ; từ vựng trừu tượng.',
      description: 'Hội hoạ, bảo tàng, di sản',
    },
    {
      name: 'Lịch sử',
      why: 'Quá khứ hoàn thành; kể sự kiện theo trình tự.',
      description: 'Sự kiện và nhân vật lịch sử',
    },
    {
      name: 'Thời trang',
      why: 'Xu hướng, tiêu dùng; tranh luận nhẹ nhàng.',
      description: 'Xu hướng thời trang, tiêu dùng',
    },
  ],
  C1: [
    {
      name: 'Xã hội và các vấn đề toàn cầu',
      why: 'Lập luận nhiều chiều; đảo ngữ, câu điều kiện hỗn hợp.',
      description: 'Bất bình đẳng, di cư, đô thị hoá',
    },
    {
      name: 'Pháp luật và tội phạm',
      why: 'Từ vựng chuyên ngành; câu bị động phức tạp.',
      description: 'Luật pháp, tội phạm, hình phạt',
    },
    {
      name: 'Tâm lý học',
      why: 'Diễn đạt sắc thái, giả định; từ vựng học thuật.',
      description: 'Hành vi, cảm xúc, thói quen con người',
    },
  ],
  C2: [
    {
      name: 'Đạo đức và triết lý',
      why: 'Tranh luận về giá trị; diễn đạt tinh tế, hàm ý.',
      description: 'Các câu hỏi về đạo đức trong đời sống',
    },
    {
      name: 'Kinh tế toàn cầu',
      why: 'Phân tích số liệu, xu hướng; ngôn ngữ báo chí chuyên sâu.',
      description: 'Thương mại, tài chính, toàn cầu hoá',
    },
  ],
};

/** Tone marks placed the old way ("khoẻ") and the new way ("khỏe") compare equal. */
const TONE_OLD: [string, string][] = [
  ['oà', 'òa'],
  ['oá', 'óa'],
  ['oả', 'ỏa'],
  ['oã', 'õa'],
  ['oạ', 'ọa'],
  ['oè', 'òe'],
  ['oé', 'óe'],
  ['oẻ', 'ỏe'],
  ['oẽ', 'õe'],
  ['oẹ', 'ọe'],
  ['uỳ', 'ùy'],
  ['uý', 'úy'],
  ['uỷ', 'ủy'],
  ['uỹ', 'ũy'],
  ['uỵ', 'ụy'],
];

/** The form topic names are matched in: lowercase, single-spaced, tone marks placed the new way. */
export function topicKey(name: string): string {
  let key = name.trim().toLowerCase().replace(/\s+/g, ' ').normalize('NFC');
  for (const [oldForm, newForm] of TONE_OLD) {
    key = key.split(oldForm).join(newForm);
  }
  return key;
}

/** Whether a topic named `name` is the suggested topic `s`. */
export function matchesSuggestion(name: string, s: SuggestedTopic): boolean {
  const key = topicKey(name);
  return [s.name, ...(s.aliases ?? [])].some((n) => topicKey(n) === key);
}

/** The level a topic named `name` is suggested for, or null when the curriculum does not list it. */
export function suggestedLevel(name: string): Level | null {
  for (const [level, list] of Object.entries(TOPIC_CURRICULUM) as [Level, SuggestedTopic[]][]) {
    if (list.some((s) => matchesSuggestion(name, s))) {
      return level;
    }
  }
  return null;
}
