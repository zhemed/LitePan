package upload

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
	"litepan/internal/eventbus"
	"litepan/internal/file"
	"litepan/internal/playback"
	"litepan/internal/settings"
	"litepan/pkg/timeutil"
)

type AccountLookup interface {
	LookupUploadAccount(ctx context.Context, accountID int64) (name, driverType string, err error)
}

type Options struct {
	Exec        *driverexec.Executor
	Files       *file.Service
	Playback    *playback.Service
	Accounts    AccountLookup
	Repo        domain.UploadTaskRepository
	Settings    *settings.Service
	Bus         *eventbus.Bus
	DataDir     string
	Log         *slog.Logger
	StartupGate <-chan struct{}
}

type Manager struct {
	exec        *driverexec.Executor
	files       *file.Service
	playback    *playback.Service
	accounts    AccountLookup
	repo        domain.UploadTaskRepository
	settings    *settings.Service
	bus         *eventbus.Bus
	dataDir     string
	log         *slog.Logger
	startupGate <-chan struct{}

	mu                      sync.Mutex
	tasks                   map[string]*taskState
	queueOrder              int
	limit                   int
	runningUploads          int
	runCond                 sync.Cond
	subs                    map[chan []byte]struct{}
	broadcastPending        bool
	broadcastAllDirty       bool
	broadcastDirtyTaskIDs   map[string]struct{}
	broadcastDeletedTaskIDs map[string]struct{}
	subMu                   sync.Mutex
	clientTaskIndex         map[string]string
	tempRegistry            *TempRegistry
	targetDirCache          *uploadTargetDirCache
	runCtx                  context.Context
	runCancel               context.CancelFunc
	stopping                bool

	resumePersistMu sync.Mutex
	resumePersist   map[string]*time.Timer

	// 批次目录预热去重：同一 (账号, 根目录) 同时只有一个预热协程。
	batchFailures map[string]batchFailureRecord

	// persistMu 串行化状态落库（0.0.34）：保证「更晚的迁移」一定在「更早的迁移」
	// 之后写库，避免过期快照覆盖新状态（内存 paused / 库内 pending 分叉的来源）。
	persistMu sync.Mutex

	batchWarmingMu sync.Mutex
	batchWarming   map[batchWarmKey]struct{}
}

type batchWarmKey struct {
	accountID int64
	rootID    string
}

func NewManager(opts Options) *Manager {
	runCtx, runCancel := context.WithCancel(context.Background())
	m := &Manager{
		exec:                    opts.Exec,
		files:                   opts.Files,
		playback:                opts.Playback,
		accounts:                opts.Accounts,
		repo:                    opts.Repo,
		settings:                opts.Settings,
		bus:                     opts.Bus,
		dataDir:                 opts.DataDir,
		log:                     opts.Log,
		startupGate:             opts.StartupGate,
		tasks:                   make(map[string]*taskState),
		limit:                   defaultLimit,
		subs:                    make(map[chan []byte]struct{}),
		clientTaskIndex:         make(map[string]string),
		broadcastDirtyTaskIDs:   make(map[string]struct{}),
		broadcastDeletedTaskIDs: make(map[string]struct{}),
		targetDirCache:          newUploadTargetDirCache(),
		runCtx:                  runCtx,
		runCancel:               runCancel,
	}
	m.runCond.L = &m.mu
	if m.log == nil {
		m.log = slog.Default()
	}
	m.tempRegistry = NewTempRegistry()
	_ = m.RefreshConcurrencyLimit(context.Background())
	m.restoreTasks()
	m.initTempCleanup()
	// 0.0.28：上传记录保留策略（启动即清理一次，之后每小时；随 runCtx 退出）
	if m.repo != nil {
		go m.retentionLoop(runCtx)
	}
	return m
}

func (m *Manager) TempDir() string {
	return TempDir(m.dataDir)
}

func (m *Manager) Create(ctx context.Context, p CreateParams) (*Task, error) {
	tasks, err := m.createBatch(ctx, []CreateParams{p})
	if err != nil {
		return nil, err
	}
	return tasks[0], nil
}

func (m *Manager) CreateBatch(ctx context.Context, params []CreateParams) ([]*Task, error) {
	return m.createBatch(ctx, params)
}

