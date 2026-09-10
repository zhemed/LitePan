// 记忆化逻辑回归校验（web 无测试运行器，用 TypeScript 编译器转译后做 Node 断言）
// 运行：npm run check:memo
import { readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import ts from "typescript";

const source = readFileSync(new URL("../src/composables/upload/uploadRowMemo.ts", import.meta.url), "utf8");
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022, isolatedModules: true },
});
const modulePath = join(tmpdir(), "litepan-memo-check.mjs");
writeFileSync(modulePath, outputText);
const { createRowMemo, createNodeMemo, nodeSignature } = await import(modulePath);

let fail = 0;
const check = (ok, msg) => { if (!ok) { fail++; console.log("FAIL |", msg); } else console.log("PASS |", msg); };

// 1) 行记忆化：5000 任务首轮全建，第二轮仅 3 个变化任务重建
const tasks = Array.from({ length: 5000 }, (_, i) => ({
  task_id: `t${i}`, status: "running", updated_at: 1000 + i, file_name: `f${i}.bin`,
}));
const memo = createRowMemo((t) => ({ id: t.task_id, upd: t.updated_at }));
tasks.forEach((t) => memo.get(t));
check(memo.stats.builds === 5000 && memo.stats.hits === 0, `首轮全建 builds=${memo.stats.builds}`);
const before = memo.stats.builds;
tasks.forEach((t) => memo.get(t));
check(memo.stats.builds === before, `未变化零重建（builds 仍 ${memo.stats.builds}）`);
check(memo.stats.hits === 5000, `命中 5000（实际 ${memo.stats.hits}）`);
for (const i of [7, 999, 4999]) { const t = tasks[i]; memo.get({ ...t, updated_at: t.updated_at + 1 }); }
check(memo.stats.builds === before + 3, `3 个变化任务重建（实际 +${memo.stats.builds - before}）`);

// 2) 上限淘汰：limit=10，插入 20 条后 size 不超过 10
const small = createRowMemo((t) => t.task_id, 10);
for (let i = 0; i < 20; i++) small.get({ task_id: `s${i}`, updated_at: i });
check(small.size() <= 10, `上限保护生效 size=${small.size()}`);

// 3) 节点记忆化：签名未变复用；批次内任一任务变化即重建
const node = (name, list) => ({ id: `task:${name}`, name, tasks: list, isFolder: false, batchId: "", path: "" });
const n1 = node("a", [{ task_id: "x", status: "running", updated_at: 5 }]);
const nodeMemo = createNodeMemo((n) => ({ n: n.name, sig: nodeSignature(n) }));
nodeMemo.get(n1);
nodeMemo.get(n1);
check(nodeMemo.stats.builds === 1 && nodeMemo.stats.hits === 1, `节点签名未变复用 builds=${nodeMemo.stats.builds} hits=${nodeMemo.stats.hits}`);
const n1b = node("a", [{ task_id: "x", status: "success", updated_at: 6 }]);
nodeMemo.get(n1b);
check(nodeMemo.stats.builds === 2, `状态/时间变化重建（builds=${nodeMemo.stats.builds}）`);
const folderA = { id: "f:batch1", name: "b", isFolder: true, batchId: "batch1", path: "", tasks: tasks.slice(0, 10) };
const folderB = { ...folderA, tasks: tasks.slice(0, 10).map((t, i) => (i === 5 ? { ...t, status: "failed", updated_at: t.updated_at + 1 } : t)) };
const s1 = nodeSignature(folderA), s2 = nodeSignature(folderB);
check(s1 !== s2, "文件夹节点签名捕获内部任务状态变化");
const same = nodeSignature(folderA);
check(same === s1, "同结构签名稳定");

console.log(fail === 0 ? "MEMO-ALL-PASS" : `MEMO-FAILURES=${fail}`);
process.exit(fail === 0 ? 0 : 1);
