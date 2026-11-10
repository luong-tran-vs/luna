/**
 * How each irregular verb is used (added 2026-11-10, design/de-xuat-dong-tu.png parts 2 and 4):
 * a past simple sentence (V2) with a time marker, a perfect sentence (have/has/had + V3), and the
 * mistake learners often make. Written by Claude, to be reviewed by the admin.
 */
export interface VerbUsage {
  /** Past simple: the V2 form with a time marker such as "yesterday" or "last week". */
  past: { en: string; vi: string };
  /** Perfect: have / has / had + the V3 form. */
  perfect: { en: string; vi: string };
  /** The mistake learners often make with the verb, in Vietnamese, one short line. */
  mistakeVi: string;
}

export const VERB_USAGE: Readonly<Record<string, VerbUsage>> = {
  abide: {
    past: {
      en: 'She abided by her promise last year.',
      vi: 'Năm ngoái cô ấy đã giữ đúng lời hứa.',
    },
    perfect: { en: 'He has always abided by the law.', vi: 'Anh ấy luôn tuân thủ pháp luật.' },
    mistakeVi: 'abide by (tuân theo) thường dùng abided; abode là dạng văn chương, ít gặp.',
  },
  arise: {
    past: {
      en: 'A problem arose at work yesterday.',
      vi: 'Hôm qua ở chỗ làm đã nảy sinh một vấn đề.',
    },
    perfect: {
      en: 'Several questions have arisen since the meeting.',
      vi: 'Từ sau cuộc họp đã nảy sinh vài câu hỏi.',
    },
    mistakeVi: 'Đừng nhầm arise (tự nảy sinh) với raise (nâng lên, có vật đi sau).',
  },
  awake: {
    past: {
      en: 'I awoke to the sound of rain last night.',
      vi: 'Tối qua tôi đã thức giấc vì tiếng mưa.',
    },
    perfect: { en: 'The baby has just awoken.', vi: 'Em bé vừa mới thức dậy.' },
    mistakeVi: 'awoke là V2, awoken là V3; không viết awaked.',
  },
  be: {
    past: { en: 'I was very tired last night.', vi: 'Tối qua tôi đã rất mệt.' },
    perfect: { en: 'I have never been to Japan.', vi: 'Tôi chưa bao giờ đến Nhật Bản.' },
    mistakeVi: 'was đi với I/he/she/it, were đi với you/we/they; V3 là been.',
  },
  bear: {
    past: {
      en: 'She bore the pain quietly last winter.',
      vi: 'Mùa đông năm ngoái cô ấy đã lặng lẽ chịu đựng cơn đau.',
    },
    perfect: {
      en: 'He has borne a lot of stress this year.',
      vi: 'Năm nay anh ấy đã chịu rất nhiều áp lực.',
    },
    mistakeVi: 'borne (V3, chịu đựng) khác born trong be born (được sinh ra).',
  },
  beat: {
    past: { en: 'Our team beat them last Sunday.', vi: 'Chủ nhật tuần trước đội tôi đã thắng họ.' },
    perfect: {
      en: 'Nobody has ever beaten him at chess.',
      vi: 'Chưa ai từng đánh bại anh ấy ở môn cờ vua.',
    },
    mistakeVi: 'V2 beat giữ nguyên, V3 là beaten; không viết beated.',
  },
  become: {
    past: { en: 'He became a teacher in 2019.', vi: 'Anh ấy đã trở thành giáo viên vào năm 2019.' },
    perfect: { en: 'The city has become very busy.', vi: 'Thành phố đã trở nên rất đông đúc.' },
    mistakeVi: 'V3 become viết giống V1, chỉ V2 là became.',
  },
  befall: {
    past: {
      en: 'A terrible storm befell the town last year.',
      vi: 'Năm ngoái một cơn bão khủng khiếp đã ập đến thị trấn.',
    },
    perfect: {
      en: 'Bad luck has befallen our family again.',
      vi: 'Vận rủi lại đến với gia đình tôi.',
    },
    mistakeVi: 'befell là V2, befallen là V3; từ trang trọng, đời thường dùng happen.',
  },
  begin: {
    past: {
      en: 'The film began at eight last night.',
      vi: 'Tối qua bộ phim đã bắt đầu lúc tám giờ.',
    },
    perfect: { en: 'The lesson has already begun.', vi: 'Bài học đã bắt đầu rồi.' },
    mistakeVi: 'Dễ nhầm V2 và V3: began là V2, begun là V3 (đi sau have).',
  },
  behold: {
    past: {
      en: 'We beheld a beautiful sunset last summer.',
      vi: 'Mùa hè năm ngoái chúng tôi đã ngắm một hoàng hôn tuyệt đẹp.',
    },
    perfect: {
      en: 'I have never beheld such a view.',
      vi: 'Tôi chưa bao giờ được ngắm cảnh như vậy.',
    },
    mistakeVi: 'Không viết beholded: V2 và V3 đều là beheld (giống hold – held).',
  },
  bend: {
    past: {
      en: 'He bent the metal bar last week.',
      vi: 'Tuần trước anh ấy đã bẻ cong thanh kim loại.',
    },
    perfect: {
      en: 'The tree has bent in the strong wind.',
      vi: 'Cái cây đã cong đi trong gió mạnh.',
    },
    mistakeVi: 'Không viết bended: bend – bent – bent (đổi d thành t).',
  },
  beset: {
    past: {
      en: 'Problems beset the project last year.',
      vi: 'Năm ngoái nhiều vấn đề đã bủa vây dự án.',
    },
    perfect: {
      en: 'Many worries have beset her recently.',
      vi: 'Gần đây nhiều nỗi lo đã bủa vây cô ấy.',
    },
    mistakeVi: 'beset giữ nguyên ở cả ba dạng, không thêm -ed.',
  },
  bespeak: {
    past: {
      en: 'She bespoke a dress from the tailor last month.',
      vi: 'Tháng trước cô ấy đã đặt thợ may một chiếc váy.',
    },
    perfect: {
      en: 'He has bespoken a table for tonight.',
      vi: 'Anh ấy đã đặt trước một bàn cho tối nay.',
    },
    mistakeVi: 'bespoke là V2, bespoken là V3; tính từ bespoke còn có nghĩa là may đo.',
  },
  bet: {
    past: {
      en: 'I bet ten dollars on the match yesterday.',
      vi: 'Hôm qua tôi đã cược mười đô la vào trận đấu.',
    },
    perfect: {
      en: 'He has bet all his money on that horse.',
      vi: 'Anh ấy đã cược hết tiền vào con ngựa đó.',
    },
    mistakeVi: 'Không viết betted: bet giữ nguyên ở cả ba dạng.',
  },
  bid: {
    past: {
      en: 'She bid fifty dollars for the old lamp yesterday.',
      vi: 'Hôm qua cô ấy đã trả giá năm mươi đô cho cây đèn cũ.',
    },
    perfect: {
      en: 'Nobody has bid on my bike yet.',
      vi: 'Chưa có ai trả giá chiếc xe đạp của tôi.',
    },
    mistakeVi: 'Nghĩa trả giá: bid giữ nguyên; bade/bidden là nghĩa cổ (chào, bảo).',
  },
  bind: {
    past: {
      en: 'She bound the old letters with a ribbon yesterday.',
      vi: 'Hôm qua cô ấy đã buộc những lá thư cũ bằng ruy băng.',
    },
    perfect: {
      en: 'They have bound the boxes together with tape.',
      vi: 'Họ đã buộc các thùng lại với nhau bằng băng dính.',
    },
    mistakeVi: 'Không viết binded: bind – bound – bound (đọc /baʊnd/).',
  },
  bite: {
    past: {
      en: 'A dog bit my little brother last week.',
      vi: 'Tuần trước một con chó đã cắn em trai tôi.',
    },
    perfect: { en: 'The mosquitoes have bitten me all over.', vi: 'Muỗi đã đốt tôi khắp người.' },
    mistakeVi: 'bit là V2, bitten là V3 (gấp đôi t); không viết bited.',
  },
  bleed: {
    past: { en: 'My nose bled a lot last night.', vi: 'Tối qua tôi đã bị chảy máu cam nhiều.' },
    perfect: {
      en: 'His knee has bled through the bandage.',
      vi: 'Đầu gối anh ấy đã chảy máu thấm qua băng.',
    },
    mistakeVi: 'Không viết bleeded: bleed – bled – bled (ee rút thành e).',
  },
  blow: {
    past: { en: 'The wind blew hard last night.', vi: 'Tối qua gió đã thổi rất mạnh.' },
    perfect: { en: 'The storm has blown the roof off.', vi: 'Cơn bão đã thổi bay mái nhà.' },
    mistakeVi: 'Không viết blowed: blow – blew – blown (giống grow, know).',
  },
  break: {
    past: {
      en: 'I broke my phone screen yesterday.',
      vi: 'Hôm qua tôi đã làm vỡ màn hình điện thoại.',
    },
    perfect: { en: 'Someone has broken the window.', vi: 'Ai đó đã làm vỡ cửa sổ.' },
    mistakeVi: 'broke là V2, broken là V3; “I’m broke” lại có nghĩa là hết tiền.',
  },
  breed: {
    past: {
      en: 'My uncle bred chickens when I was a child.',
      vi: 'Hồi tôi còn nhỏ, chú tôi đã nuôi gà.',
    },
    perfect: {
      en: 'They have bred dogs for many years.',
      vi: 'Họ đã nuôi chó giống nhiều năm nay.',
    },
    mistakeVi: 'Không viết breeded: breed – bred – bred.',
  },
  bring: {
    past: {
      en: 'She brought a cake to the office yesterday.',
      vi: 'Hôm qua cô ấy đã mang một chiếc bánh đến văn phòng.',
    },
    perfect: { en: 'Have you brought your passport?', vi: 'Bạn đã mang hộ chiếu chưa?' },
    mistakeVi: 'brought (mang đến) dễ nhầm với bought (mua) và thought (nghĩ).',
  },
  broadcast: {
    past: {
      en: 'The TV channel broadcast the match live last night.',
      vi: 'Tối qua kênh truyền hình đã phát trực tiếp trận đấu.',
    },
    perfect: { en: 'They have broadcast the news every hour.', vi: 'Họ đã phát bản tin mỗi giờ.' },
    mistakeVi: 'broadcast giữ nguyên ở cả ba dạng; broadcasted ít dùng.',
  },
  build: {
    past: {
      en: 'My grandfather built this house in 1980.',
      vi: 'Ông tôi đã xây ngôi nhà này năm 1980.',
    },
    perfect: { en: 'They have just built a new bridge.', vi: 'Họ vừa xây xong một cây cầu mới.' },
    mistakeVi: 'Không viết builded: build – built – built (đổi d thành t).',
  },
  burn: {
    past: { en: 'I burnt the toast this morning.', vi: 'Sáng nay tôi đã làm cháy bánh mì nướng.' },
    perfect: { en: 'She has burnt the rice again.', vi: 'Cô ấy lại làm cháy cơm rồi.' },
    mistakeVi: 'burnt (Anh) và burned (Mỹ) đều đúng ở cả V2 và V3.',
  },
  burst: {
    past: {
      en: 'The balloon burst with a loud bang yesterday.',
      vi: 'Hôm qua quả bóng bay đã nổ đánh bốp.',
    },
    perfect: {
      en: 'A water pipe has burst in the kitchen.',
      vi: 'Một ống nước trong bếp đã bị vỡ.',
    },
    mistakeVi: 'Không viết bursted: burst giữ nguyên ở cả ba dạng.',
  },
  buy: {
    past: {
      en: 'We bought a new sofa last weekend.',
      vi: 'Cuối tuần trước chúng tôi đã mua một chiếc ghế sofa mới.',
    },
    perfect: { en: 'I have already bought the tickets.', vi: 'Tôi đã mua vé rồi.' },
    mistakeVi: 'bought (mua) khác brought (mang): đừng thêm r.',
  },
  cast: {
    past: {
      en: 'The director cast her in the main role last year.',
      vi: 'Năm ngoái đạo diễn đã chọn cô ấy vào vai chính.',
    },
    perfect: {
      en: 'They have cast a famous actor in the film.',
      vi: 'Họ đã chọn một diễn viên nổi tiếng cho bộ phim.',
    },
    mistakeVi: 'cast giữ nguyên ở cả ba dạng, không viết casted.',
  },
  catch: {
    past: {
      en: 'I caught the last bus last night.',
      vi: 'Tối qua tôi đã bắt kịp chuyến xe buýt cuối.',
    },
    perfect: { en: 'He has caught a cold.', vi: 'Anh ấy đã bị cảm lạnh.' },
    mistakeVi: 'Không viết catched: catch – caught – caught, đọc /kɔːt/.',
  },
  chide: {
    past: {
      en: 'My mother chid me for being late yesterday.',
      vi: 'Hôm qua mẹ đã mắng tôi vì đi muộn.',
    },
    perfect: {
      en: 'The teacher has chidden him twice this week.',
      vi: 'Tuần này thầy giáo đã quở trách cậu ấy hai lần.',
    },
    mistakeVi: 'chid, chidden là dạng cổ; ngày nay thường dùng chided.',
  },
  choose: {
    past: {
      en: 'She chose the blue dress yesterday.',
      vi: 'Hôm qua cô ấy đã chọn chiếc váy xanh.',
    },
    perfect: { en: 'Have you chosen a name for the baby?', vi: 'Bạn đã chọn tên cho em bé chưa?' },
    mistakeVi: 'chose (V2) đọc /tʃəʊz/, chosen là V3; đừng nhầm với choose.',
  },
  cleave: {
    past: {
      en: 'He clove the log in two last winter.',
      vi: 'Mùa đông năm ngoái anh ấy đã chẻ đôi khúc gỗ.',
    },
    perfect: {
      en: 'The axe has cloven the wood cleanly.',
      vi: 'Chiếc rìu đã chẻ khúc gỗ gọn gàng.',
    },
    mistakeVi: 'clove/cleft, cloven/cleft là dạng văn chương; đời thường dùng split.',
  },
  cling: {
    past: {
      en: 'The child clung to her mother yesterday.',
      vi: 'Hôm qua đứa bé đã bám chặt lấy mẹ.',
    },
    perfect: {
      en: 'The wet shirt has clung to his back all day.',
      vi: 'Chiếc áo ướt đã dính vào lưng anh ấy cả ngày.',
    },
    mistakeVi: 'Không viết clinged: cling – clung – clung.',
  },
  come: {
    past: {
      en: 'My cousin came to visit last Sunday.',
      vi: 'Chủ nhật tuần trước anh họ tôi đã đến chơi.',
    },
    perfect: { en: 'Spring has come early this year.', vi: 'Năm nay mùa xuân đã đến sớm.' },
    mistakeVi: 'V3 là come (giống V1), không phải came: I have come.',
  },
  cost: {
    past: {
      en: 'My new shoes cost fifty dollars last month.',
      vi: 'Tháng trước đôi giày mới của tôi có giá năm mươi đô.',
    },
    perfect: {
      en: 'The trip has cost us a lot of money.',
      vi: 'Chuyến đi đã tốn của chúng tôi rất nhiều tiền.',
    },
    mistakeVi: 'Không thêm -ed: cost giữ nguyên ở cả ba dạng.',
  },
  creep: {
    past: {
      en: 'The cat crept into the kitchen last night.',
      vi: 'Tối qua con mèo đã rón rén bò vào bếp.',
    },
    perfect: { en: 'A spider has crept under the door.', vi: 'Một con nhện đã bò vào dưới cửa.' },
    mistakeVi: 'Không viết creeped: creep – crept – crept.',
  },
  crow: {
    past: {
      en: 'The rooster crew at five this morning.',
      vi: 'Sáng nay con gà trống đã gáy lúc năm giờ.',
    },
    perfect: {
      en: 'The rooster has crowed three times already.',
      vi: 'Con gà trống đã gáy ba lần rồi.',
    },
    mistakeVi: 'V2 có crew hoặc crowed, nhưng V3 chỉ có crowed.',
  },
  cut: {
    past: { en: 'She cut her hair short last week.', vi: 'Tuần trước cô ấy đã cắt tóc ngắn.' },
    perfect: { en: 'I have already cut the bread.', vi: 'Tôi đã cắt bánh mì rồi.' },
    mistakeVi: 'Không viết cutted: cut giữ nguyên ở cả ba dạng.',
  },
  deal: {
    past: {
      en: 'We dealt with the problem yesterday.',
      vi: 'Hôm qua chúng tôi đã giải quyết vấn đề.',
    },
    perfect: {
      en: 'She has dealt with angry customers before.',
      vi: 'Cô ấy đã từng xử lý những khách hàng nóng tính.',
    },
    mistakeVi: 'dealt đọc /delt/ (e ngắn); không viết dealed.',
  },
  dig: {
    past: {
      en: 'The kids dug a hole in the sand yesterday.',
      vi: 'Hôm qua bọn trẻ đã đào một cái hố trên cát.',
    },
    perfect: {
      en: 'The dog has dug up the garden again.',
      vi: 'Con chó lại đào bới khu vườn rồi.',
    },
    mistakeVi: 'Không viết digged: dig – dug – dug.',
  },
  dive: {
    past: {
      en: 'He dived into the pool this morning.',
      vi: 'Sáng nay anh ấy đã lao xuống hồ bơi.',
    },
    perfect: { en: 'She has dived in the sea many times.', vi: 'Cô ấy đã lặn biển nhiều lần.' },
    mistakeVi: 'dived dùng cho cả V2 và V3; dove chỉ là V2 kiểu Mỹ.',
  },
  do: {
    past: { en: 'I did my homework last night.', vi: 'Tối qua tôi đã làm bài tập về nhà.' },
    perfect: { en: 'Have you done the washing-up yet?', vi: 'Bạn đã rửa bát xong chưa?' },
    mistakeVi: 'did là V2, done là V3; không nói “I have did”.',
  },
  draw: {
    past: { en: 'My son drew a cat yesterday.', vi: 'Hôm qua con trai tôi đã vẽ một con mèo.' },
    perfect: { en: 'She has drawn a map for us.', vi: 'Cô ấy đã vẽ cho chúng tôi một tấm bản đồ.' },
    mistakeVi: 'Không viết drawed: draw – drew – drawn.',
  },
  dream: {
    past: { en: 'I dreamt about flying last night.', vi: 'Tối qua tôi đã mơ thấy mình bay.' },
    perfect: { en: 'I have always dreamt of visiting Paris.', vi: 'Tôi luôn mơ được đến Paris.' },
    mistakeVi: 'dreamt đọc /dremt/; dreamed cũng đúng ở V2 và V3.',
  },
  drink: {
    past: {
      en: 'We drank lemon juice at lunch yesterday.',
      vi: 'Hôm qua chúng tôi đã uống nước chanh vào bữa trưa.',
    },
    perfect: {
      en: 'He has drunk three cups of coffee today.',
      vi: 'Hôm nay anh ấy đã uống ba cốc cà phê.',
    },
    mistakeVi: 'Không viết drinked: chỉ đổi nguyên âm i → a → u (drank, drunk).',
  },
  drive: {
    past: {
      en: 'My dad drove me to school this morning.',
      vi: 'Sáng nay bố đã lái xe đưa tôi đến trường.',
    },
    perfect: { en: 'Have you ever driven a truck?', vi: 'Bạn đã bao giờ lái xe tải chưa?' },
    mistakeVi: 'drove là V2, driven là V3; không viết drived.',
  },
  dwell: {
    past: {
      en: 'My grandparents dwelt in a small village long ago.',
      vi: 'Ngày xưa ông bà tôi đã sống ở một ngôi làng nhỏ.',
    },
    perfect: {
      en: 'They have dwelt in this valley for centuries.',
      vi: 'Họ đã sống ở thung lũng này hàng thế kỷ.',
    },
    mistakeVi: 'dwelt và dwelled đều đúng; ngày nay hay dùng live thay dwell.',
  },
  eat: {
    past: { en: 'We ate pho for breakfast this morning.', vi: 'Sáng nay chúng tôi đã ăn phở.' },
    perfect: { en: 'Have you eaten yet?', vi: 'Bạn ăn chưa?' },
    mistakeVi: 'ate là V2, eaten là V3; không viết eated.',
  },
  fall: {
    past: { en: 'My son fell off his bike yesterday.', vi: 'Hôm qua con trai tôi đã ngã xe đạp.' },
    perfect: { en: 'Prices have fallen this month.', vi: 'Tháng này giá cả đã giảm.' },
    mistakeVi: 'fell là V2, fallen là V3; đừng nhầm fell với felt (V2 của feel).',
  },
  feed: {
    past: { en: 'I fed the fish this morning.', vi: 'Sáng nay tôi đã cho cá ăn.' },
    perfect: { en: 'She has just fed the baby.', vi: 'Cô ấy vừa cho em bé ăn xong.' },
    mistakeVi: 'Không viết feeded: feed – fed – fed.',
  },
  feel: {
    past: { en: 'I felt sick after lunch yesterday.', vi: 'Hôm qua tôi đã thấy mệt sau bữa trưa.' },
    perfect: { en: 'I have never felt so happy.', vi: 'Tôi chưa bao giờ thấy hạnh phúc như vậy.' },
    mistakeVi: 'felt là V2/V3 của feel; đừng nhầm với fell (V2 của fall).',
  },
  fight: {
    past: {
      en: 'The two boys fought in the playground yesterday.',
      vi: 'Hôm qua hai cậu bé đã đánh nhau ở sân chơi.',
    },
    perfect: {
      en: 'They have fought about money for years.',
      vi: 'Họ đã cãi nhau vì tiền nhiều năm nay.',
    },
    mistakeVi: 'fought đọc /fɔːt/; không viết fighted.',
  },
  find: {
    past: {
      en: 'I found my lost keys this morning.',
      vi: 'Sáng nay tôi đã tìm thấy chùm chìa khoá bị mất.',
    },
    perfect: { en: 'Have you found a new job yet?', vi: 'Bạn đã tìm được việc mới chưa?' },
    mistakeVi: 'found là V2/V3 của find, nhưng found cũng là V1 “thành lập” (founded).',
  },
  fit: {
    past: {
      en: 'This jacket fit me perfectly last year.',
      vi: 'Năm ngoái chiếc áo khoác này đã vừa khít với tôi.',
    },
    perfect: {
      en: 'The new door has never fit properly.',
      vi: 'Cánh cửa mới chưa bao giờ vừa khít.',
    },
    mistakeVi: 'Mỹ dùng fit, Anh hay dùng fitted cho V2 và V3.',
  },
  flee: {
    past: {
      en: 'The family fled the war in 1975.',
      vi: 'Gia đình ấy đã chạy trốn chiến tranh năm 1975.',
    },
    perfect: {
      en: 'Thousands of people have fled the city.',
      vi: 'Hàng nghìn người đã chạy khỏi thành phố.',
    },
    mistakeVi: 'fled (chạy trốn, của flee) khác flew (bay, V2 của fly).',
  },
  fling: {
    past: {
      en: 'She flung her bag onto the bed last night.',
      vi: 'Tối qua cô ấy đã quăng túi lên giường.',
    },
    perfect: {
      en: 'He has flung his clothes all over the floor.',
      vi: 'Anh ấy đã vứt quần áo khắp sàn nhà.',
    },
    mistakeVi: 'Không viết flinged: fling – flung – flung.',
  },
  fly: {
    past: {
      en: 'We flew to Da Nang last summer.',
      vi: 'Mùa hè năm ngoái chúng tôi đã bay đến Đà Nẵng.',
    },
    perfect: {
      en: 'Have you ever flown in a helicopter?',
      vi: 'Bạn đã bao giờ đi trực thăng chưa?',
    },
    mistakeVi: 'flew là V2, flown là V3; không viết flied.',
  },
  forbear: {
    past: {
      en: 'He forbore from shouting at them yesterday.',
      vi: 'Hôm qua anh ấy đã kiềm chế không quát họ.',
    },
    perfect: {
      en: 'She has forborne to complain all week.',
      vi: 'Cả tuần nay cô ấy đã nhịn không phàn nàn.',
    },
    mistakeVi: 'forbore là V2, forborne là V3; từ trang trọng, đời thường dùng hold back.',
  },
  forbid: {
    past: {
      en: 'My parents forbade me to go out last night.',
      vi: 'Tối qua bố mẹ đã cấm tôi ra ngoài.',
    },
    perfect: {
      en: 'The school has forbidden phones in class.',
      vi: 'Nhà trường đã cấm dùng điện thoại trong lớp.',
    },
    mistakeVi: 'forbade (V2) đọc /fəˈbæd/; V3 là forbidden (gấp đôi d).',
  },
  forecast: {
    past: { en: 'The radio forecast heavy rain yesterday.', vi: 'Hôm qua đài đã dự báo mưa to.' },
    perfect: {
      en: 'Experts have forecast a hot summer.',
      vi: 'Các chuyên gia đã dự báo một mùa hè nóng.',
    },
    mistakeVi: 'forecast giữ nguyên ở cả ba dạng; forecasted cũng được chấp nhận.',
  },
  foresee: {
    past: {
      en: 'Nobody foresaw the problem last year.',
      vi: 'Năm ngoái không ai thấy trước vấn đề này.',
    },
    perfect: {
      en: 'We have foreseen most of the risks.',
      vi: 'Chúng tôi đã lường trước hầu hết rủi ro.',
    },
    mistakeVi: 'foresaw là V2, foreseen là V3 (như see – saw – seen).',
  },
  foretell: {
    past: {
      en: 'The old woman foretold his future last night.',
      vi: 'Tối qua bà lão đã đoán trước tương lai của anh ấy.',
    },
    perfect: {
      en: 'She has foretold many things correctly.',
      vi: 'Bà ấy đã đoán đúng nhiều điều.',
    },
    mistakeVi: 'Không viết foretelled: giống tell – told – told.',
  },
  forget: {
    past: { en: 'I forgot my umbrella yesterday.', vi: 'Hôm qua tôi đã quên ô.' },
    perfect: { en: 'I have forgotten her name.', vi: 'Tôi đã quên tên cô ấy rồi.' },
    mistakeVi: 'forgot là V2, forgotten là V3; không nói “I have forgot”.',
  },
  forgive: {
    past: { en: 'She forgave him last week.', vi: 'Tuần trước cô ấy đã tha thứ cho anh ấy.' },
    perfect: { en: 'Have you forgiven your brother yet?', vi: 'Bạn đã tha thứ cho anh trai chưa?' },
    mistakeVi: 'forgave là V2, forgiven là V3 (giống give – gave – given).',
  },
  forsake: {
    past: { en: 'He forsook his hometown in 1990.', vi: 'Năm 1990 anh ấy đã bỏ quê hương ra đi.' },
    perfect: { en: 'She has never forsaken her dreams.', vi: 'Cô ấy chưa bao giờ từ bỏ ước mơ.' },
    mistakeVi: 'forsook là V2, forsaken là V3; từ văn chương, đời thường dùng abandon.',
  },
  freeze: {
    past: {
      en: 'The water in the bucket froze last night.',
      vi: 'Tối qua nước trong xô đã đóng băng.',
    },
    perfect: { en: 'The pipes have frozen again.', vi: 'Đường ống lại bị đóng băng rồi.' },
    mistakeVi: 'froze là V2, frozen là V3; không viết freezed.',
  },
  get: {
    past: {
      en: 'I got a letter from my aunt yesterday.',
      vi: 'Hôm qua tôi đã nhận được thư của dì.',
    },
    perfect: { en: 'She has just got a new job.', vi: 'Cô ấy vừa có được công việc mới.' },
    mistakeVi: 'Anh dùng have got, Mỹ dùng have gotten; không viết getted.',
  },
  gild: {
    past: { en: 'They gilded the temple roof last year.', vi: 'Năm ngoái họ đã mạ vàng mái chùa.' },
    perfect: {
      en: 'The artist has gilded the frame.',
      vi: 'Người nghệ sĩ đã mạ vàng khung tranh.',
    },
    mistakeVi: 'gilded hay dùng hơn gilt; gilt thường là tính từ (mạ vàng).',
  },
  gird: {
    past: {
      en: 'The soldier girded his sword last night.',
      vi: 'Tối qua người lính đã đeo gươm vào.',
    },
    perfect: {
      en: 'She has girded herself for a hard day.',
      vi: 'Cô ấy đã chuẩn bị tinh thần cho một ngày vất vả.',
    },
    mistakeVi: 'girded/girt là từ cổ, hay gặp trong gird oneself (chuẩn bị tinh thần).',
  },
  give: {
    past: {
      en: 'My aunt gave me a book last Christmas.',
      vi: 'Giáng sinh năm ngoái dì đã tặng tôi một cuốn sách.',
    },
    perfect: { en: 'Have you given her the message?', vi: 'Bạn đã đưa lời nhắn cho cô ấy chưa?' },
    mistakeVi: 'gave là V2, given là V3; không nói “I have gave”.',
  },
  go: {
    past: { en: 'We went to the cinema last night.', vi: 'Tối qua chúng tôi đã đi xem phim.' },
    perfect: { en: 'She has gone to the market.', vi: 'Cô ấy đã đi chợ (chưa về).' },
    mistakeVi: 'gone (đã đi, chưa về) khác been (đã đi và về): have been to.',
  },
  grind: {
    past: {
      en: 'My dad ground the coffee beans this morning.',
      vi: 'Sáng nay bố đã xay hạt cà phê.',
    },
    perfect: { en: 'Have you ground the pepper yet?', vi: 'Bạn đã xay hạt tiêu chưa?' },
    mistakeVi: 'Không viết grinded: grind – ground – ground (trùng chữ ground: mặt đất).',
  },
  grow: {
    past: {
      en: 'My little sister grew five centimetres last year.',
      vi: 'Năm ngoái em gái tôi đã cao thêm năm phân.',
    },
    perfect: { en: 'These trees have grown very tall.', vi: 'Những cái cây này đã mọc rất cao.' },
    mistakeVi: 'Không viết growed: grow – grew – grown.',
  },
  hang: {
    past: {
      en: 'I hung the picture on the wall yesterday.',
      vi: 'Hôm qua tôi đã treo bức tranh lên tường.',
    },
    perfect: { en: 'She has hung the washing outside.', vi: 'Cô ấy đã phơi quần áo ở ngoài.' },
    mistakeVi: 'Treo đồ vật: hung; treo cổ: hanged (có quy tắc).',
  },
  have: {
    past: { en: 'We had pizza for dinner last night.', vi: 'Tối qua chúng tôi đã ăn pizza.' },
    perfect: { en: 'She has had a cold since Monday.', vi: 'Cô ấy bị cảm từ thứ Hai.' },
    mistakeVi: 'Không viết haved: have – had – had; had had là quá khứ hoàn thành.',
  },
  hear: {
    past: {
      en: 'I heard a strange noise last night.',
      vi: 'Tối qua tôi đã nghe thấy một tiếng động lạ.',
    },
    perfect: { en: 'Have you heard the news?', vi: 'Bạn đã nghe tin chưa?' },
    mistakeVi: 'heard đọc /hɜːd/, khác hear /hɪə/; không viết heared.',
  },
  heave: {
    past: {
      en: 'We heaved the heavy box onto the truck yesterday.',
      vi: 'Hôm qua chúng tôi đã nâng thùng nặng lên xe tải.',
    },
    perfect: {
      en: 'They have heaved the boat out of the water.',
      vi: 'Họ đã kéo con thuyền lên khỏi mặt nước.',
    },
    mistakeVi: 'heaved là dạng thường dùng; hove chỉ gặp trong ngành hàng hải.',
  },
  hew: {
    past: {
      en: 'My grandfather hewed wood for the fire when I was a child.',
      vi: 'Hồi tôi còn nhỏ, ông đã đẽo củi để nhóm lửa.',
    },
    perfect: {
      en: 'The workers have hewn the stone into blocks.',
      vi: 'Công nhân đã đẽo đá thành từng khối.',
    },
    mistakeVi: 'V2 là hewed, V3 là hewn hoặc hewed; không viết hewen.',
  },
  hide: {
    past: {
      en: 'The kids hid behind the door this morning.',
      vi: 'Sáng nay bọn trẻ đã trốn sau cánh cửa.',
    },
    perfect: { en: 'Where have you hidden my keys?', vi: 'Bạn đã giấu chìa khoá của tôi ở đâu?' },
    mistakeVi: 'hid là V2, hidden là V3 (gấp đôi d); không viết hided.',
  },
  hit: {
    past: { en: 'A car hit the wall last night.', vi: 'Tối qua một chiếc ô tô đã đâm vào tường.' },
    perfect: { en: 'Has the storm hit the coast yet?', vi: 'Cơn bão đã đổ bộ vào bờ biển chưa?' },
    mistakeVi: 'Không viết hitted: hit giữ nguyên ở cả ba dạng.',
  },
  hold: {
    past: {
      en: 'We held a party for Mum last Saturday.',
      vi: 'Thứ Bảy tuần trước chúng tôi đã tổ chức tiệc cho mẹ.',
    },
    perfect: {
      en: 'They have held the meeting in this room before.',
      vi: 'Họ đã từng tổ chức cuộc họp trong phòng này.',
    },
    mistakeVi: 'Không viết holded: hold – held – held.',
  },
  hurt: {
    past: {
      en: 'I hurt my back at the gym yesterday.',
      vi: 'Hôm qua tôi đã bị đau lưng ở phòng tập.',
    },
    perfect: { en: 'Have you hurt yourself?', vi: 'Bạn có bị đau ở đâu không?' },
    mistakeVi: 'Không viết hurted: hurt giữ nguyên ở cả ba dạng.',
  },
  inlay: {
    past: {
      en: 'The craftsman inlaid the box with pearl last month.',
      vi: 'Tháng trước người thợ đã khảm xà cừ lên chiếc hộp.',
    },
    perfect: {
      en: 'He has inlaid gold into the wooden table.',
      vi: 'Ông ấy đã cẩn vàng vào chiếc bàn gỗ.',
    },
    mistakeVi: 'inlay – inlaid – inlaid (giống lay – laid); không viết inlayed.',
  },
  input: {
    past: { en: 'I input all the data yesterday.', vi: 'Hôm qua tôi đã nhập hết dữ liệu.' },
    perfect: {
      en: 'Have you input your password correctly?',
      vi: 'Bạn đã nhập đúng mật khẩu chưa?',
    },
    mistakeVi: 'input giữ nguyên ở cả ba dạng; inputted cũng được chấp nhận.',
  },
  inset: {
    past: {
      en: 'The jeweller inset a small stone into the ring last week.',
      vi: 'Tuần trước người thợ kim hoàn đã gắn một viên đá nhỏ vào nhẫn.',
    },
    perfect: {
      en: 'They have inset a photo into the page.',
      vi: 'Họ đã ghép một bức ảnh vào trang.',
    },
    mistakeVi: 'inset giữ nguyên ở cả ba dạng, không viết insetted.',
  },
  keep: {
    past: { en: 'She kept the door open last night.', vi: 'Tối qua cô ấy đã để cửa mở.' },
    perfect: { en: 'I have kept your seat for you.', vi: 'Tôi đã giữ chỗ cho bạn rồi.' },
    mistakeVi: 'Không viết keeped: keep – kept – kept (giống sleep – slept).',
  },
  kneel: {
    past: {
      en: 'Everyone knelt down to pray last Sunday.',
      vi: 'Chủ nhật tuần trước mọi người đã quỳ xuống cầu nguyện.',
    },
    perfect: {
      en: 'She has knelt in the garden all morning.',
      vi: 'Cô ấy đã quỳ trong vườn cả buổi sáng.',
    },
    mistakeVi: 'knelt (Anh) và kneeled (Mỹ) đều đúng; chữ k đầu không đọc.',
  },
  knit: {
    past: {
      en: 'My grandma knit me a scarf last winter.',
      vi: 'Mùa đông năm ngoái bà đã đan cho tôi một chiếc khăn.',
    },
    perfect: {
      en: 'She has knit three hats this month.',
      vi: 'Tháng này bà ấy đã đan ba chiếc mũ.',
    },
    mistakeVi: 'knit (Mỹ) và knitted (Anh) đều đúng; chữ k đầu không đọc.',
  },
  know: {
    past: { en: 'I knew the answer yesterday.', vi: 'Hôm qua tôi đã biết câu trả lời.' },
    perfect: { en: 'I have known him since 2015.', vi: 'Tôi quen anh ấy từ năm 2015.' },
    mistakeVi: 'knew đọc /njuː/ như new; known đọc /nəʊn/; không viết knowed.',
  },
  lay: {
    past: {
      en: 'The hen laid two eggs this morning.',
      vi: 'Sáng nay con gà mái đã đẻ hai quả trứng.',
    },
    perfect: { en: 'She has laid the table for dinner.', vi: 'Cô ấy đã dọn bàn ăn tối.' },
    mistakeVi: 'lay (đặt) có vật đi sau; đừng nhầm với lay là V2 của lie (nằm).',
  },
  lead: {
    past: {
      en: 'Our guide led us to the old temple yesterday.',
      vi: 'Hôm qua hướng dẫn viên đã dẫn chúng tôi đến ngôi chùa cổ.',
    },
    perfect: {
      en: 'She has led the team for five years.',
      vi: 'Cô ấy đã dẫn dắt đội được năm năm.',
    },
    mistakeVi: 'led đọc /led/; không viết leaded (lead /led/ là chì).',
  },
  lean: {
    past: { en: 'He leant against the wall last night.', vi: 'Tối qua anh ấy đã dựa vào tường.' },
    perfect: {
      en: 'She has leant her bike against the tree.',
      vi: 'Cô ấy đã dựng xe đạp tựa vào gốc cây.',
    },
    mistakeVi: 'leant đọc /lent/; leaned cũng đúng ở V2 và V3.',
  },
  leap: {
    past: {
      en: 'The cat leapt onto the table this morning.',
      vi: 'Sáng nay con mèo đã nhảy phốc lên bàn.',
    },
    perfect: { en: 'Prices have leapt this year.', vi: 'Năm nay giá cả đã tăng vọt.' },
    mistakeVi: 'leapt đọc /lept/ (e ngắn); leaped cũng đúng.',
  },
  learn: {
    past: { en: 'I learnt to swim when I was a child.', vi: 'Tôi đã học bơi từ hồi còn nhỏ.' },
    perfect: {
      en: 'She has learnt a lot of new words.',
      vi: 'Cô ấy đã học được rất nhiều từ mới.',
    },
    mistakeVi: 'learnt hay dùng ở Anh, learned ở Mỹ; cả hai đúng ở V2 và V3.',
  },
  leave: {
    past: {
      en: 'We left home at seven this morning.',
      vi: 'Sáng nay chúng tôi đã rời nhà lúc bảy giờ.',
    },
    perfect: { en: 'Has the train left yet?', vi: 'Tàu đã chạy chưa?' },
    mistakeVi: 'left (V2/V3 của leave) trùng chữ left (bên trái).',
  },
  lend: {
    past: {
      en: 'My sister lent me her dress last weekend.',
      vi: 'Cuối tuần trước chị tôi đã cho tôi mượn váy.',
    },
    perfect: { en: 'I have lent him my notes.', vi: 'Tôi đã cho anh ấy mượn vở ghi.' },
    mistakeVi: 'lend (cho mượn) – lent; borrow (mượn) là động từ có quy tắc.',
  },
  let: {
    past: {
      en: 'Dad let us stay up late last night.',
      vi: 'Tối qua bố đã cho chúng tôi thức khuya.',
    },
    perfect: {
      en: 'She has never let me drive her car.',
      vi: 'Cô ấy chưa bao giờ cho tôi lái xe của cô ấy.',
    },
    mistakeVi: 'Không viết letted: let giữ nguyên ở cả ba dạng.',
  },
  lie: {
    past: {
      en: 'The cat lay in the sun all afternoon yesterday.',
      vi: 'Hôm qua con mèo đã nằm phơi nắng cả buổi chiều.',
    },
    perfect: { en: 'He has lain in bed all day.', vi: 'Anh ấy đã nằm trên giường cả ngày.' },
    mistakeVi: 'lie – lay – lain không có vật đi sau; lie (nói dối) là lied.',
  },
  light: {
    past: {
      en: 'We lit a fire on the beach last night.',
      vi: 'Tối qua chúng tôi đã đốt lửa trên bãi biển.',
    },
    perfect: { en: 'Have you lit the candles yet?', vi: 'Bạn đã thắp nến chưa?' },
    mistakeVi: 'lit hay dùng hơn lighted; đừng nhầm với light (nhẹ, ánh sáng).',
  },
  lose: {
    past: {
      en: 'I lost my wallet on the bus yesterday.',
      vi: 'Hôm qua tôi đã đánh mất ví trên xe buýt.',
    },
    perfect: { en: 'Our team has lost three games.', vi: 'Đội chúng tôi đã thua ba trận.' },
    mistakeVi: 'lost (của lose) khác loose (lỏng); lose đọc /luːz/.',
  },
  make: {
    past: {
      en: 'Mum made a chocolate cake on Sunday.',
      vi: 'Hôm Chủ nhật mẹ đã làm bánh sô-cô-la.',
    },
    perfect: { en: 'Have you made your bed?', vi: 'Bạn đã dọn giường chưa?' },
    mistakeVi: 'Không viết maked: make – made – made.',
  },
  mean: {
    past: { en: 'I meant to call you last night.', vi: 'Tối qua tôi đã định gọi cho bạn.' },
    perfect: { en: 'I have always meant to learn French.', vi: 'Tôi luôn định học tiếng Pháp.' },
    mistakeVi: 'meant đọc /ment/ (e ngắn), không đọc như mean.',
  },
  meet: {
    past: {
      en: 'We met at a café last Friday.',
      vi: 'Thứ Sáu tuần trước chúng tôi đã gặp nhau ở quán cà phê.',
    },
    perfect: { en: 'Have you met my brother?', vi: 'Bạn đã gặp anh trai tôi chưa?' },
    mistakeVi: 'met là V2/V3 của meet; không viết meeted.',
  },
  mislay: {
    past: { en: 'I mislaid my glasses this morning.', vi: 'Sáng nay tôi đã để lạc mất kính.' },
    perfect: {
      en: 'She has mislaid her car keys again.',
      vi: 'Cô ấy lại để lạc chìa khoá xe rồi.',
    },
    mistakeVi: 'mislay – mislaid – mislaid (như lay – laid), không viết mislayed.',
  },
  mislead: {
    past: { en: 'The map misled us yesterday.', vi: 'Hôm qua tấm bản đồ đã làm chúng tôi đi lạc.' },
    perfect: {
      en: 'The advert has misled many customers.',
      vi: 'Quảng cáo đó đã đánh lừa nhiều khách hàng.',
    },
    mistakeVi: 'misled đọc /mɪsˈled/; không viết misleaded.',
  },
  misread: {
    past: {
      en: 'I misread the bus timetable this morning.',
      vi: 'Sáng nay tôi đã đọc nhầm lịch xe buýt.',
    },
    perfect: { en: 'You have misread my message.', vi: 'Bạn đã hiểu sai tin nhắn của tôi.' },
    mistakeVi: 'misread V2/V3 viết giống V1 nhưng đọc là /mɪsˈred/.',
  },
  misspell: {
    past: {
      en: 'He misspelt my name in the email yesterday.',
      vi: 'Hôm qua anh ấy đã viết sai tên tôi trong email.',
    },
    perfect: {
      en: 'You have misspelt this word again.',
      vi: 'Bạn lại viết sai chính tả từ này rồi.',
    },
    mistakeVi: 'misspell có hai chữ s; V2/V3 là misspelt hoặc misspelled.',
  },
  mistake: {
    past: {
      en: 'I mistook her for her sister last night.',
      vi: 'Tối qua tôi đã nhầm cô ấy với em gái cô ấy.',
    },
    perfect: {
      en: 'You have mistaken me for someone else.',
      vi: 'Bạn đã nhầm tôi với người khác.',
    },
    mistakeVi: 'mistake A for B: mistook là V2, mistaken là V3.',
  },
  misunderstand: {
    past: {
      en: 'She misunderstood my question yesterday.',
      vi: 'Hôm qua cô ấy đã hiểu lầm câu hỏi của tôi.',
    },
    perfect: { en: 'I think you have misunderstood me.', vi: 'Tôi nghĩ bạn đã hiểu lầm tôi.' },
    mistakeVi: 'Không viết misunderstanded: giống understand – understood.',
  },
  mow: {
    past: { en: 'My dad mowed the lawn on Sunday.', vi: 'Hôm Chủ nhật bố tôi đã cắt cỏ.' },
    perfect: { en: 'Have you mown the grass yet?', vi: 'Bạn đã cắt cỏ chưa?' },
    mistakeVi: 'V2 chỉ là mowed; V3 là mown hoặc mowed.',
  },
  outbid: {
    past: {
      en: 'A rich man outbid me at the auction yesterday.',
      vi: 'Hôm qua một người giàu đã trả giá cao hơn tôi ở buổi đấu giá.',
    },
    perfect: {
      en: 'Someone has outbid you on this phone.',
      vi: 'Ai đó đã trả giá cao hơn bạn cho chiếc điện thoại này.',
    },
    mistakeVi: 'outbid giữ nguyên ở cả ba dạng, giống bid.',
  },
  outdo: {
    past: {
      en: 'She outdid everyone in the test last week.',
      vi: 'Tuần trước cô ấy đã làm bài kiểm tra giỏi hơn mọi người.',
    },
    perfect: {
      en: 'You have really outdone yourself this time.',
      vi: 'Lần này bạn đã làm tốt hơn hẳn mọi khi.',
    },
    mistakeVi: 'outdid là V2, outdone là V3 (giống do – did – done).',
  },
  outgrow: {
    past: {
      en: 'My son outgrew his shoes last month.',
      vi: 'Tháng trước con trai tôi đã lớn quá cỡ đôi giày.',
    },
    perfect: {
      en: 'She has outgrown all her old clothes.',
      vi: 'Cô bé đã lớn quá khổ hết quần áo cũ.',
    },
    mistakeVi: 'outgrew là V2, outgrown là V3 (giống grow – grew – grown).',
  },
  output: {
    past: {
      en: 'The machine output a long report yesterday.',
      vi: 'Hôm qua chiếc máy đã xuất ra một bản báo cáo dài.',
    },
    perfect: {
      en: 'The program has output the results to a file.',
      vi: 'Chương trình đã xuất kết quả ra một tệp.',
    },
    mistakeVi: 'output giữ nguyên ở cả ba dạng; outputted cũng được.',
  },
  outrun: {
    past: {
      en: 'The thief outran the police last night.',
      vi: 'Tối qua tên trộm đã chạy nhanh hơn cảnh sát.',
    },
    perfect: {
      en: 'He has never outrun his older brother.',
      vi: 'Cậu ấy chưa bao giờ chạy nhanh hơn anh trai.',
    },
    mistakeVi: 'outran là V2; V3 là outrun (giống V1), như run – ran – run.',
  },
  outsell: {
    past: {
      en: 'Her book outsold mine last year.',
      vi: 'Năm ngoái sách của cô ấy đã bán chạy hơn sách của tôi.',
    },
    perfect: {
      en: 'This phone has outsold all the others.',
      vi: 'Chiếc điện thoại này đã bán chạy hơn tất cả các loại khác.',
    },
    mistakeVi: 'outsell – outsold – outsold (giống sell – sold – sold).',
  },
  overcome: {
    past: {
      en: 'She overcame her fear of water last summer.',
      vi: 'Mùa hè năm ngoái cô ấy đã vượt qua nỗi sợ nước.',
    },
    perfect: { en: 'He has overcome many problems.', vi: 'Anh ấy đã vượt qua nhiều khó khăn.' },
    mistakeVi: 'overcame là V2; V3 overcome giống V1 (như come).',
  },
  overdo: {
    past: { en: 'I overdid the exercise yesterday.', vi: 'Hôm qua tôi đã tập thể dục quá sức.' },
    perfect: {
      en: 'You have overdone the salt in this soup.',
      vi: 'Bạn đã cho quá nhiều muối vào món canh này.',
    },
    mistakeVi: 'overdid là V2, overdone là V3; overdone còn nghĩa là nấu quá chín.',
  },
  overeat: {
    past: {
      en: 'We overate at the party last night.',
      vi: 'Tối qua chúng tôi đã ăn quá nhiều ở bữa tiệc.',
    },
    perfect: { en: 'I have overeaten again.', vi: 'Tôi lại ăn quá no rồi.' },
    mistakeVi: 'overate là V2, overeaten là V3 (giống eat – ate – eaten).',
  },
  overfly: {
    past: {
      en: 'A plane overflew our village this morning.',
      vi: 'Sáng nay một chiếc máy bay đã bay qua làng tôi.',
    },
    perfect: {
      en: 'Planes have overflown this area all day.',
      vi: 'Máy bay đã bay qua khu vực này cả ngày.',
    },
    mistakeVi: 'overflew là V2, overflown là V3; đừng nhầm với overflow (tràn).',
  },
  overhang: {
    past: {
      en: 'A big tree overhung the road when I was a child.',
      vi: 'Hồi tôi còn nhỏ, một cái cây lớn đã chìa ra phía trên con đường.',
    },
    perfect: {
      en: 'The roof has overhung the path for years.',
      vi: 'Mái nhà đã nhô ra phía trên lối đi nhiều năm nay.',
    },
    mistakeVi: 'overhang – overhung – overhung (giống hang – hung).',
  },
  overhear: {
    past: {
      en: 'I overheard their conversation yesterday.',
      vi: 'Hôm qua tôi đã nghe lỏm cuộc trò chuyện của họ.',
    },
    perfect: {
      en: 'Have you ever overheard a secret?',
      vi: 'Bạn đã bao giờ nghe lỏm được bí mật chưa?',
    },
    mistakeVi: 'overheard đọc giống heard /hɜːd/; không viết overheared.',
  },
  overlay: {
    past: {
      en: 'She overlaid the cake with cream last night.',
      vi: 'Tối qua cô ấy đã phủ kem lên bánh.',
    },
    perfect: {
      en: 'They have overlaid the floor with new tiles.',
      vi: 'Họ đã lát gạch mới phủ lên sàn.',
    },
    mistakeVi: 'overlay – overlaid – overlaid; không viết overlayed.',
  },
  overpay: {
    past: {
      en: 'I overpaid for this jacket last week.',
      vi: 'Tuần trước tôi đã trả quá nhiều tiền cho chiếc áo này.',
    },
    perfect: { en: 'You have overpaid by ten dollars.', vi: 'Bạn đã trả thừa mười đô la.' },
    mistakeVi: 'Không viết overpayed: giống pay – paid – paid.',
  },
  overrun: {
    past: {
      en: 'Tourists overran the small town last summer.',
      vi: 'Mùa hè năm ngoái khách du lịch đã tràn ngập thị trấn nhỏ.',
    },
    perfect: { en: 'Mice have overrun the old house.', vi: 'Chuột đã tràn ngập ngôi nhà cũ.' },
    mistakeVi: 'overran là V2; V3 overrun giống V1 (như run – ran – run).',
  },
  oversee: {
    past: {
      en: 'Mr Lam oversaw the project last year.',
      vi: 'Năm ngoái ông Lâm đã giám sát dự án.',
    },
    perfect: {
      en: 'She has overseen the work from the start.',
      vi: 'Bà ấy đã giám sát công việc từ đầu.',
    },
    mistakeVi: 'oversaw là V2, overseen là V3 (giống see – saw – seen).',
  },
  overshoot: {
    past: {
      en: 'The plane overshot the runway yesterday.',
      vi: 'Hôm qua chiếc máy bay đã chạy quá đường băng.',
    },
    perfect: {
      en: 'We have overshot our budget this month.',
      vi: 'Tháng này chúng tôi đã chi vượt ngân sách.',
    },
    mistakeVi: 'Không viết overshooted: giống shoot – shot – shot.',
  },
  oversleep: {
    past: {
      en: 'I overslept and missed the bus this morning.',
      vi: 'Sáng nay tôi đã ngủ quên và lỡ xe buýt.',
    },
    perfect: { en: 'He has overslept again.', vi: 'Anh ấy lại ngủ quên rồi.' },
    mistakeVi: 'Không viết oversleeped: giống sleep – slept – slept.',
  },
  overtake: {
    past: {
      en: 'A red car overtook us on the highway yesterday.',
      vi: 'Hôm qua một chiếc xe đỏ đã vượt chúng tôi trên cao tốc.',
    },
    perfect: {
      en: 'Our team has overtaken them in the table.',
      vi: 'Đội chúng tôi đã vượt họ trên bảng xếp hạng.',
    },
    mistakeVi: 'overtook là V2, overtaken là V3 (giống take – took – taken).',
  },
  overthrow: {
    past: {
      en: 'The people overthrew the king in 1789.',
      vi: 'Năm 1789 người dân đã lật đổ nhà vua.',
    },
    perfect: { en: 'The army has overthrown the government.', vi: 'Quân đội đã lật đổ chính phủ.' },
    mistakeVi: 'overthrew là V2, overthrown là V3 (giống throw – threw – thrown).',
  },
  pay: {
    past: { en: 'I paid the electricity bill yesterday.', vi: 'Hôm qua tôi đã trả tiền điện.' },
    perfect: { en: 'Have you paid the rent yet?', vi: 'Bạn đã trả tiền nhà chưa?' },
    mistakeVi: 'Không viết payed: pay – paid – paid (y đổi thành i).',
  },
  prove: {
    past: {
      en: 'She proved her point at the meeting yesterday.',
      vi: 'Hôm qua cô ấy đã chứng minh quan điểm của mình trong cuộc họp.',
    },
    perfect: {
      en: 'Scientists have proven the theory wrong.',
      vi: 'Các nhà khoa học đã chứng minh giả thuyết đó sai.',
    },
    mistakeVi: 'V2 chỉ là proved; V3 là proven hoặc proved.',
  },
  put: {
    past: {
      en: 'I put the milk in the fridge this morning.',
      vi: 'Sáng nay tôi đã để sữa vào tủ lạnh.',
    },
    perfect: { en: 'Where have you put my glasses?', vi: 'Bạn đã để kính của tôi ở đâu?' },
    mistakeVi: 'Không viết putted: put giữ nguyên ở cả ba dạng.',
  },
  quit: {
    past: { en: 'He quit smoking two years ago.', vi: 'Anh ấy đã bỏ thuốc lá hai năm trước.' },
    perfect: { en: 'I have quit my piano lessons.', vi: 'Tôi đã thôi học đàn piano.' },
    mistakeVi: 'quit giữ nguyên ở cả ba dạng; quitted rất ít dùng.',
  },
  read: {
    past: { en: 'I read a good book last week.', vi: 'Tuần trước tôi đã đọc một cuốn sách hay.' },
    perfect: { en: 'Have you read the newspaper today?', vi: 'Hôm nay bạn đã đọc báo chưa?' },
    mistakeVi: 'read V2/V3 viết giống V1 nhưng đọc là /red/.',
  },
  rebuild: {
    past: {
      en: 'They rebuilt the bridge after the flood last year.',
      vi: 'Năm ngoái họ đã xây lại cây cầu sau trận lũ.',
    },
    perfect: { en: 'The town has rebuilt its old market.', vi: 'Thị trấn đã xây lại khu chợ cũ.' },
    mistakeVi: 'Không viết rebuilded: giống build – built – built.',
  },
  redo: {
    past: { en: 'I redid my homework last night.', vi: 'Tối qua tôi đã làm lại bài tập về nhà.' },
    perfect: { en: 'Have you redone the test?', vi: 'Bạn đã làm lại bài kiểm tra chưa?' },
    mistakeVi: 'redid là V2, redone là V3 (giống do – did – done).',
  },
  remake: {
    past: { en: 'They remade the old film in 2019.', vi: 'Năm 2019 họ đã làm lại bộ phim cũ.' },
    perfect: { en: 'She has remade the bed.', vi: 'Cô ấy đã dọn lại giường.' },
    mistakeVi: 'Không viết remaked: giống make – made – made.',
  },
  rend: {
    past: {
      en: 'The storm rent the sail last night.',
      vi: 'Tối qua cơn bão đã xé toạc cánh buồm.',
    },
    perfect: {
      en: 'Her cry has rent the silence.',
      vi: 'Tiếng kêu của cô ấy đã xé tan sự im lặng.',
    },
    mistakeVi: 'rent (V2/V3 của rend) trùng chữ rent (thuê); rend là từ văn chương.',
  },
  repay: {
    past: {
      en: 'I repaid my brother last week.',
      vi: 'Tuần trước tôi đã trả lại tiền cho anh trai.',
    },
    perfect: { en: 'Have you repaid the loan yet?', vi: 'Bạn đã trả khoản vay chưa?' },
    mistakeVi: 'Không viết repayed: giống pay – paid – paid.',
  },
  resell: {
    past: {
      en: 'He resold the concert tickets yesterday.',
      vi: 'Hôm qua anh ấy đã bán lại vé buổi hoà nhạc.',
    },
    perfect: {
      en: 'She has resold her old phone online.',
      vi: 'Cô ấy đã bán lại điện thoại cũ trên mạng.',
    },
    mistakeVi: 'resell – resold – resold (giống sell – sold – sold).',
  },
  retell: {
    past: {
      en: 'Grandma retold the old story last night.',
      vi: 'Tối qua bà đã kể lại câu chuyện xưa.',
    },
    perfect: {
      en: 'He has retold that joke many times.',
      vi: 'Anh ấy đã kể lại câu chuyện cười đó nhiều lần.',
    },
    mistakeVi: 'Không viết retelled: giống tell – told – told.',
  },
  rewrite: {
    past: { en: 'I rewrote my essay last night.', vi: 'Tối qua tôi đã viết lại bài luận.' },
    perfect: { en: 'Have you rewritten the first page?', vi: 'Bạn đã viết lại trang đầu chưa?' },
    mistakeVi: 'rewrote là V2, rewritten là V3 (gấp đôi t).',
  },
  rid: {
    past: {
      en: 'We rid the kitchen of ants last month.',
      vi: 'Tháng trước chúng tôi đã diệt sạch kiến trong bếp.',
    },
    perfect: {
      en: 'She has finally rid herself of that bad habit.',
      vi: 'Cuối cùng cô ấy đã bỏ được thói quen xấu đó.',
    },
    mistakeVi: 'rid giữ nguyên ở cả ba dạng; hay gặp nhất trong get rid of.',
  },
  ride: {
    past: { en: 'I rode my bike to school yesterday.', vi: 'Hôm qua tôi đã đạp xe đến trường.' },
    perfect: { en: 'Have you ever ridden a horse?', vi: 'Bạn đã bao giờ cưỡi ngựa chưa?' },
    mistakeVi: 'rode là V2, ridden là V3 (gấp đôi d); không viết rided.',
  },
  ring: {
    past: { en: 'The phone rang twice last night.', vi: 'Tối qua điện thoại đã reo hai lần.' },
    perfect: { en: 'Has the bell rung yet?', vi: 'Chuông đã reo chưa?' },
    mistakeVi: 'rang là V2, rung là V3; không nói “I have rang”.',
  },
  rise: {
    past: { en: 'The sun rose at six this morning.', vi: 'Sáng nay mặt trời đã mọc lúc sáu giờ.' },
    perfect: {
      en: 'The river has risen after the heavy rain.',
      vi: 'Nước sông đã dâng lên sau trận mưa lớn.',
    },
    mistakeVi: 'rise không có vật đi sau; raise (nâng lên) là động từ có quy tắc.',
  },
  run: {
    past: { en: 'He ran to the bus stop yesterday.', vi: 'Hôm qua anh ấy đã chạy ra bến xe buýt.' },
    perfect: { en: 'She has run three marathons.', vi: 'Cô ấy đã chạy ba cuộc marathon.' },
    mistakeVi: 'ran là V2; V3 là run (giống V1): I have run.',
  },
  saw: {
    past: {
      en: 'He sawed the old table in half yesterday.',
      vi: 'Hôm qua anh ấy đã cưa đôi chiếc bàn cũ.',
    },
    perfect: {
      en: 'They have sawn the logs into pieces.',
      vi: 'Họ đã cưa các khúc gỗ thành từng đoạn.',
    },
    mistakeVi: 'saw (cưa) có V2 sawed; đừng nhầm với saw là V2 của see.',
  },
  say: {
    past: {
      en: 'He said goodbye at the door last night.',
      vi: 'Tối qua anh ấy đã chào tạm biệt ở cửa.',
    },
    perfect: { en: 'Have you said sorry to her?', vi: 'Bạn đã xin lỗi cô ấy chưa?' },
    mistakeVi: 'said đọc /sed/, không đọc /seɪd/; không viết sayed.',
  },
  see: {
    past: { en: 'I saw a rainbow yesterday.', vi: 'Hôm qua tôi đã nhìn thấy cầu vồng.' },
    perfect: { en: 'I have never seen snow.', vi: 'Tôi chưa bao giờ nhìn thấy tuyết.' },
    mistakeVi: 'Không nói “I seen”: saw là V2, seen phải đi sau have.',
  },
  seek: {
    past: {
      en: 'They sought help from a doctor last month.',
      vi: 'Tháng trước họ đã tìm đến bác sĩ để được giúp đỡ.',
    },
    perfect: {
      en: 'She has sought a new job for months.',
      vi: 'Cô ấy đã tìm việc mới nhiều tháng nay.',
    },
    mistakeVi: 'Không viết seeked: seek – sought – sought, đọc /sɔːt/.',
  },
  sell: {
    past: { en: 'He sold his bike last month.', vi: 'Tháng trước anh ấy đã bán xe đạp.' },
    perfect: { en: 'We have sold all the tickets.', vi: 'Chúng tôi đã bán hết vé.' },
    mistakeVi: 'sold là V2/V3 của sell; không viết selled.',
  },
  send: {
    past: {
      en: 'She sent me a postcard last summer.',
      vi: 'Mùa hè năm ngoái cô ấy đã gửi cho tôi một tấm bưu thiếp.',
    },
    perfect: { en: 'Have you sent the letter yet?', vi: 'Bạn đã gửi thư chưa?' },
    mistakeVi: 'Không viết sended: send – sent – sent (đổi d thành t).',
  },
  set: {
    past: {
      en: 'We set off at dawn yesterday.',
      vi: 'Hôm qua chúng tôi đã khởi hành lúc bình minh.',
    },
    perfect: { en: 'The sun has already set.', vi: 'Mặt trời đã lặn rồi.' },
    mistakeVi: 'set giữ nguyên ở cả ba dạng; đừng nhầm với sit – sat – sat.',
  },
  sew: {
    past: {
      en: 'My mum sewed a button on my shirt last night.',
      vi: 'Tối qua mẹ đã khâu cúc áo cho tôi.',
    },
    perfect: { en: 'She has sewn her own dress.', vi: 'Cô ấy đã tự may váy cho mình.' },
    mistakeVi: 'sew đọc /səʊ/ như so; V3 là sewn hoặc sewed.',
  },
  shake: {
    past: { en: 'We shook hands last week.', vi: 'Tuần trước chúng tôi đã bắt tay nhau.' },
    perfect: { en: 'The news has shaken everyone.', vi: 'Tin tức đó đã làm mọi người bàng hoàng.' },
    mistakeVi: 'shook là V2, shaken là V3; không viết shaked.',
  },
  shed: {
    past: {
      en: 'The dog shed hair all over the sofa last week.',
      vi: 'Tuần trước con chó đã rụng lông khắp ghế sofa.',
    },
    perfect: {
      en: 'Our cat has shed its winter coat.',
      vi: 'Con mèo nhà tôi đã rụng hết lớp lông mùa đông.',
    },
    mistakeVi: 'shed giữ nguyên ở cả ba dạng; shed còn là danh từ (nhà kho).',
  },
  shine: {
    past: { en: 'The sun shone all day yesterday.', vi: 'Hôm qua mặt trời đã chiếu sáng cả ngày.' },
    perfect: {
      en: 'The sun has shone every day this week.',
      vi: 'Tuần này ngày nào trời cũng nắng.',
    },
    mistakeVi: 'shine (chiếu sáng) – shone; nghĩa đánh bóng (giày) là shined.',
  },
  shoot: {
    past: {
      en: 'He shot the ball into the net last night.',
      vi: 'Tối qua anh ấy đã sút bóng vào lưới.',
    },
    perfect: { en: 'She has shot a short film.', vi: 'Cô ấy đã quay một bộ phim ngắn.' },
    mistakeVi: 'shot (V2/V3) đọc /ʃɒt/; không viết shooted.',
  },
  show: {
    past: { en: 'She showed me her photos yesterday.', vi: 'Hôm qua cô ấy đã cho tôi xem ảnh.' },
    perfect: { en: 'Have you shown him the house?', vi: 'Bạn đã dẫn anh ấy đi xem nhà chưa?' },
    mistakeVi: 'V2 chỉ là showed; V3 là shown hoặc showed.',
  },
  shrink: {
    past: {
      en: 'My T-shirt shrank in the wash last week.',
      vi: 'Tuần trước áo phông của tôi đã bị co lại khi giặt.',
    },
    perfect: { en: 'The lake has shrunk this summer.', vi: 'Mùa hè này hồ đã cạn đi nhiều.' },
    mistakeVi: 'shrank là V2, shrunk là V3 (giống drink – drank – drunk).',
  },
  shut: {
    past: { en: 'She shut the window last night.', vi: 'Tối qua cô ấy đã đóng cửa sổ.' },
    perfect: { en: 'The shop has already shut.', vi: 'Cửa hàng đã đóng cửa rồi.' },
    mistakeVi: 'Không viết shutted: shut giữ nguyên ở cả ba dạng.',
  },
  sing: {
    past: {
      en: 'We sang happy birthday last night.',
      vi: 'Tối qua chúng tôi đã hát chúc mừng sinh nhật.',
    },
    perfect: {
      en: 'She has sung in that choir for years.',
      vi: 'Cô ấy đã hát trong dàn hợp xướng đó nhiều năm.',
    },
    mistakeVi: 'sang là V2, sung là V3; không nói “I have sang”.',
  },
  sink: {
    past: { en: 'The old boat sank in 1912.', vi: 'Con thuyền cũ đã chìm năm 1912.' },
    perfect: {
      en: 'The ship has sunk to the bottom of the sea.',
      vi: 'Con tàu đã chìm xuống đáy biển.',
    },
    mistakeVi: 'sank là V2, sunk là V3; i → a → u như drink.',
  },
  sit: {
    past: { en: 'We sat in the garden last night.', vi: 'Tối qua chúng tôi đã ngồi ngoài vườn.' },
    perfect: { en: 'She has sat there all morning.', vi: 'Cô ấy đã ngồi đó cả buổi sáng.' },
    mistakeVi: 'sat là V2/V3 của sit; đừng nhầm với set (đặt).',
  },
  slay: {
    past: {
      en: 'The hero slew the dragon long ago.',
      vi: 'Ngày xưa người anh hùng đã giết con rồng.',
    },
    perfect: { en: 'The knight has slain the monster.', vi: 'Chàng hiệp sĩ đã giết con quái vật.' },
    mistakeVi: 'slew là V2, slain là V3; từ văn chương, đời thường dùng kill.',
  },
  sleep: {
    past: { en: 'I slept for ten hours last night.', vi: 'Tối qua tôi đã ngủ mười tiếng.' },
    perfect: { en: 'Have you slept well?', vi: 'Bạn ngủ có ngon không?' },
    mistakeVi: 'Không viết sleeped: sleep – slept – slept.',
  },
  slide: {
    past: { en: 'The boy slid down the hill yesterday.', vi: 'Hôm qua cậu bé đã trượt xuống đồi.' },
    perfect: { en: 'The book has slid off the table.', vi: 'Cuốn sách đã trượt khỏi bàn.' },
    mistakeVi: 'Không viết slided: slide – slid – slid (chỉ một d).',
  },
  sling: {
    past: {
      en: 'He slung his bag over his shoulder this morning.',
      vi: 'Sáng nay anh ấy đã quàng túi lên vai.',
    },
    perfect: { en: 'She has slung her coat on the chair.', vi: 'Cô ấy đã quăng áo khoác lên ghế.' },
    mistakeVi: 'Không viết slinged: sling – slung – slung (như cling, swing).',
  },
  slink: {
    past: {
      en: 'The cat slunk out of the room last night.',
      vi: 'Tối qua con mèo đã lẻn ra khỏi phòng.',
    },
    perfect: { en: 'The dog has slunk away again.', vi: 'Con chó lại lẻn đi mất rồi.' },
    mistakeVi: 'Không viết slinked: slink – slunk – slunk.',
  },
  slit: {
    past: {
      en: 'She slit the envelope open this morning.',
      vi: 'Sáng nay cô ấy đã rạch phong bì ra.',
    },
    perfect: { en: 'He has slit the bag with a knife.', vi: 'Anh ấy đã rạch cái túi bằng dao.' },
    mistakeVi: 'slit giữ nguyên ở cả ba dạng, không viết slitted.',
  },
  smell: {
    past: { en: 'The room smelt of paint yesterday.', vi: 'Hôm qua căn phòng có mùi sơn.' },
    perfect: {
      en: 'I have never smelt such a lovely flower.',
      vi: 'Tôi chưa bao giờ ngửi thấy bông hoa nào thơm như vậy.',
    },
    mistakeVi: 'smelt (Anh) và smelled (Mỹ) đều đúng ở V2 và V3.',
  },
  smite: {
    past: {
      en: 'The hero smote the giant long ago.',
      vi: 'Ngày xưa người anh hùng đã đánh gục gã khổng lồ.',
    },
    perfect: {
      en: 'A great storm has smitten the coast.',
      vi: 'Một cơn bão lớn đã giáng xuống bờ biển.',
    },
    mistakeVi: 'smote là V2, smitten là V3; be smitten with còn nghĩa là say mê.',
  },
  sow: {
    past: { en: 'Farmers sowed the rice last month.', vi: 'Tháng trước nông dân đã gieo lúa.' },
    perfect: {
      en: 'We have sown the seeds in the garden.',
      vi: 'Chúng tôi đã gieo hạt trong vườn.',
    },
    mistakeVi: 'sow (gieo) đọc /səʊ/; V2 sowed, V3 sown; đừng nhầm với sew (may).',
  },
  speak: {
    past: { en: 'I spoke to my teacher yesterday.', vi: 'Hôm qua tôi đã nói chuyện với cô giáo.' },
    perfect: { en: 'Have you spoken to him?', vi: 'Bạn đã nói chuyện với anh ấy chưa?' },
    mistakeVi: 'spoke là V2, spoken là V3; không viết speaked.',
  },
  speed: {
    past: {
      en: 'The car sped down the road last night.',
      vi: 'Tối qua chiếc xe đã phóng nhanh trên đường.',
    },
    perfect: { en: 'The time has sped by.', vi: 'Thời gian đã trôi nhanh quá.' },
    mistakeVi: 'sped hay dùng; speeded thường đi với speed up (tăng tốc).',
  },
  spell: {
    past: {
      en: 'She spelt the word correctly yesterday.',
      vi: 'Hôm qua cô ấy đã đánh vần đúng từ đó.',
    },
    perfect: { en: 'Have you spelt my name right?', vi: 'Bạn đã viết đúng tên tôi chưa?' },
    mistakeVi: 'spelt (Anh) và spelled (Mỹ) đều đúng ở V2 và V3.',
  },
  spend: {
    past: {
      en: 'I spent too much money last weekend.',
      vi: 'Cuối tuần trước tôi đã tiêu quá nhiều tiền.',
    },
    perfect: { en: 'She has spent all her savings.', vi: 'Cô ấy đã tiêu hết tiền tiết kiệm.' },
    mistakeVi: 'Không viết spended: spend – spent – spent (đổi d thành t).',
  },
  spill: {
    past: {
      en: 'I spilt coffee on my shirt this morning.',
      vi: 'Sáng nay tôi đã làm đổ cà phê lên áo.',
    },
    perfect: { en: 'Someone has spilt milk on the floor.', vi: 'Ai đó đã làm đổ sữa ra sàn.' },
    mistakeVi: 'spilt (Anh) và spilled (Mỹ) đều đúng ở V2 và V3.',
  },
  spin: {
    past: {
      en: 'The kids spun around in circles yesterday.',
      vi: 'Hôm qua bọn trẻ đã quay vòng vòng.',
    },
    perfect: { en: 'The wheel has spun for hours.', vi: 'Bánh xe đã quay hàng giờ liền.' },
    mistakeVi: 'Không viết spinned: spin – spun – spun.',
  },
  spit: {
    past: { en: 'The baby spat out his food last night.', vi: 'Tối qua em bé đã nhè thức ăn ra.' },
    perfect: { en: 'He has spat on the ground again.', vi: 'Anh ta lại nhổ xuống đất.' },
    mistakeVi: 'spat là dạng chuẩn; spit làm V2, V3 chủ yếu gặp ở Mỹ.',
  },
  split: {
    past: {
      en: 'We split the bill at dinner last night.',
      vi: 'Tối qua chúng tôi đã chia nhau tiền bữa ăn.',
    },
    perfect: { en: 'They have split the money equally.', vi: 'Họ đã chia đều số tiền.' },
    mistakeVi: 'split giữ nguyên ở cả ba dạng, không viết splitted.',
  },
  spoil: {
    past: {
      en: 'The rain spoilt our picnic on Sunday.',
      vi: 'Hôm Chủ nhật cơn mưa đã làm hỏng buổi dã ngoại.',
    },
    perfect: { en: 'Grandma has spoilt the kids again.', vi: 'Bà lại chiều hư bọn trẻ rồi.' },
    mistakeVi: 'spoilt (Anh) và spoiled (Mỹ) đều đúng ở V2 và V3.',
  },
  spread: {
    past: {
      en: 'She spread butter on her toast this morning.',
      vi: 'Sáng nay cô ấy đã phết bơ lên bánh mì.',
    },
    perfect: {
      en: 'The fire has spread to the next house.',
      vi: 'Ngọn lửa đã lan sang nhà bên cạnh.',
    },
    mistakeVi: 'spread giữ nguyên ở cả ba dạng, đọc /spred/.',
  },
  spring: {
    past: {
      en: 'The cat sprang onto the wall yesterday.',
      vi: 'Hôm qua con mèo đã bật nhảy lên tường.',
    },
    perfect: { en: 'A leak has sprung in the roof.', vi: 'Mái nhà đã bị dột.' },
    mistakeVi: 'sprang là V2, sprung là V3; i → a → u như sing.',
  },
  stand: {
    past: {
      en: 'We stood in the rain for an hour yesterday.',
      vi: 'Hôm qua chúng tôi đã đứng dưới mưa một tiếng.',
    },
    perfect: { en: 'He has stood there for ages.', vi: 'Anh ấy đã đứng đó rất lâu.' },
    mistakeVi: 'Không viết standed: stand – stood – stood (như understand).',
  },
  steal: {
    past: {
      en: 'Someone stole my bike last night.',
      vi: 'Tối qua ai đó đã lấy trộm xe đạp của tôi.',
    },
    perfect: { en: 'Thieves have stolen her car.', vi: 'Kẻ trộm đã lấy cắp xe của cô ấy.' },
    mistakeVi: 'stole là V2, stolen là V3; steal (lấy trộm đồ) khác rob (cướp người).',
  },
  stick: {
    past: {
      en: 'I stuck a note on the fridge this morning.',
      vi: 'Sáng nay tôi đã dán một mẩu giấy lên tủ lạnh.',
    },
    perfect: { en: 'The door has stuck again.', vi: 'Cánh cửa lại bị kẹt rồi.' },
    mistakeVi: 'Không viết sticked: stick – stuck – stuck.',
  },
  sting: {
    past: {
      en: 'A bee stung me on the arm yesterday.',
      vi: 'Hôm qua một con ong đã đốt vào tay tôi.',
    },
    perfect: { en: 'Has a jellyfish ever stung you?', vi: 'Bạn đã bao giờ bị sứa châm chưa?' },
    mistakeVi: 'Không viết stinged: sting – stung – stung.',
  },
  stink: {
    past: { en: 'The rubbish stank last week.', vi: 'Tuần trước rác đã bốc mùi hôi.' },
    perfect: { en: 'The kitchen has stunk of fish all day.', vi: 'Nhà bếp bốc mùi cá cả ngày.' },
    mistakeVi: 'V2 là stank (hoặc stunk), V3 chỉ có stunk.',
  },
  strew: {
    past: {
      en: 'The kids strewed toys all over the floor yesterday.',
      vi: 'Hôm qua bọn trẻ đã vứt đồ chơi khắp sàn.',
    },
    perfect: { en: 'The wind has strewn leaves across the yard.', vi: 'Gió đã rải lá khắp sân.' },
    mistakeVi: 'V2 chỉ là strewed; V3 là strewn hoặc strewed.',
  },
  stride: {
    past: {
      en: 'He strode into the room this morning.',
      vi: 'Sáng nay anh ấy đã sải bước vào phòng.',
    },
    perfect: { en: 'She has stridden across the stage.', vi: 'Cô ấy đã sải bước qua sân khấu.' },
    mistakeVi: 'strode là V2; V3 stridden rất hiếm, thường tránh dùng.',
  },
  strike: {
    past: {
      en: 'Lightning struck the old tree last night.',
      vi: 'Tối qua sét đã đánh trúng cây cổ thụ.',
    },
    perfect: {
      en: 'The workers have struck for better pay.',
      vi: 'Công nhân đã đình công đòi tăng lương.',
    },
    mistakeVi: 'struck là V2/V3; stricken chỉ dùng như tính từ (bị ảnh hưởng).',
  },
  string: {
    past: {
      en: 'We strung lights around the tree last Christmas.',
      vi: 'Giáng sinh năm ngoái chúng tôi đã giăng đèn quanh cây.',
    },
    perfect: {
      en: 'She has strung the beads into a necklace.',
      vi: 'Cô ấy đã xâu hạt thành vòng cổ.',
    },
    mistakeVi: 'Không viết stringed: string – strung – strung.',
  },
  strive: {
    past: {
      en: 'She strove to finish on time last month.',
      vi: 'Tháng trước cô ấy đã cố gắng hoàn thành đúng hạn.',
    },
    perfect: {
      en: 'He has always striven to help others.',
      vi: 'Anh ấy luôn cố gắng giúp đỡ người khác.',
    },
    mistakeVi: 'strove là V2, striven là V3; strived cũng gặp trong tiếng Mỹ.',
  },
  swear: {
    past: {
      en: 'He swore to tell the truth yesterday.',
      vi: 'Hôm qua anh ấy đã thề sẽ nói sự thật.',
    },
    perfect: { en: 'I have never sworn at anyone.', vi: 'Tôi chưa bao giờ chửi thề với ai.' },
    mistakeVi: 'swore là V2, sworn là V3 (giống wear – wore – worn).',
  },
  sweep: {
    past: { en: 'I swept the floor this morning.', vi: 'Sáng nay tôi đã quét nhà.' },
    perfect: { en: 'Have you swept the yard?', vi: 'Bạn đã quét sân chưa?' },
    mistakeVi: 'Không viết sweeped: sweep – swept – swept.',
  },
  swell: {
    past: {
      en: 'My ankle swelled after the fall yesterday.',
      vi: 'Hôm qua mắt cá chân tôi đã sưng lên sau cú ngã.',
    },
    perfect: { en: 'Her eyes have swollen from crying.', vi: 'Mắt cô ấy đã sưng vì khóc.' },
    mistakeVi: 'V2 là swelled; V3 swollen (hay dùng) hoặc swelled.',
  },
  swim: {
    past: { en: 'We swam in the lake last summer.', vi: 'Mùa hè năm ngoái chúng tôi đã bơi ở hồ.' },
    perfect: { en: 'I have never swum in the sea.', vi: 'Tôi chưa bao giờ bơi ở biển.' },
    mistakeVi: 'swam là V2, swum là V3; không viết swimmed.',
  },
  swing: {
    past: {
      en: 'The children swung on the gate yesterday.',
      vi: 'Hôm qua bọn trẻ đã đu đưa trên cánh cổng.',
    },
    perfect: { en: 'The door has swung open.', vi: 'Cánh cửa đã bật mở.' },
    mistakeVi: 'Không viết swinged: swing – swung – swung.',
  },
  take: {
    past: {
      en: 'I took a taxi to the airport yesterday.',
      vi: 'Hôm qua tôi đã đi taxi ra sân bay.',
    },
    perfect: { en: 'Have you taken your medicine?', vi: 'Bạn đã uống thuốc chưa?' },
    mistakeVi: 'took là V2, taken là V3; không nói “I have took”.',
  },
  teach: {
    past: {
      en: 'My dad taught me to swim last summer.',
      vi: 'Mùa hè năm ngoái bố đã dạy tôi bơi.',
    },
    perfect: {
      en: 'She has taught English for ten years.',
      vi: 'Cô ấy đã dạy tiếng Anh mười năm.',
    },
    mistakeVi: 'Không viết teached: teach – taught – taught, đọc /tɔːt/.',
  },
  tear: {
    past: {
      en: 'I tore my jeans on a nail yesterday.',
      vi: 'Hôm qua tôi đã làm rách quần bò vì vướng đinh.',
    },
    perfect: {
      en: 'Someone has torn a page out of my book.',
      vi: 'Ai đó đã xé một trang trong sách của tôi.',
    },
    mistakeVi: 'tear (xé) đọc /teə/, khác tear (nước mắt); V2 tore, V3 torn.',
  },
  tell: {
    past: { en: 'She told me the news last night.', vi: 'Tối qua cô ấy đã báo tin cho tôi.' },
    perfect: { en: 'Have you told your parents?', vi: 'Bạn đã nói với bố mẹ chưa?' },
    mistakeVi: 'told là V2/V3; tell luôn có người nghe đi sau (tell me).',
  },
  think: {
    past: { en: 'I thought about you yesterday.', vi: 'Hôm qua tôi đã nghĩ về bạn.' },
    perfect: {
      en: 'Have you thought about my offer?',
      vi: 'Bạn đã suy nghĩ về lời đề nghị của tôi chưa?',
    },
    mistakeVi: 'thought (nghĩ) dễ nhầm với taught (dạy); không viết thinked.',
  },
  throw: {
    past: {
      en: 'He threw the ball to his dog this morning.',
      vi: 'Sáng nay anh ấy đã ném bóng cho con chó.',
    },
    perfect: { en: 'Who has thrown rubbish here?', vi: 'Ai đã vứt rác ở đây?' },
    mistakeVi: 'threw là V2, thrown là V3; không viết throwed.',
  },
  thrust: {
    past: {
      en: 'She thrust the letter into my hand yesterday.',
      vi: 'Hôm qua cô ấy đã dúi lá thư vào tay tôi.',
    },
    perfect: { en: 'He has thrust his hands into his pockets.', vi: 'Anh ấy đã thọc tay vào túi.' },
    mistakeVi: 'thrust giữ nguyên ở cả ba dạng, không viết thrusted.',
  },
  tread: {
    past: { en: 'I trod on a nail yesterday.', vi: 'Hôm qua tôi đã giẫm phải đinh.' },
    perfect: { en: 'Someone has trodden mud into the carpet.', vi: 'Ai đó đã giẫm bùn lên thảm.' },
    mistakeVi: 'tread đọc /tred/; V2 trod, V3 trodden hoặc trod.',
  },
  unbend: {
    past: {
      en: 'The old man unbent his back slowly this morning.',
      vi: 'Sáng nay ông cụ đã từ từ duỗi thẳng lưng.',
    },
    perfect: {
      en: 'She has finally unbent and smiled.',
      vi: 'Cuối cùng cô ấy đã thư giãn và mỉm cười.',
    },
    mistakeVi: 'unbend – unbent – unbent (giống bend – bent – bent).',
  },
  undercut: {
    past: {
      en: 'The new shop undercut our prices last month.',
      vi: 'Tháng trước cửa hàng mới đã bán rẻ hơn chúng tôi.',
    },
    perfect: {
      en: 'They have undercut every rival in town.',
      vi: 'Họ đã bán rẻ hơn mọi đối thủ trong thị trấn.',
    },
    mistakeVi: 'undercut giữ nguyên ở cả ba dạng, giống cut.',
  },
  undergo: {
    past: {
      en: 'My dad underwent an operation last year.',
      vi: 'Năm ngoái bố tôi đã trải qua một ca phẫu thuật.',
    },
    perfect: {
      en: 'The city has undergone big changes.',
      vi: 'Thành phố đã trải qua nhiều thay đổi lớn.',
    },
    mistakeVi: 'underwent là V2, undergone là V3 (giống go – went – gone).',
  },
  underlie: {
    past: {
      en: 'A simple idea underlay the whole plan last year.',
      vi: 'Năm ngoái một ý tưởng đơn giản đã làm nền cho cả kế hoạch.',
    },
    perfect: {
      en: 'Many factors have underlain this crisis.',
      vi: 'Nhiều yếu tố đã làm nền cho cuộc khủng hoảng này.',
    },
    mistakeVi: 'underlie – underlay – underlain (giống lie); đừng nhầm với underline.',
  },
  underpay: {
    past: {
      en: 'The company underpaid its workers last year.',
      vi: 'Năm ngoái công ty đã trả lương thấp cho công nhân.',
    },
    perfect: {
      en: 'They have underpaid me again this month.',
      vi: 'Tháng này họ lại trả lương thấp cho tôi.',
    },
    mistakeVi: 'Không viết underpayed: giống pay – paid – paid.',
  },
  undersell: {
    past: {
      en: 'The supermarket undersold the small shops last year.',
      vi: 'Năm ngoái siêu thị đã bán rẻ hơn các cửa hàng nhỏ.',
    },
    perfect: {
      en: 'We have never undersold our products.',
      vi: 'Chúng tôi chưa bao giờ bán rẻ sản phẩm của mình.',
    },
    mistakeVi: 'undersell – undersold – undersold (giống sell – sold – sold).',
  },
  understand: {
    past: { en: 'I understood the lesson yesterday.', vi: 'Hôm qua tôi đã hiểu bài.' },
    perfect: { en: 'Have you understood the question?', vi: 'Bạn đã hiểu câu hỏi chưa?' },
    mistakeVi: 'Không viết understanded: understand – understood – understood.',
  },
  undertake: {
    past: {
      en: 'She undertook a new project last month.',
      vi: 'Tháng trước cô ấy đã đảm nhận một dự án mới.',
    },
    perfect: {
      en: 'He has undertaken a difficult task.',
      vi: 'Anh ấy đã nhận một nhiệm vụ khó khăn.',
    },
    mistakeVi: 'undertook là V2, undertaken là V3 (giống take – took – taken).',
  },
  underwrite: {
    past: {
      en: 'The bank underwrote the loan last year.',
      vi: 'Năm ngoái ngân hàng đã bảo lãnh khoản vay.',
    },
    perfect: {
      en: 'An insurance company has underwritten the ship.',
      vi: 'Một công ty bảo hiểm đã bảo hiểm con tàu.',
    },
    mistakeVi: 'underwrote là V2, underwritten là V3 (giống write).',
  },
  undo: {
    past: { en: 'The baby undid his shoelaces yesterday.', vi: 'Hôm qua em bé đã tháo dây giày.' },
    perfect: { en: 'Have you undone the knot?', vi: 'Bạn đã gỡ nút thắt chưa?' },
    mistakeVi: 'undid là V2, undone là V3 (giống do – did – done).',
  },
  unfreeze: {
    past: { en: 'I unfroze the chicken last night.', vi: 'Tối qua tôi đã rã đông thịt gà.' },
    perfect: { en: 'Have you unfrozen the fish yet?', vi: 'Bạn đã rã đông cá chưa?' },
    mistakeVi: 'unfroze là V2, unfrozen là V3; đời thường hay nói defrost.',
  },
  unwind: {
    past: {
      en: 'I unwound with a hot bath last night.',
      vi: 'Tối qua tôi đã thư giãn bằng một bồn nước nóng.',
    },
    perfect: {
      en: 'Have you unwound after your trip?',
      vi: 'Bạn đã nghỉ ngơi sau chuyến đi chưa?',
    },
    mistakeVi: 'unwind đọc /-waɪnd/; V2, V3 unwound đọc /-waʊnd/.',
  },
  uphold: {
    past: {
      en: 'The court upheld the decision last week.',
      vi: 'Tuần trước toà án đã giữ nguyên quyết định.',
    },
    perfect: {
      en: 'The school has upheld its rules for years.',
      vi: 'Nhà trường đã giữ vững nội quy nhiều năm.',
    },
    mistakeVi: 'Không viết upholded: giống hold – held – held.',
  },
  upset: {
    past: { en: 'The bad news upset my mum yesterday.', vi: 'Hôm qua tin xấu đã làm mẹ tôi buồn.' },
    perfect: { en: 'Have I upset you?', vi: 'Tôi có làm bạn buồn không?' },
    mistakeVi: 'upset giữ nguyên ở cả ba dạng, không viết upsetted.',
  },
  wake: {
    past: { en: 'I woke up at five this morning.', vi: 'Sáng nay tôi đã thức dậy lúc năm giờ.' },
    perfect: { en: 'The noise has woken the baby.', vi: 'Tiếng ồn đã đánh thức em bé.' },
    mistakeVi: 'woke là V2, woken là V3; không viết waked.',
  },
  waylay: {
    past: {
      en: 'A stranger waylaid me on the street yesterday.',
      vi: 'Hôm qua một người lạ đã chặn đường tôi.',
    },
    perfect: {
      en: 'Reporters have waylaid the singer outside the hotel.',
      vi: 'Phóng viên đã chặn ca sĩ ngoài khách sạn.',
    },
    mistakeVi: 'waylay – waylaid – waylaid (giống lay); không viết waylayed.',
  },
  wear: {
    past: {
      en: 'She wore a red dress to the party last night.',
      vi: 'Tối qua cô ấy đã mặc váy đỏ đi dự tiệc.',
    },
    perfect: { en: 'I have never worn a tie.', vi: 'Tôi chưa bao giờ đeo cà vạt.' },
    mistakeVi: 'wore là V2, worn là V3; không viết weared.',
  },
  weave: {
    past: {
      en: 'My grandmother wove this basket in 1990.',
      vi: 'Bà tôi đã đan chiếc giỏ này năm 1990.',
    },
    perfect: {
      en: 'They have woven silk here for centuries.',
      vi: 'Họ đã dệt lụa ở đây hàng thế kỷ.',
    },
    mistakeVi: 'wove là V2, woven là V3; không viết weaved.',
  },
  wed: {
    past: {
      en: 'They wed in a small church last spring.',
      vi: 'Mùa xuân năm ngoái họ đã cưới nhau ở một nhà thờ nhỏ.',
    },
    perfect: { en: 'The couple have wed in secret.', vi: 'Cặp đôi đã bí mật kết hôn.' },
    mistakeVi: 'wed và wedded đều đúng; đời thường hay nói get married.',
  },
  weep: {
    past: {
      en: 'She wept at the end of the film last night.',
      vi: 'Tối qua cô ấy đã khóc ở cuối phim.',
    },
    perfect: { en: 'He has wept for his dog all day.', vi: 'Anh ấy đã khóc vì con chó cả ngày.' },
    mistakeVi: 'Không viết weeped: weep – wept – wept (như sleep, keep).',
  },
  wet: {
    past: { en: 'The baby wet the bed last night.', vi: 'Tối qua em bé đã tè dầm.' },
    perfect: { en: 'The rain has wet all my clothes.', vi: 'Mưa đã làm ướt hết quần áo của tôi.' },
    mistakeVi: 'wet giữ nguyên ở V2 và V3; wetted ít dùng hơn.',
  },
  win: {
    past: { en: 'Our team won the cup last year.', vi: 'Năm ngoái đội chúng tôi đã giành cúp.' },
    perfect: {
      en: 'She has won three gold medals.',
      vi: 'Cô ấy đã giành được ba huy chương vàng.',
    },
    mistakeVi: 'won đọc /wʌn/ như one; không viết winned.',
  },
  wind: {
    past: {
      en: 'I wound the old clock last night.',
      vi: 'Tối qua tôi đã lên dây chiếc đồng hồ cũ.',
    },
    perfect: {
      en: 'He has wound the rope around the post.',
      vi: 'Anh ấy đã quấn sợi dây quanh cột.',
    },
    mistakeVi: 'wind /waɪnd/ – wound /waʊnd/; khác wound /wuːnd/ (vết thương).',
  },
  withdraw: {
    past: {
      en: 'I withdrew some money from the bank yesterday.',
      vi: 'Hôm qua tôi đã rút ít tiền ở ngân hàng.',
    },
    perfect: { en: 'She has withdrawn from the race.', vi: 'Cô ấy đã rút khỏi cuộc đua.' },
    mistakeVi: 'withdrew là V2, withdrawn là V3 (giống draw – drew – drawn).',
  },
  withhold: {
    past: {
      en: 'The company withheld his pay last month.',
      vi: 'Tháng trước công ty đã giữ lại lương của anh ấy.',
    },
    perfect: { en: 'They have withheld the truth from us.', vi: 'Họ đã giấu chúng tôi sự thật.' },
    mistakeVi: 'Không viết withholded: giống hold – held – held.',
  },
  withstand: {
    past: {
      en: 'The old house withstood the storm last night.',
      vi: 'Tối qua ngôi nhà cũ đã chống chọi được cơn bão.',
    },
    perfect: {
      en: 'This bridge has withstood many floods.',
      vi: 'Cây cầu này đã chịu được nhiều trận lũ.',
    },
    mistakeVi: 'Không viết withstanded: giống stand – stood – stood.',
  },
  wring: {
    past: {
      en: 'She wrung out the wet towel this morning.',
      vi: 'Sáng nay cô ấy đã vắt khô chiếc khăn ướt.',
    },
    perfect: { en: 'I have wrung out all the clothes.', vi: 'Tôi đã vắt hết quần áo rồi.' },
    mistakeVi: 'wring (vắt): chữ w không đọc, dễ nhầm với ring – rang – rung.',
  },
  write: {
    past: {
      en: 'I wrote a letter to my grandma last week.',
      vi: 'Tuần trước tôi đã viết thư cho bà.',
    },
    perfect: { en: 'Have you written your essay yet?', vi: 'Bạn đã viết bài luận chưa?' },
    mistakeVi: 'wrote là V2, written là V3 (gấp đôi t); không viết writed.',
  },
};