// RenameTask 仅修改尚未开始的任务。
func (m *Manager) RenameTask(_ context.Context, taskID, newName, newTargetPath, newDisplayPath string) (bool, error) {
	name := strings.TrimSpace(newName)
	if name == "" {
		return false, nil
	}
	renamed := false
	m.patch(taskID, func(st *taskState) {
		if st.Status != StatusPending && st.Status != StatusPaused {
			return
		}
		st.FileName = name
		if newTargetPath != "" {
			st.TargetPath = newTargetPath
		}
		if newDisplayPath != "" {
			st.TargetDisplayPath = newDisplayPath
		}
		renamed = true
	})
	return renamed, nil
}

func (m *Manager) createBatch(ctx context.Context, params []CreateParams) ([]*Task, error) {
	if len(params) == 0 {
		return nil, domain.Errorf(domain.CodeValidation, "上传任务不能为空")
	}
	prepared := make([]CreateParams, len(params))
	for i, p := range params {
		var err error
		prepared[i], err = m.normalizeCreateParams(ctx, p)
		if err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	if m.stopping {
		m.mu.Unlock()
		return nil, domain.Errorf(domain.CodeInternal, "上传服务正在停止")
	}
	result := make([]*Task, len(prepared))
	created := make([]*taskState, 0, len(prepared))
	for i, p := range prepared {
		if existing := m.findByClientTaskIDLocked(p.ClientTaskID); existing != nil {
			result[i] = m.snapshot(existing)
			continue
		}
		st := m.newTaskStateLocked(p)
		m.addTaskLocked(st)
		created = append(created, st)
		result[i] = m.snapshot(st)
	}
	m.mu.Unlock()

	persisted := make([]string, 0, len(created))
	for _, st := range created {
		if err := m.persistTask(st); err != nil {
			m.mu.Lock()
			for _, item := range created {
				m.removeTaskLocked(item.TaskID)
			}
			m.mu.Unlock()
			for _, id := range persisted {
				m.deletePersisted(id)
			}
			return nil, domain.Wrap(domain.CodeInternal, err)
		}
		persisted = append(persisted, st.TaskID)
	}
	if len(created) > 0 {
		ids := make([]string, 0, len(created))
		for _, st := range created {
			ids = append(ids, st.TaskID)
		}
		m.broadcast(ids...)
	}
	for _, st := range created {
		go m.runTask(st.TaskID)
	}
	m.preWarmBatchDirs(created)
	return result, nil
}

// preWarmBatchDirs 收集批次内全部唯一 (账号, 根目录, rel_dir)，后台预热目录缓存。
// 去重 + 字典序（父前缀先行），同批次重复创建（分块建批）只预热一次；
// 预热与上传 worker 共用账号间隔门，不改变请求节奏，只消除边传边解析的长尾。
func (m *Manager) preWarmBatchDirs(created []*taskState) {
	if len(created) == 0 || m.files == nil {
		return
	}
	dirs := collectBatchWarmDirs(created)
	for k, list := range dirs {
		m.batchWarmingMu.Lock()
		if m.batchWarming == nil {
			m.batchWarming = make(map[batchWarmKey]struct{})
		}
		if _, warming := m.batchWarming[k]; warming {
			m.batchWarmingMu.Unlock()
			continue
		}
		m.batchWarming[k] = struct{}{}
		m.batchWarmingMu.Unlock()
		go func(k batchWarmKey, list []string) {
			defer func() {
				m.batchWarmingMu.Lock()
				delete(m.batchWarming, k)
				m.batchWarmingMu.Unlock()
			}()
			ctx := m.runCtx
			if ctx == nil {
				ctx = context.Background()
			}
			warmTargetDirs(ctx, m.files, m.targetDirCache, k.accountID, k.rootID, list)
		}(k, list)
	}
}

// collectBatchWarmDirs 汇总批次任务的唯一 (账号, 根目录, rel_dir)。
func collectBatchWarmDirs(created []*taskState) map[batchWarmKey][]string {
	sets := make(map[batchWarmKey]map[string]struct{})
	for _, st := range created {
		if st == nil || st.RelDir == "" {
			continue
		}
		k := batchWarmKey{accountID: st.AccountID, rootID: st.TargetPath}
		set, ok := sets[k]
		if !ok {
			set = make(map[string]struct{})
			sets[k] = set
		}
		set[st.RelDir] = struct{}{}
	}
	out := make(map[batchWarmKey][]string, len(sets))
	for k, set := range sets {
		list := make([]string, 0, len(set))
		for dir := range set {
			list = append(list, dir)
		}
		out[k] = list
	}
	return out
}

func normalizeClientTaskID(clientTaskID string) string {
	return strings.TrimSpace(clientTaskID)
}

func (m *Manager) addTaskLocked(st *taskState) {
	if st == nil {
		return
	}
	m.tasks[st.TaskID] = st
	clientTaskID := normalizeClientTaskID(st.ClientTaskID)
	if clientTaskID == "" {
		return
	}
	if m.clientTaskIndex == nil {
		m.clientTaskIndex = make(map[string]string)
	}
	m.clientTaskIndex[clientTaskID] = st.TaskID
}

func (m *Manager) removeTaskLocked(taskID string) *taskState {
	st, ok := m.tasks[taskID]
	if !ok {
		return nil
	}
	delete(m.tasks, taskID)
	clientTaskID := normalizeClientTaskID(st.ClientTaskID)
	if clientTaskID != "" {
		if indexedID, ok := m.clientTaskIndex[clientTaskID]; ok && indexedID == taskID {
			delete(m.clientTaskIndex, clientTaskID)
		}
	}
	return st
}

func (m *Manager) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	if !m.stopping {
		m.stopping = true
		for _, st := range m.tasks {
			if st.Status == StatusRunning {
				st.cancelMode = "pause"
			}
		}
		m.runCancel()
	}
	done := make([]chan struct{}, 0, len(m.tasks))
	seen := make(map[chan struct{}]struct{}, len(m.tasks))
	for _, st := range m.tasks {
		if st.runDone == nil {
			continue
		}
		if _, ok := seen[st.runDone]; ok {
			continue
		}
		seen[st.runDone] = struct{}{}
		done = append(done, st.runDone)
	}
	m.mu.Unlock()
	m.runCond.Broadcast()

	for _, ch := range done {
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (m *Manager) normalizeCreateParams(ctx context.Context, p CreateParams) (CreateParams, error) {
	if p.TotalBytes < 0 {
		return CreateParams{}, domain.Errorf(domain.CodeValidation, "上传文件大小非法")
	}
	if p.AccountName == "" || p.DriverType == "" {
		if m.accounts == nil {
			return CreateParams{}, domain.Errorf(domain.CodeInternal, "上传服务未配置账号查询")
		}
		var err error
		p.AccountName, p.DriverType, err = m.accounts.LookupUploadAccount(ctx, p.AccountID)
		if err != nil {
			return CreateParams{}, err
		}
	}
	return p, nil
}

func (m *Manager) newTaskStateLocked(p CreateParams) *taskState {
	name := p.DisplayName
	if name == "" {
		name = p.FileName
	}
	sourceType := p.SourceType
	if sourceType == "" {
		sourceType = SourceTypeManual
	}
	phase := p.Phase
	if phase == "" {
		phase = PhaseUploading
	}
	now := time.Now()
	m.queueOrder++
	order := m.queueOrder
	id := newTaskID()
	localPath := p.LocalPath
	cleanupLocalMode := p.CleanupLocalMode
	cleanupLocalPath := p.CleanupLocalPath
	if cleanupLocalMode == "" && localPath != "" {
		if sourceType == SourceTypeManual {
			cleanupLocalMode = CleanupLocalFileOnSuccess
		}
	}
	if cleanupLocalPath == "" {
		cleanupLocalPath = localPath
	}
	message := "等待上传"
	if sourceType == SourceTypeOfflineHandoff {
		message = "等待离线文件上传"
	}
	var initialResult map[string]any
	if strings.TrimSpace(p.BatchRootID) != "" {
		initialResult = map[string]any{
			"batch_root_id":        strings.TrimSpace(p.BatchRootID),
			"batch_root_parent_id": strings.TrimSpace(p.BatchRootParentID),
			"batch_root_owned":     p.BatchRootOwned,
		}
		if p.BatchTaskTotal > 0 {
			initialResult["batch_task_total"] = p.BatchTaskTotal
		}
	}
	st := &taskState{
		Task: Task{
			TaskID:            id,
			ClientTaskID:      p.ClientTaskID,
			BatchID:           strings.TrimSpace(p.BatchID),
			BatchName:         strings.TrimSpace(p.BatchName),
			AccountID:         p.AccountID,
			AccountName:       p.AccountName,
			DriverType:        p.DriverType,
			FileName:          name,
			SourceType:        sourceType,
			SourceAccountID:   p.SourceAccountID,
			SourceAccountName: p.SourceAccountName,
			SourceDriverType:  p.SourceDriverType,
			SourceFileID:      p.SourceFileID,
			RelPath:           p.RelPath,
			RelDir:            p.RelDir,
			TargetPath:        p.TargetPath,
			TargetDisplayPath: p.TargetDisplayPath,
			Status:            StatusPending,
			Phase:             phase,
			Message:           message,
			CleanupLocalMode:  cleanupLocalMode,
			CleanupLocalPath:  cleanupLocalPath,
			TotalBytes:        p.TotalBytes,
			QueueOrder:        order,
			CreatedAt:         timeutil.UnixFloat(now),
			UpdatedAt:         timeutil.UnixFloat(now),
			Result:            initialResult,
		},
		localPath:      localPath,
		conflictPolicy: p.ConflictPolicy,
		runDone:        make(chan struct{}),
	}
	return st
}

func (m *Manager) CreateServerLocalTask(ctx context.Context, p ServerLocalCreateParams) (*Task, error) {
	tasks, err := m.CreateServerLocalTasks(ctx, []ServerLocalCreateParams{p})
	if err != nil {
		return nil, err
	}
	return tasks[0], nil
}

func (m *Manager) CreateServerLocalTasks(ctx context.Context, params []ServerLocalCreateParams) ([]*Task, error) {
	if len(params) == 0 {
		return nil, domain.Errorf(domain.CodeValidation, "服务器上传任务不能为空")
	}
	result := make([]*Task, len(params))
	prepared := make([]CreateParams, 0, len(params))
	preparedIndexes := make([]int, 0, len(params))
	for i, p := range params {
		if strings.TrimSpace(p.LocalPath) == "" {
			return nil, domain.Errorf(domain.CodeValidation, "服务器上传缺少本地文件路径")
		}
		sourceType := strings.TrimSpace(p.SourceType)
		if sourceType == "" {
			sourceType = SourceTypeOfflineHandoff
		}
		if sourceType != SourceTypeOfflineHandoff && sourceType != SourceTypeServerLocal {
			return nil, domain.Errorf(domain.CodeValidation, "服务器上传来源类型不合法")
		}
		if p.ClientTaskID != "" {
			if existing := m.FindByClientTaskID(p.ClientTaskID); existing != nil {
				result[i] = existing
				continue
			}
		}
		info, err := os.Stat(p.LocalPath)
		if err != nil {
			return nil, domain.Wrap(domain.CodeNotFound, err)
		}
		if info.IsDir() {
			return nil, domain.Errorf(domain.CodeValidation, "离线交棒暂不支持目录，请提供文件路径")
		}
		size := p.TotalBytes
		if size <= 0 {
			size = info.Size()
		}
		prepared = append(prepared, CreateParams{
			ClientTaskID:      p.ClientTaskID,
			BatchID:           p.BatchID,
			BatchName:         p.BatchName,
			AccountID:         p.AccountID,
			AccountName:       p.AccountName,
			DriverType:        p.DriverType,
			FileName:          p.FileName,
			DisplayName:       p.DisplayName,
			SourceType:        sourceType,
			RelPath:           p.RelPath,
			RelDir:            p.RelDir,
			TargetPath:        p.TargetPath,
			TargetDisplayPath: p.TargetDisplayPath,
			LocalPath:         p.LocalPath,
			CleanupLocalMode:  p.CleanupLocalMode,
			CleanupLocalPath:  p.CleanupLocalPath,
			TotalBytes:        size,
			ConflictPolicy:    p.ConflictPolicy,
			Phase:             PhaseUploading,
		})
		preparedIndexes = append(preparedIndexes, i)
	}
	if len(prepared) == 0 {
		return result, nil
	}
	created, err := m.createBatch(ctx, prepared)
	if err != nil {
		return nil, err
	}
	for i, task := range created {
		result[preparedIndexes[i]] = task
	}
	return result, nil
}

func (m *Manager) findByClientTaskIDLocked(clientTaskID string) *taskState {
	clientTaskID = normalizeClientTaskID(clientTaskID)
	if clientTaskID == "" {
		return nil
	}
	if taskID, ok := m.clientTaskIndex[clientTaskID]; ok {
		if st, exists := m.tasks[taskID]; exists && normalizeClientTaskID(st.ClientTaskID) == clientTaskID {
			return st
		}
		delete(m.clientTaskIndex, clientTaskID)
	}
	for _, st := range m.tasks {
		if normalizeClientTaskID(st.ClientTaskID) == clientTaskID {
			if m.clientTaskIndex == nil {
				m.clientTaskIndex = make(map[string]string)
			}
			m.clientTaskIndex[clientTaskID] = st.TaskID
			return st
		}
	}
	return nil
}

func (m *Manager) FindByClientTaskID(clientTaskID string) *Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.findByClientTaskIDLocked(clientTaskID)
	if st == nil {
		return nil
	}
	return m.snapshot(st)
}

const offlineHandoffClientPrefix = "offline-handoff:"

func OfflineHandoffClientID(groupID string, index int) string {
	return fmt.Sprintf("%s%s:%d", offlineHandoffClientPrefix, strings.TrimSpace(groupID), index)
}

func (m *Manager) List(_ context.Context, accountID int64) []Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Task, 0, len(m.tasks))
	for _, st := range m.tasks {
		if accountID > 0 && st.AccountID != accountID {
			continue
		}
		out = append(out, *m.snapshot(st))
	}
	sortTasksDesc(out)
	return out
}

