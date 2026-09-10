package upload

import (
	"context"
	"fmt"
	"strings"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/driver"
	"litepan/internal/eventbus"
)

// executeUpload 执行一次上传尝试。
// 返回 requeue=true 表示任务已被置回 pending（如账号网络冷却等待），
// 调用方（runTask）应重新进入队列等待，否则该任务将无人接管。
func (m *Manager) executeUpload(ctx context.Context, taskID string) bool {
	m.mu.Lock()
	st, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return false
	}
	resume := cloneMap(st.resumeData)
	resuming := len(resume) > 0
	progress, uploaded := resumedProgress(st)
	accountID := st.AccountID
	localPath := st.localPath
	cleanupLocalMode := st.CleanupLocalMode
	cleanupLocalPath := st.CleanupLocalPath
	fileName := st.FileName
	targetPath := st.TargetPath
	conflictPolicy := st.conflictPolicy
	m.mu.Unlock()

	msg := "正在上传到网盘"
	if resuming {
		msg = "正在继续上传到网盘"
	}
	started := false
	m.patch(taskID, func(st *taskState) {
		if ctx.Err() != nil || st.Status != StatusPending {
			return
		}
		started = true
		st.Status = StatusRunning
		st.Phase = PhaseUploading
		st.Progress = progress
		st.UploadedBytes = uploaded
		st.SpeedBytesPerSecond = 0
		st.Message = msg
		st.Error = ""
		st.speed.Reset()
	})
	if !started {
		return false
	}

	entryName := uploadEntryName(fileName)

	result, err := m.runLocalUpload(ctx, accountID, driver.LocalUploadRequest{
		LocalPath:      localPath,
		FileName:       entryName,
		ParentID:       targetPath,
		ConflictPolicy: conflictPolicy,
		ResumeState:    resume,
		OnResumeState: func(state map[string]any) {
			m.applyResumeState(taskID, state)
		},
		OnProgress: func(uploaded, total int64, message string) {
			m.updateProgress(taskID, uploaded, total, message)
		},
	})

	m.mu.Lock()
	st, ok = m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return false
	}
	mode := st.cancelMode
	m.mu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			if mode == "pause" {
				m.patch(taskID, func(st *taskState) {
					st.Status = StatusPaused
					st.SpeedBytesPerSecond = 0
					st.Message = "上传已暂停"
				})
				return false
			}
			m.patch(taskID, func(st *taskState) {
				st.Status = StatusCanceled
				st.SpeedBytesPerSecond = 0
				st.Message = "上传任务已取消"
				st.Error = "上传任务已取消"
			})
			return false
		}
		// 账号网络冷却：可等待的瞬时状态，不是任务终态失败。
		// 退回 pending（置顶、保留进度/resumeData），等冷却结束再继续——
		// 并发=1 时天然让整条队列等待，避免"零 I/O 空转"秒级判死整批（0.0.24）。
		// 0.0.30：仅在任务仍可重试时置 pending（不得覆盖同期暂停），
		// 且等待结束后返回 requeue=true 让 runTask 重入队列（否则任务成为孤儿）。
		if seconds, cooling := driverexec.IsCooldownError(err); cooling {
			if !m.canCooldownWait(taskID) {
				return false
			}
			m.patch(taskID, func(st *taskState) {
				st.Status = StatusPending
				st.SpeedBytesPerSecond = 0
				st.Message = fmt.Sprintf("账号网络冷却中，%d 秒后自动重试", seconds)
				st.Error = ""
				st.resumePriority = true
			})
			m.mu.Lock()
			m.runCond.Broadcast()
			m.mu.Unlock()
			select {
			case <-ctx.Done():
				return false
			case <-time.After(time.Duration(seconds) * time.Second):
			}
			return m.canCooldownWait(taskID)
		}
		if shouldResetResumeState(err.Error()) {
			m.patch(taskID, func(st *taskState) {
				st.Status = StatusFailed
				st.SpeedBytesPerSecond = 0
				st.Message = "上传失败"
				st.Error = translateError(err.Error())
				st.resumeData = nil
				st.UploadedBytes = 0
				st.Progress = 0
			})
			return false
		}
		m.failTask(taskID, err)
		return false
	}

	status := StatusSuccess
	msg = result.Message
	if result.Skipped {
		status = StatusSkipped
	}
	m.cleanupLocalSource(localPath, cleanupLocalPath, cleanupLocalMode)
	m.patch(taskID, func(st *taskState) {
		st.Status = status
		st.Phase = PhaseUploading
		st.Progress = 100
		st.DownloadedBytes = st.TotalBytes
		st.UploadedBytes = st.TotalBytes
		st.SpeedBytesPerSecond = 0
		st.Message = msg
		st.Error = ""
		st.resumeData = nil
		resultMeta := retainBatchRootMetadata(st.Result)
		st.Result = map[string]any{
			"file_id":   result.FileID,
			"parent_id": result.ParentID,
			"file_name": result.FileName,
			"size":      result.Size,
		}
		for key, value := range resultMeta {
			st.Result[key] = value
		}
	})
	if m.files == nil && m.bus != nil {
		parentID := result.ParentID
		if parentID == "" {
			parentID = targetPath
		}
		m.bus.Publish(context.Background(), eventbus.FileMutated{
			AccountID: accountID,
			Op:        "upload",
			ParentID:  parentID,
			FileID:    result.FileID,
		})
	}
	return false
}
func shouldResetResumeState(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	return strings.Contains(lower, "invalidpartorder") || strings.Contains(lower, "previous part hash context")
}

