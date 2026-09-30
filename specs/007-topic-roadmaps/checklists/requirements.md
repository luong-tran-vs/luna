# Specification Quality Checklist: Chủ đề và lộ trình theo trình độ

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-30
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- Lần kiểm tra 1: đạt tất cả mục. Chi tiết kỹ thuật khối 2 (package `topic`, collection `topics`/`migrations`, endpoint, bỏ
  collection `roadmap`) để dành cho `/speckit-plan`.
- Bao phủ 4 tiêu chí nghiệm thu F14 trong `docs/phases/giai-doan-1.md`: lộ trình chỉ chứa bài của chủ đề (FR-011/013, SC-003);
  không xoá chủ đề còn bài (FR-003, SC-004); cảnh báo dưới 3 bài theo từng lộ trình (FR-014, SC-005); chuyển dữ liệu không mất
  bài, giữ thứ tự (FR-016–018, SC-001/002).
- Quyết định mặc định (không cần hỏi): đổi chủ đề của bài chỉ chuyển lộ trình khi bài đang ở lộ trình cũ; sửa trình độ chủ đề
  kéo theo trình độ của bài; gộp tên chủ đề cũ khác hoa thường/khoảng trắng; "Chung" là chủ đề thường.
- Thay đổi F2: FR-024 (một lộ trình chung) và ô trình độ/chủ đề gõ tự do bị thay thế. Phụ thuộc: F1 (quyền quản trị), F2. L
  dùng danh sách chủ đề theo trình độ và lộ trình của từng chủ đề.
