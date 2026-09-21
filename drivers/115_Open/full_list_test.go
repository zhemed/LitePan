package pan115open

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"litepan/internal/domain"
)

// emptyPage 是分页结束信号：真实接口在越界 offset 上返回 data 为空的页。
func emptyPage() listPageResp { return listPageResp{} }

func TestCollectFullListPagesContinuesAfterShortPageAndLowCount(t *testing.T) {
	offsets := make([]int, 0, 4)
	pages := []listPageResp{
		{Count: 1, Data: []fileEntry{{Fid: "f1", Fn: "1.mkv", Pid: "d1"}, {Fid: "f2", Fn: "2.mkv", Pid: "d1"}}},
		{Count: 2, Data: []fileEntry{{Fid: "f3", Fn: "3.mkv", Pid: "d2"}}},
		emptyPage(),
		emptyPage(),
	}
	entries, err := collectFullListPages(context.Background(), func(_ context.Context, offset, _ int) (listPageResp, error) {
		offsets = append(offsets, offset)
		page := pages[0]
		pages = pages[1:]
		return page, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("完整清单条目数 = %d，期望 3", len(entries))
	}
	if !reflect.DeepEqual(offsets, []int{0, 2, 3, 3}) {
		t.Fatalf("分页 offset = %v，期望 [0 2 3 3]", offsets)
	}
}

func TestCollectFullListPagesRejectsRepeatedPage(t *testing.T) {
	page := listPageResp{Count: 99, Data: []fileEntry{{Fid: "f1", Fn: "1.mkv", Pid: "d1"}}}
	calls := 0
	_, err := collectFullListPages(context.Background(), func(_ context.Context, _, _ int) (listPageResp, error) {
		calls++
		return page, nil
	})
	if err == nil {
		t.Fatal("重复分页必须中止，不能把不完整清单交给上层")
	}
	if calls != 2 {
		t.Fatalf("重复分页调用次数 = %d，期望 2", calls)
	}
}

func TestCollectFullListPagesDeduplicatesOverlappingEntries(t *testing.T) {
	pages := []listPageResp{
		{Data: []fileEntry{{Fid: "f1", Fn: "1.mkv", Pid: "d1"}, {Fid: "f2", Fn: "2.mkv", Pid: "d1"}}},
		{Data: []fileEntry{{Fid: "f2", Fn: "2.mkv", Pid: "d1"}, {Fid: "f3", Fn: "3.mkv", Pid: "d2"}}},
		emptyPage(),
		emptyPage(),
	}
	entries, err := collectFullListPages(context.Background(), func(_ context.Context, _, _ int) (listPageResp, error) {
		page := pages[0]
		pages = pages[1:]
		return page, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("重叠分页去重后条目数 = %d，期望 3", len(entries))
	}
}

// 单次空页可能只是厂商缓存抖动，必须重试确认，不能当成“已经读完”。
func TestCollectFullListPagesRetriesEmptyPageBeforeStopping(t *testing.T) {
	offsets := make([]int, 0, 5)
	pages := []listPageResp{
		{Count: 2, Data: []fileEntry{{Fid: "f1", Fn: "1.mkv", Pid: "d1"}}},
		emptyPage(),
		{Count: 2, Data: []fileEntry{{Fid: "f2", Fn: "2.mkv", Pid: "d1"}}},
		emptyPage(),
		emptyPage(),
	}
	entries, err := collectFullListPages(context.Background(), func(_ context.Context, offset, _ int) (listPageResp, error) {
		offsets = append(offsets, offset)
		page := pages[0]
		pages = pages[1:]
		return page, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("抖动恢复后条目数 = %d，期望 2（空页不得提前结束扫描）", len(entries))
	}
	if !reflect.DeepEqual(offsets, []int{0, 1, 1, 2, 2}) {
		t.Fatalf("分页 offset = %v，期望 [0 1 1 2 2]", offsets)
	}
}

// 服务端声明了更多条目却只拿到更少的，说明清单不完整：必须报错，不得静默返回残缺结果。
func TestCollectFullListPagesRejectsIncompleteListAgainstCount(t *testing.T) {
	pages := []listPageResp{
		{Count: 5, Data: []fileEntry{{Fid: "f1", Fn: "1.mkv", Pid: "d1"}}},
		emptyPage(),
		emptyPage(),
	}
	entries, err := collectFullListPages(context.Background(), func(_ context.Context, _, _ int) (listPageResp, error) {
		page := pages[0]
		pages = pages[1:]
		return page, nil
	})
	if err == nil {
		t.Fatal("清单数量少于服务端声明数量时必须报错")
	}
	if entries != nil {
		t.Fatalf("报错时不得返回部分清单，实际返回 %d 条", len(entries))
	}
	if !strings.Contains(err.Error(), "应有 5 个文件，实际只获取 1 个") {
		t.Fatalf("错误信息未说明数量差异：%v", err)
	}
	if appErr, ok := domain.AsAppError(err); !ok || appErr.Code != domain.CodeDriverError {
		t.Fatalf("错误码不是驱动错误：%v", err)
	}
}

// 空页重试等待必须可被 context 取消打断，否则上层取消后仍会白等一拍。
func TestCollectFullListPagesAbortsEmptyRetryOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	_, err := collectFullListPages(ctx, func(_ context.Context, _, _ int) (listPageResp, error) {
		calls++
		if calls == 1 {
			return listPageResp{Count: 1, Data: []fileEntry{{Fid: "f1", Fn: "1.mkv", Pid: "d1"}}}, nil
		}
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		return emptyPage(), nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消后应返回 context.Canceled，实际 %v", err)
	}
	if calls != 2 {
		t.Fatalf("取消后不得继续拉取，调用次数 = %d，期望 2", calls)
	}
}

func TestBuildDirPathTruncatesAtMountRoot(t *testing.T) {
	paths := []dirPathEntry{
		{FileID: flexString("mount"), FileName: "账号挂载点"},
		{FileID: flexString("root"), FileName: "挂载根"},
		{FileID: flexString("sub"), FileName: "子目录"},
	}
	path, ok := buildDirPath(paths, "叶子.mp4", "root")
	if !ok {
		t.Fatal("根内的目录必须能拼出相对路径")
	}
	if path != "子目录/叶子.mp4" {
		t.Fatalf("相对挂载根的路径 = %q，期望 %q", path, "子目录/叶子.mp4")
	}
}

func TestBuildDirPathRejectsDirOutsideMountRoot(t *testing.T) {
	paths := []dirPathEntry{
		{FileID: flexString("other"), FileName: "别的挂载点"},
		{FileID: flexString("sub"), FileName: "子目录"},
	}
	path, ok := buildDirPath(paths, "叶子.mp4", "root")
	if ok {
		t.Fatalf("不在挂载根下的目录必须返回 ok=false，实际路径 %q", path)
	}
	if path != "" {
		t.Fatalf("拒绝时应返回空路径，实际 %q", path)
	}
}

func TestBuildDirPathSkipsZeroSegmentAndEmptyNames(t *testing.T) {
	paths := []dirPathEntry{
		{FileID: flexString("0"), FileName: ""},
		{FileID: flexString("root"), FileName: "挂载根"},
		{FileID: flexString("sub"), FileName: "   "},
	}
	path, ok := buildDirPath(paths, "叶子.mp4", "root")
	if !ok {
		t.Fatal("根内的目录必须能拼出相对路径")
	}
	if path != "叶子.mp4" {
		t.Fatalf("空名与 file_id=0 段应跳过，实际 %q", path)
	}
}

func TestBuildDirPathAcceptsZeroRoot(t *testing.T) {
	paths := []dirPathEntry{
		{FileID: flexString("top"), FileName: "顶层目录"},
		{FileID: flexString("sub"), FileName: "子目录"},
	}
	path, ok := buildDirPath(paths, "叶子.mp4", "0")
	if !ok {
		t.Fatal("rootID 为 0 时任意目录都在根下")
	}
	if path != "顶层目录/子目录/叶子.mp4" {
		t.Fatalf("路径 = %q，期望 %q", path, "顶层目录/子目录/叶子.mp4")
	}
}

// 目录名里的分隔符不能变成层级：否则上层会把路径用到错误的目录上。
func TestBuildDirPathKeepsSeparatorsInsideSegment(t *testing.T) {
	paths := []dirPathEntry{
		{FileID: flexString("root"), FileName: "挂载根"},
		{FileID: flexString("sub"), FileName: "a/b"},
	}
	path, ok := buildDirPath(paths, `c\d`, "root")
	if !ok {
		t.Fatal("根内的目录必须能拼出相对路径")
	}
	if path != "a_b/c_d" {
		t.Fatalf("路径 = %q，期望 %q", path, "a_b/c_d")
	}
	if strings.Count(path, "/") != 1 {
		t.Fatalf("目录名内的分隔符不得产生层级：%q", path)
	}
}

func TestPathSegmentName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"去首尾空白", "  剧集  ", "剧集"},
		{"斜杠替换", "第1季/第2季", "第1季_第2季"},
		{"反斜杠替换", `第1季\第2季`, "第1季_第2季"},
		{"混用分隔符", `a/b\c`, "a_b_c"},
		{"纯空白视为空", "   ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathSegmentName(tc.in); got != tc.want {
				t.Fatalf("pathSegmentName(%q) = %q，期望 %q", tc.in, got, tc.want)
			}
		})
	}
}