// ListFiltered 过滤/分页列表（排序语义与 List 一致：新→旧）。
func (m *Manager) ListFiltered(_ context.Context, accountID int64, f ListFilter) []Task {
	want := make(map[string]struct{}, len(f.Statuses))
	for _, s := range f.Statuses {
		s = strings.TrimSpace(s)
		if s != "" {
			want[s] = struct{}{}
		}
	}
	m.mu.Lock()
	out := make([]Task, 0, len(m.tasks))
	for _, st := range m.tasks {
		if accountID > 0 && st.AccountID != accountID {
			continue
		}
		if len(want) > 0 {
			if _, ok := want[st.Status]; !ok {
				continue
			}
		}
		out = append(out, *m.snapshot(st))
	}
	m.mu.Unlock()
	sortTasksDesc(out)
	if f.Offset > 0 {
		if f.Offset >= len(out) {
			return []Task{}
		}
		out = out[f.Offset:]
	}
	if f.Limit > 0 && f.Limit < len(out) {
		out = out[:f.Limit]
	}
	return out
}

// Summary 任务级汇总（各状态计数）。
func (m *Manager) Summary(_ context.Context, accountID int64) TaskSummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	summary := TaskSummary{Counts: make(map[string]int)}
	for _, st := range m.tasks {
		if accountID > 0 && st.AccountID != accountID {
			continue
		}
		summary.Total++
		summary.Counts[st.Status]++
	}
	return summary
}