func (m *Manager) taskLocalPath(taskID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.tasks[taskID]; ok {
		return st.localPath
	}
	return ""
}

func (m *Manager) deleteUploadedFile(ctx context.Context, st *taskState) error {
	if st.Result == nil {
		return nil
	}
	raw, _ := st.Result["file_id"].(string)
	if raw == "" {
		return nil
	}
	if err := m.exec.Check(ctx, st.AccountID); err != nil {
		return err
	}
	err := m.exec.Run(ctx, st.AccountID, func(drv driver.Driver) error {
		deleter, err := driverexec.Require[driver.Deleter](drv)
		if err != nil {
			return err
		}
		return deleter.DeleteFiles(ctx, []string{raw})
	})
	if err != nil {
		return err
	}
	m.publishUploadedFileDeleted(st, raw)
	return nil
}

// deleteUploadedFiles 批量删除已上传的网盘文件（一次请求，驱动支持批量删除）。
func (m *Manager) deleteUploadedFiles(ctx context.Context, accountID int64, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}
	if err := m.exec.Check(ctx, accountID); err != nil {
		return err
	}
	return m.exec.Run(ctx, accountID, func(drv driver.Driver) error {
		deleter, err := driverexec.Require[driver.Deleter](drv)
		if err != nil {
			return err
		}
		return deleter.DeleteFiles(ctx, fileIDs)
	})
}

func (m *Manager) publishUploadedFileDeleted(st *taskState, fileID string) {
	if m.bus == nil {
		return
	}
	parentID, _ := st.Result["parent_id"].(string)
	if parentID == "" {
		parentID = st.TargetPath
	}
	m.bus.Publish(context.Background(), eventbus.FileMutated{
		AccountID: st.AccountID,
		Op:        "delete",
		ParentID:  parentID,
		FileIDs:   []string{fileID},
	})
}


// canCooldownWait 判断任务此刻是否适合进入"冷却等待"（可重试且未被暂停/取消/停止）。
func (m *Manager) canCooldownWait(taskID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.tasks[taskID]
	if !ok || m.stopping {
		return false
	}
	if st.cancelMode == "pause" || st.Status == StatusPaused || st.Status == StatusCanceled {
		return false
	}
	return st.Status == StatusRunning || st.Status == StatusPending
}
