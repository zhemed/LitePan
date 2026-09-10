package upload

import (
	"context"
	"strings"

	"litepan/internal/domain"
)

func (m *Manager) Delete(ctx context.Context, taskID string, deleteUploadedFile bool) (bool, error) {
	m.mu.Lock()
	st, ok := m.tasks[taskID]
	var snapshot *taskState
	if ok {
		snapshot = &taskState{Task: *m.snapshot(st)}
	}
	m.mu.Unlock()
	if !ok {
		return false, nil
	}
	if deleteUploadedFile && snapshot.Status == StatusSuccess {
		if err := m.deleteUploadedFile(ctx, snapshot); err != nil {
			return true, err
		}
	}
	if err := m.stopTaskForDelete(ctx, taskID); err != nil {
		return true, err
	}
	popped := m.popTask(taskID)
	if popped == nil {
		return true, domain.Errorf(domain.CodeInternal, "任务正在停止，请稍后再次删除")
	}
	m.cleanupLocalSourceAfterDelete(popped)
	m.broadcastDeleted(taskID)
	return true, nil
}

func (m *Manager) BatchDelete(ctx context.Context, taskIDs []string, deleteUploadedFile, deleteBatchRoots bool) BatchDeleteResult {
	result := BatchDeleteResult{FailedMessages: map[string]string{}}
	seen := map[string]struct{}{}
	type item struct {
		id     string
		task   Task
		fileID string
	}
	m.mu.Lock()
	var items []item
	for _, id := range taskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		st, ok := m.tasks[id]
		if !ok {
			result.MissingTaskIDs = append(result.MissingTaskIDs, id)
			continue
		}
		fileID := ""
		task := *m.snapshot(st)
		if deleteUploadedFile && task.Status == StatusSuccess && task.Result != nil {
			fileID, _ = task.Result["file_id"].(string)
		}
		items = append(items, item{id: id, task: task, fileID: strings.TrimSpace(fileID)})
	}
	m.mu.Unlock()
	if deleteUploadedFile {
		// 只有从任务面板根层删除完整目录批次时，才尝试一次删除本次上传新建的顶层目录。
		// 复用的旧目录、只选中子目录/部分任务或旧任务缺少元数据时，继续走逐文件批量删除。
		rootDeletedBatches := map[string]struct{}{}
		failedByID := map[string]string{}
		if deleteBatchRoots && m.files != nil {
			selectedByBatch := map[string][]item{}
			for _, it := range items {
				if batchID := strings.TrimSpace(it.task.BatchID); batchID != "" {
					selectedByBatch[batchID] = append(selectedByBatch[batchID], it)
				}
			}
			for batchID, batchItems := range selectedByBatch {
				selectedIDs := make(map[string]struct{}, len(batchItems))
				for _, it := range batchItems {
					selectedIDs[it.id] = struct{}{}
				}
				m.mu.Lock()
				completeBatch := true
				for taskID, candidate := range m.tasks {
					if candidate.BatchID != batchID {
						continue
					}
					if _, ok := selectedIDs[taskID]; !ok {
						completeBatch = false
						break
					}
				}
				m.mu.Unlock()
				if !completeBatch {
					continue
				}
				// 第二层保护（0.0.29）：要求选中数 == 该批历史总数（result.batch_task_total）。
				// 历史批次缺该字段 → 保守拒绝根删除（仍可逐文件删除），避免记录被清理后
				// "部分选择"被误判为"整批选择"而误删云端目录。
				total := 0
				consistentTotal := true
				for _, it := range batchItems {
					if v := batchTaskTotalOf(it.task.Result); v > 0 {
						if total == 0 {
							total = v
						} else if total != v {
							consistentTotal = false
							break
						}
					}
				}
				if !consistentTotal {
					total = 0
				}
				if total <= 0 || len(batchItems) != total {
					continue
				}

				accountID := batchItems[0].task.AccountID
				rootID, rootParentID := "", ""
				rootName := strings.TrimSpace(batchItems[0].task.BatchName)
				owned := false
				consistent := true
				for _, it := range batchItems {
					if !isTerminalUploadStatus(it.task.Status) || it.task.AccountID != accountID || it.task.Result == nil || strings.TrimSpace(it.task.BatchName) != rootName {
						consistent = false
						break
					}
					itemRootID, _ := it.task.Result["batch_root_id"].(string)
					itemRootParentID, _ := it.task.Result["batch_root_parent_id"].(string)
					itemOwned, _ := it.task.Result["batch_root_owned"].(bool)
					itemRootID = strings.TrimSpace(itemRootID)
					itemRootParentID = strings.TrimSpace(itemRootParentID)
					if itemRootID == "" || !itemOwned {
						consistent = false
						break
					}
					if rootID == "" {
						rootID, rootParentID, owned = itemRootID, itemRootParentID, itemOwned
						continue
					}
					if itemRootID != rootID || itemRootParentID != rootParentID || itemOwned != owned {
						consistent = false
						break
					}
				}
				if !consistent || rootID == "" || rootName == "" || !owned {
					continue
				}
				// 批次根信息来自上传请求。删除前再向网盘确认它仍是指定父目录下的
				// 同名文件夹；校验失败时退回逐文件删除，避免错误元数据误删其它目录。
				entries, err := m.files.List(ctx, accountID, rootParentID, true)
				if err != nil {
					continue
				}
				rootConfirmed := false
				for _, entry := range entries {
					if entry.ID == rootID && entry.IsDir && entry.Name == rootName {
						rootConfirmed = true
						break
					}
				}
				if !rootConfirmed {
					continue
				}
				if err := m.files.DeleteFiles(ctx, accountID, []string{rootID}, rootParentID); err != nil {
					for _, it := range batchItems {
						failedByID[it.id] = err.Error()
					}
					continue
				}
				rootDeletedBatches[batchID] = struct{}{}
			}
		}

		type groupKey struct {
			accountID int64
			parent    string
		}
		byGroup := map[groupKey][]string{}
		var order []groupKey
		for _, it := range items {
			if _, deleted := rootDeletedBatches[it.task.BatchID]; deleted {
				continue
			}
			if _, failed := failedByID[it.id]; failed {
				continue
			}
			if it.fileID == "" {
				continue
			}
			g := groupKey{accountID: it.task.AccountID, parent: it.task.TargetPath}
			if _, ok := byGroup[g]; !ok {
				order = append(order, g)
			}
			byGroup[g] = append(byGroup[g], it.fileID)
		}
		for _, g := range order {
			var err error
			if m.files != nil {
				err = m.files.DeleteFiles(ctx, g.accountID, byGroup[g], g.parent)
			} else {
				err = m.deleteUploadedFiles(ctx, g.accountID, byGroup[g])
			}
			if err != nil {
				for _, it := range items {
					if it.task.AccountID == g.accountID && it.task.TargetPath == g.parent && it.fileID != "" {
						failedByID[it.id] = err.Error()
					}
				}
			}
		}
		for id, msg := range failedByID {
			result.FailedTaskIDs = append(result.FailedTaskIDs, id)
			result.FailedMessages[id] = msg
		}
		for _, g := range order {
			if g.parent == "" || m.files == nil {
				continue
			}
			// 0.0.29 安全加固：空目录自动清理必须同时满足
			//   ①用户显式勾选"同时删除批次根目录"（deleteBatchRoots）
			//   ②该父目录确为本次删除任务记录的 owned 批次根（元数据一致）
			// 否则只删文件、保留目录——避免"用户只删了几条记录却把文件夹删掉"，
			// 也避免误删用户自建/复用的目录。
			if !deleteBatchRoots {
				continue
			}
			groupRootID, groupOwned := "", false
			for _, it := range items {
				if it.task.AccountID != g.accountID || it.task.TargetPath != g.parent {
					continue
				}
				if it.task.Result == nil {
					continue
				}
				rootID, _ := it.task.Result["batch_root_id"].(string)
				owned, _ := it.task.Result["batch_root_owned"].(bool)
				if strings.TrimSpace(rootID) == g.parent && owned {
					groupRootID, groupOwned = g.parent, true
					break
				}
			}
			if !groupOwned || groupRootID == "" {
				continue
			}
			groupFailed := false
			for _, it := range items {
				if it.task.AccountID == g.accountID && it.task.TargetPath == g.parent && it.fileID != "" {
					if _, bad := failedByID[it.id]; bad {
						groupFailed = true
						break
					}
				}
			}
			if groupFailed {
				continue
			}
			// 空判定强制走云端实时列表（forceRefresh），避免缓存导致的"假空"误删。
			entries, lerr := m.files.List(ctx, g.accountID, g.parent, true)
			if lerr != nil || len(entries) > 0 {
				continue
			}
			_ = m.files.DeleteFiles(ctx, g.accountID, []string{g.parent}, "")
		}
	}
	for _, it := range items {
		if _, failed := result.FailedMessages[it.id]; failed {
			continue
		}
		if err := m.stopTaskForDelete(ctx, it.id); err != nil {
			result.FailedTaskIDs = append(result.FailedTaskIDs, it.id)
			result.FailedMessages[it.id] = err.Error()
			continue
		}
		popped := m.popTask(it.id)
		if popped == nil {
			result.FailedTaskIDs = append(result.FailedTaskIDs, it.id)
			result.FailedMessages[it.id] = "任务正在停止，请稍后再次删除"
			continue
		}
		m.cleanupLocalSourceAfterDelete(popped)
		result.DeletedTaskIDs = append(result.DeletedTaskIDs, it.id)
	}
	if len(result.DeletedTaskIDs) > 0 {
		m.broadcastDeleted(result.DeletedTaskIDs...)
	}
	return result
}

func isTerminalUploadStatus(status string) bool {
	switch status {
	case StatusSuccess, StatusFailed, StatusCanceled, StatusSkipped:
		return true
	default:
		return false
	}
}

// 删除任务时不碰服务器上传的用户源文件。
func (m *Manager) cleanupLocalSourceAfterDelete(st *taskState) {
	if st == nil || st.SourceType == SourceTypeServerLocal {
		return
	}
	if st.CleanupLocalMode != "" {
		m.cleanupLocalSource(st.localPath, st.CleanupLocalPath, st.CleanupLocalMode)
		return
	}
	m.removeLocalFile(st.localPath)
}

// batchTaskTotalOf 读取批次历史总数（创建时写入 result.batch_task_total）。
func batchTaskTotalOf(result map[string]any) int {
	if result == nil {
		return 0
	}
	switch v := result["batch_task_total"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}