// DefaultTaskWindow 默认窗口的“已完成”保留条数（列表与 SSE 快照共用）。
const DefaultTaskWindow = 500

// WindowTasks 默认窗口：全部非终态任务 + 最近 window 条终态（成功/跳过）任务。
// 避免历史成功记录把列表/快照撑到数 MB（7814 任务时 4.5MB → 约 1.1MB）。
func (m *Manager) WindowTasks(_ context.Context, accountID int64, window int) ([]Task, TaskSummary) {
	all := m.List(context.Background(), accountID)
	summary := TaskSummary{Counts: make(map[string]int), Total: len(all)}
	for _, t := range all {
		summary.Counts[t.Status]++
	}
	if window <= 0 {
		window = DefaultTaskWindow
	}
	active := make([]Task, 0, len(all))
	terminal := make([]Task, 0, window)
	for _, t := range all {
		switch t.Status {
		case StatusSuccess, StatusSkipped:
			terminal = append(terminal, t)
		default:
			active = append(active, t)
		}
	}
	if len(terminal) > window {
		terminal = terminal[:window]
	}
	return append(active, terminal...), summary
}

func (m *Manager) Get(_ context.Context, taskID string) (*Task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.tasks[taskID]
	if !ok {
		return nil, false
	}
	t := m.snapshot(st)
	return t, true
}

func (m *Manager) RemoveTasksByAccount(ctx context.Context, accountID int64) (int64, error) {
	if accountID <= 0 {
		return 0, nil
	}
	m.mu.Lock()
	ids := make([]string, 0)
	for id, st := range m.tasks {
		if st.AccountID == accountID {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()

	var removed int64
	for _, id := range ids {
		found, err := m.Delete(ctx, id, false)
		if err != nil {
			return removed, err
		}
		if found {
			removed++
		}
	}
	return removed, nil
}
