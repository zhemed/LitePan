package adminauth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"litepan/internal/domain"
)

// fakeConfigRepo 记录调用次数，用于观察缓存是否真的省掉了读库，以及读写失败时的回退行为。
type fakeConfigRepo struct {
	mu       sync.Mutex
	data     map[string]string
	allCalls int
	getCalls int
	allErr   error
	setErr   error
}

func newFakeConfigRepo(values map[string]string) *fakeConfigRepo {
	repo := &fakeConfigRepo{data: make(map[string]string, len(values))}
	for k, v := range values {
		repo.data[k] = v
	}
	return repo
}

func (r *fakeConfigRepo) Get(_ context.Context, key string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getCalls++
	value, ok := r.data[key]
	return value, ok, nil
}

func (r *fakeConfigRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.setErr != nil {
		return r.setErr
	}
	r.data[key] = value
	return nil
}

func (r *fakeConfigRepo) All(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.allCalls++
	if r.allErr != nil {
		return nil, r.allErr
	}
	out := make(map[string]string, len(r.data))
	for k, v := range r.data {
		out[k] = v
	}
	return out, nil
}

func (r *fakeConfigRepo) snapshot() (allCalls, getCalls int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.allCalls, r.getCalls
}

func (r *fakeConfigRepo) put(values map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range values {
		r.data[k] = v
	}
}

func newCacheTestService(repo domain.ConfigRepository) *Service {
	return New(repo, []byte("test-secret-key-min-16b"), nil)
}

// blockingSetRepo 让第一次 Set 在"已落库、尚未返回"时停住，用于复现两个并发写的交错。
// 用调用计数（而不是 sync.Once）区分两次写入：Once 会把后一次挡在门外，等价于已经串行化。
type blockingSetRepo struct {
	*fakeConfigRepo
	entered chan struct{}
	release chan struct{}
	callMu  sync.Mutex
	calls   int
}

func newBlockingSetRepo(values map[string]string) *blockingSetRepo {
	return &blockingSetRepo{
		fakeConfigRepo: newFakeConfigRepo(values),
		entered:        make(chan struct{}),
		release:        make(chan struct{}),
	}
}

func (r *blockingSetRepo) Set(ctx context.Context, key, value string) error {
	r.callMu.Lock()
	r.calls++
	call := r.calls
	r.callMu.Unlock()
	if err := r.fakeConfigRepo.Set(ctx, key, value); err != nil {
		return err
	}
	if call == 1 {
		close(r.entered)
		<-r.release
	}
	return nil
}

// 并发写同一独占键后，缓存必须与库里最终值一致。
// 反例（先写库再取锁）：A 落库 → B 落库并更新缓存 → A 更新缓存，缓存会永久停在旧值。
func TestConcurrentSetConfigKeepsCacheConsistentWithDatabase(t *testing.T) {
	repo := newBlockingSetRepo(map[string]string{KeyPublicIndexEnabled: "false"})
	svc := newCacheTestService(repo)
	ctx := context.Background()
	if svc.publicIndexEnabled(ctx) {
		t.Fatal("初始应为关闭")
	}

	aDone := make(chan struct{})
	go func() {
		_ = svc.setConfig(ctx, KeyPublicIndexEnabled, "false")
		close(aDone)
	}()
	<-repo.entered

	bDone := make(chan struct{})
	go func() {
		_ = svc.setConfig(ctx, KeyPublicIndexEnabled, "true")
		close(bDone)
	}()
	// 修复实现下 B 必须等 A 释放锁，这里只等一个短窗口即可；
	// 旧实现下 B 会抢先落库并更新缓存（这正是要复现的交错）。
	select {
	case <-bDone:
	case <-time.After(200 * time.Millisecond):
	}
	close(repo.release)
	select {
	case <-aDone:
	case <-time.After(2 * time.Second):
		t.Fatal("写入 A 未完成")
	}
	select {
	case <-bDone:
	case <-time.After(2 * time.Second):
		t.Fatal("写入 B 未完成")
	}

	stored, ok, err := repo.Get(ctx, KeyPublicIndexEnabled)
	if err != nil || !ok {
		t.Fatalf("读库失败：ok=%v err=%v", ok, err)
	}
	if got := svc.publicIndexEnabled(ctx); got != (stored == "true") {
		t.Fatalf("缓存 = %v，库里是 %q：并发写后缓存与库不一致", got, stored)
	}
}

// 独占键的读取只应触发一次全量加载，之后的读取都命中缓存。
func TestConfigCacheServesOwnedKeysWithoutRepeatedReads(t *testing.T) {
	repo := newFakeConfigRepo(map[string]string{KeyPublicIndexEnabled: "true"})
	svc := newCacheTestService(repo)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if !svc.publicIndexEnabled(ctx) {
			t.Fatalf("第 %d 次读取：public_index_enabled 应为 true", i+1)
		}
	}
	if all, gets := repo.snapshot(); all != 1 || gets != 0 {
		t.Fatalf("独占键读取应只触发一次 All 且不直读：All=%d Get=%d", all, gets)
	}
}

// 本服务的写入必须同步到内存快照，否则改完密码/开关会读到旧值。
func TestSetConfigUpdatesCacheAfterWrite(t *testing.T) {
	repo := newFakeConfigRepo(nil)
	svc := newCacheTestService(repo)
	ctx := context.Background()

	if svc.publicIndexEnabled(ctx) {
		t.Fatal("初始应关闭匿名访问")
	}
	if err := svc.setConfig(ctx, KeyPublicIndexEnabled, "true"); err != nil {
		t.Fatalf("setConfig: %v", err)
	}
	if !svc.publicIndexEnabled(ctx) {
		t.Fatal("写入后应立刻读到新值，不能等到下次全量加载")
	}
	if all, _ := repo.snapshot(); all != 1 {
		t.Fatalf("写入后更新内存快照，不应额外触发全量加载：All=%d", all)
	}
	if stored, ok, _ := repo.Get(ctx, KeyPublicIndexEnabled); !ok || stored != "true" {
		t.Fatalf("写库结果 = %q(ok=%v)，期望 true", stored, ok)
	}
}

