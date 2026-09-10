package upload

import (
	"fmt"
	"strings"

	"litepan/internal/domain"
)

// batchBreakerThreshold：同批次连续同因系统级失败达到该数量 → 自动暂停批次剩余
// 任务（熔断安全网）。目的：账号级故障（网络/认证/内部）下不再成百地无效判死。
const batchBreakerThreshold = 5

// batchKeyOf 返回熔断分组键：优先批次 id；空批次退化为「账号 + 目标目录」，
// 两者都没有时退化为账号级。
//
// 0.0.35：此前直接要求 BatchID 非空才计数，而自动化（local_upload）创建的任务
// batch_id 为空，导致这批任务完全没有熔断保护（账号级故障时可成百地无效判死）。
func batchKeyOf(st *taskState) string {
	if st == nil {
		return ""
	}
	if id := strings.TrimSpace(st.BatchID); id != "" {
		return "batch:" + id
	}
	if target := strings.TrimSpace(st.TargetPath); target != "" {
		return fmt.Sprintf("acct:%d|target:%s", st.AccountID, target)
	}
	return fmt.Sprintf("acct:%d", st.AccountID)
}

// batchFailureRecord 记录同一分组键的连续同因失败。
type batchFailureRecord struct {
	sig   string
	count int
}

// failureSignature 失败签名：错误码 + 文案前缀（同因归并、异因重置）。
func failureSignature(err error) string {
	if err == nil {
		return ""
	}
	code := ""
	if ae, ok := domain.AsAppError(err); ok {
		code = string(ae.Code)
	}
	msg := []rune(strings.TrimSpace(err.Error()))
	if len(msg) > 40 {
		msg = msg[:40]
	}
	return code + "|" + string(msg)
}

// systemicFailure 判定是否为账号/系统级失败（值得熔断），文件级错误
// （参数/不存在/权限/名称非法）不计入，避免误伤单文件问题。
func systemicFailure(err error) bool {
	ae, ok := domain.AsAppError(err)
	if !ok {
		return false
	}
	switch ae.Code {
	case domain.CodeDriverError, domain.CodeInternal, domain.CodeRateLimited, domain.CodeAuthExpired:
		return true
	}
	return false
}

// observeBatchFailure 累计同一熔断分组（见 batchKeyOf）的连续同因失败；
// 达到阈值时暂停该分组剩余任务。计数与挑选必须使用同一个键。
func (m *Manager) observeBatchFailure(taskID string, err error) {
	sig := failureSignature(err)
	if sig == "" || !systemicFailure(err) {
		return
	}
	m.mu.Lock()
	st, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return
	}
	key := batchKeyOf(st)
	if key == "" {
		m.mu.Unlock()
		return
	}
	if m.batchFailures == nil {
		m.batchFailures = make(map[string]batchFailureRecord)
	}
	rec := m.batchFailures[key]
	if rec.sig != sig {
		rec = batchFailureRecord{sig: sig}
	}
	rec.count++
	tripped := rec.count >= batchBreakerThreshold
	if tripped {
		delete(m.batchFailures, key)
	} else {
		m.batchFailures[key] = rec
	}
	var pending []string
	if tripped {
		for id, other := range m.tasks {
			if other.Status == StatusPending && batchKeyOf(other) == key {
				pending = append(pending, id)
			}
		}
	}
	count := rec.count
	m.mu.Unlock()
	if !tripped {
		return
	}
	reason := fmt.Sprintf("批次已自动暂停：连续 %d 个任务同因失败（%s），请排查账号状态后恢复", count, systemFailureLabel(err))
	for _, id := range pending {
		m.patch(id, func(st *taskState) {
			st.Status = StatusPaused
			st.SpeedBytesPerSecond = 0
			st.Message = reason
			st.Error = ""
		})
	}
}

// systemFailureLabel 给用户看的简短原因标签。
func systemFailureLabel(err error) string {
	if ae, ok := domain.AsAppError(err); ok {
		switch ae.Code {
		case domain.CodeAuthExpired:
			return "账号认证已失效"
		case domain.CodeRateLimited:
			return "账号限流"
		case domain.CodeInternal:
			return "服务内部错误"
		}
	}
	msg := strings.TrimSpace(err.Error())
	if i := strings.IndexByte(msg, ':'); i > 0 && i < 30 {
		return strings.TrimSpace(msg[:i])
	}
	if len([]rune(msg)) > 30 {
		return string([]rune(msg)[:30])
	}
	return msg
}
