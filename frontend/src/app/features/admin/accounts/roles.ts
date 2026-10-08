import { Role } from '../../../core/models/user';
import { IconName } from '../../../shared/components/icon/icon';

/** How each role shows on the accounts page, from most to least access. */
export const ROLE_INFO: readonly { role: Role; name: string; icon: IconName; description: string }[] = [
  { role: 'admin', name: 'Quản trị', icon: 'sliders', description: 'Quản lý bài học, nội dung và tài khoản.' },
  { role: 'member', name: 'Thành viên', icon: 'user', description: 'Học mọi bài và phần Ngữ pháp.' },
  { role: 'guest', name: 'Khách', icon: 'lock', description: 'Chỉ học bài đầu của mỗi chủ đề; các bài sau bị khoá.' },
];
