# fix-file-name-align-chinese

## Goal

修复 internal/file name_align 2 例存量测试失败：中文数字集号(如'第二十八集')应解析出正确集号并可识别模板，定位解析逻辑缺陷，修复后 go test 全绿并走 0.0.16 发布管线

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
