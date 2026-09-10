#!/usr/bin/env python3
"""Trellis 流程门禁：在 start / archive 两个关键时刻做机器校验。

用法：
  python3 ./.trellis/scripts/flow_gate.py pre-start <task> [--tasks-root DIR]
  python3 ./.trellis/scripts/flow_gate.py mark-check <task> [--note "检查摘要"] [--tasks-root DIR]
  python3 ./.trellis/scripts/flow_gate.py pre-archive <task> [--tasks-root DIR]
  # 显式跳过（须写明原因，写入 .check-passed 备注）：
  ... pre-start|pre-archive <task> --force "原因"

退出码：0 通过；1 拒绝（信息明确到"缺什么、怎么补"）。
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
from pathlib import Path

COMPLEX_SCOPES = {"feature", "refactor", "cross-layer", "infra", "multi-deliverable"}
TBD_RE = re.compile(r"^\s*(?:[-*]\s*)?TBD\s*$", re.MULTILINE)
UNCHECKED_RE = re.compile(r"^\s*-\s*\[ \]\s*\S", re.MULTILINE)
CHECK_MARKER = ".check-passed"


class GateError(Exception):
    pass


def find_task_dir(tasks_root: Path, name: str) -> Path:
    direct = tasks_root / name
    if direct.is_dir():
        return direct
    archive = tasks_root / "archive"
    if archive.is_dir():
        hits = sorted(p for p in archive.glob(f"*/{name}") if p.is_dir())
        if hits:
            return hits[-1]
    raise GateError(f"找不到任务目录：{name}（在 {tasks_root} 与 {tasks_root}/archive/* 下均未命中）")


def load_task_json(task_dir: Path) -> dict:
    path = task_dir / "task.json"
    if not path.is_file():
        raise GateError(f"缺少 task.json：{path}")
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:  # pragma: no cover - 防御
        raise GateError(f"task.json 解析失败：{exc}") from exc


def require_prd(task_dir: Path) -> str:
    prd = task_dir / "prd.md"
    if not prd.is_file():
        raise GateError("缺少 prd.md —— 先写 PRD（需求/约束/验收标准），再 start")
    text = prd.read_text(encoding="utf-8")
    if TBD_RE.search(text):
        raise GateError("prd.md 仍含 TBD 占位 —— 填完需求与验收标准再 start")
    if "## Acceptance Criteria" not in text:
        raise GateError("prd.md 缺少 '## Acceptance Criteria' 段 —— 验收标准是规划门必备产物")
    if not re.search(r"^\s*-\s*\[[ x]\]", text, re.MULTILINE):
        raise GateError("prd.md 的验收标准为空 —— 至少写一条可勾选判据")
    return text


def is_complex(task_json: dict) -> bool:
    scope = str(task_json.get("scope") or "").strip().lower()
    dev_type = str(task_json.get("dev_type") or "").strip().lower()
    if scope in COMPLEX_SCOPES or dev_type in COMPLEX_SCOPES:
        return True
    return False


def cmd_pre_start(args: argparse.Namespace) -> int:
    task_dir = find_task_dir(args.tasks_root, args.task)
    task_json = load_task_json(task_dir)
    require_prd(task_dir)
    scope = task_json.get("scope")
    if not scope:
        print("[WARN] task.json 未标注 scope（建议 `task.py set-scope <name> <scope>`）")
    if is_complex(task_json):
        missing = [f for f in ("design.md", "implement.md") if not (task_dir / f).is_file()]
        if missing:
            raise GateError(
                "复杂任务（scope=%s）在 start 前必须有 %s —— 先补齐再 start" % (scope, " 与 ".join(missing))
            )
    print(f"[OK] pre-start 通过：{task_dir.name}（scope={scope or '未标注'}）")
    print(f"     下一步：python3 ./.trellis/scripts/task.py start {args.task}")
    print("     提醒：实施前先按需读取 .trellis/spec/*/index.md；完成后跑 skill trellis-check")
    return 0


def cmd_mark_check(args: argparse.Namespace) -> int:
    task_dir = find_task_dir(args.tasks_root, args.task)
    marker = task_dir / CHECK_MARKER
    stamp = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    note = args.note or "质量门通过（go vet / go test / web type-check+build 等）"
    marker.write_text(f"checked_at={stamp}\nnote={note}\n", encoding="utf-8")
    print(f"[OK] 已记录 check 标记：{marker}（{stamp}）")
    print(f"     下一步：勾选 prd.md 验收项 → python3 ./.trellis/scripts/task.py archive {args.task}")
    return 0


def cmd_pre_archive(args: argparse.Namespace) -> int:
    task_dir = find_task_dir(args.tasks_root, args.task)
    text = require_prd(task_dir)
    if UNCHECKED_RE.search(text):
        raise GateError("prd.md 验收标准仍有未勾选项（`- [ ]`）——逐条核实并勾选后再 archive")
    if not (task_dir / CHECK_MARKER).is_file():
        raise GateError(
            "缺少 check 标记（.check-passed）——先跑 skill trellis-check 的质量门，"
            f"再执行 flow_gate.py mark-check {args.task}"
        )
    print(f"[OK] pre-archive 通过：{task_dir.name}")
    print("     下一步：")
    print(f"       1) python3 ./.trellis/scripts/task.py archive {args.task} --skip-branch-validation（调查类）")
    print("       2) python3 ./.trellis/scripts/add_session.py --title ... --commit <work-hash> --summary ...")
    print("       3) git push（archive 会自动提交 task；journal 提交后一并推送）")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Trellis 流程门禁")
    parser.add_argument("command", choices=["pre-start", "mark-check", "pre-archive"])
    parser.add_argument("task", help="任务名（如 09-10-enforce-trellis-strict-flow）")
    parser.add_argument("--tasks-root", default=".trellis/tasks", help="任务根目录（默认 .trellis/tasks，测试可覆盖）")
    parser.add_argument("--note", default="", help="mark-check 的备注")
    parser.add_argument("--force", default="", help="显式跳过门禁，必须写明原因（记录在 .check-passed）")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    args.tasks_root = Path(args.tasks_root)
    handlers = {
        "pre-start": cmd_pre_start,
        "mark-check": cmd_mark_check,
        "pre-archive": cmd_pre_archive,
    }
    try:
        return handlers[args.command](args)
    except GateError as exc:
        if args.force:
            print(f"[FORCED] 门禁已被显式跳过：{exc}")
            print(f"         原因：{args.force}")
            try:
                if args.command != "mark-check":
                    task_dir = find_task_dir(args.tasks_root, args.task)
                    (task_dir / CHECK_MARKER).write_text(
                        f"forced_at={dt.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
                        f"reason={args.force}\n",
                        encoding="utf-8",
                    )
            except GateError:
                pass
            return 0
        print(f"[REJECT] {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