// 缓存未加载时的写入不应顺手把缓存标记为已加载，避免缓存里只有刚写的那一个键。
func TestSetConfigWithoutWarmedCacheDoesNotMarkLoaded(t *testing.T) {
	repo := newFakeConfigRepo(map[string]string{KeySessionTimeout: "1800"})
	svc := newCacheTestService(repo)
	ctx := context.Background()

	if err := svc.setConfig(ctx, KeyCompactHomeEnabled, "true"); err != nil {
		t.Fatalf("setConfig: %v", err)
	}
	if !svc.compactHomeEnabled(ctx) {
		t.Fatal("刚写入的键应可读")
	}
	if all, _ := repo.snapshot(); all != 1 {
		t.Fatalf("首次读取应触发一次全量加载：All=%d", all)
	}
	if svc.sessionTimeout(ctx) != 1800 {
		t.Fatal("全量加载必须读回其它独占键，而不是只保留写入过的键")
	}
}

// 别的服务写同一张 configs 表的键不得进缓存，否则设置页改完这里会一直读旧值。
func TestConfigCacheExcludesKeysOwnedByOtherServices(t *testing.T) {
	repo := newFakeConfigRepo(map[string]string{"auth_active_refresh_enabled": "true"})
	svc := newCacheTestService(repo)
	ctx := context.Background()

	if !svc.configBool(ctx, "auth_active_refresh_enabled", true) {
		t.Fatal("初始应为 true")
	}
	// 模拟设置服务直接写库（不经本服务）。
	repo.put(map[string]string{"auth_active_refresh_enabled": "false"})
	if svc.configBool(ctx, "auth_active_refresh_enabled", true) {
		t.Fatal("非独占键必须直读，内存设置后应立刻读到 false")
	}
	if all, gets := repo.snapshot(); all != 0 || gets != 2 {
		t.Fatalf("非独占键不应触发全量加载：All=%d Get=%d，期望 0/2", all, gets)
	}
}

// All() 失败时必须本次直读、且不置位，下次访问重试加载。
func TestConfigCacheLoadFailureFallsBackAndRetries(t *testing.T) {
	repo := newFakeConfigRepo(map[string]string{KeyPublicIndexEnabled: "true"})
	repo.mu.Lock()
	repo.allErr = errors.New("db unavailable")
	repo.mu.Unlock()
	svc := newCacheTestService(repo)
	ctx := context.Background()

	if !svc.publicIndexEnabled(ctx) {
		t.Fatal("全量加载失败时应回退直读，而不是返回默认值")
	}
	if all, gets := repo.snapshot(); all != 1 || gets != 1 {
		t.Fatalf("首次失败路径应 All=1 Get=1，实际 All=%d Get=%d", all, gets)
	}

	// 恢复后下一次读取必须重试加载（失败不得置位），并命中缓存。
	repo.mu.Lock()
	repo.allErr = nil
	repo.mu.Unlock()
	if !svc.publicIndexEnabled(ctx) {
		t.Fatal("恢复后应读到 true")
	}
	if all, gets := repo.snapshot(); all != 2 || gets != 1 {
		t.Fatalf("恢复后应重试加载并命中缓存：All=%d Get=%d，期望 2/1", all, gets)
	}
	if !svc.publicIndexEnabled(ctx) {
		t.Fatal("第三次读取应命中缓存")
	}
	if all, gets := repo.snapshot(); all != 2 || gets != 1 {
		t.Fatalf("命中缓存不应再读库：All=%d Get=%d", all, gets)
	}
}

// 写库失败时不得更新缓存，否则内存与库会长期不一致。
func TestSetConfigFailureKeepsCacheUnchanged(t *testing.T) {
	repo := newFakeConfigRepo(map[string]string{KeyPublicIndexEnabled: "false"})
	svc := newCacheTestService(repo)
	ctx := context.Background()

	if svc.publicIndexEnabled(ctx) {
		t.Fatal("初始应关闭")
	}
	repo.mu.Lock()
	repo.setErr = errors.New("write failed")
	repo.mu.Unlock()
	if err := svc.setConfig(ctx, KeyPublicIndexEnabled, "true"); err == nil {
		t.Fatal("写库失败必须返回错误")
	}
	if svc.publicIndexEnabled(ctx) {
		t.Fatal("写库失败后缓存不得变成新值")
	}
}

// 登录状态判定依赖多把独占键，缓存必须让它们在同一次加载里保持一致快照。
func TestConfigCacheKeepsCredentialSnapshot(t *testing.T) {
	repo := newFakeConfigRepo(map[string]string{
		KeyAdminUsername:  "admin",
		KeyAdminPassword:  "hashed-secret",
		KeySessionTimeout: "900",
	})
	svc := newCacheTestService(repo)
	ctx := context.Background()

	username, password := svc.adminCredentials(ctx)
	if username != "admin" || password != "hashed-secret" {
		t.Fatalf("凭据 = %q/%q，期望 admin/hashed-secret", username, password)
	}
	if svc.sessionTimeout(ctx) != 900 {
		t.Fatalf("session_timeout = %d，期望 900", svc.sessionTimeout(ctx))
	}
	if all, gets := repo.snapshot(); all != 1 || gets != 0 {
		t.Fatalf("整轮凭据判定应只加载一次：All=%d Get=%d", all, gets)
	}
}
