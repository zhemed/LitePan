package playback

import (
	"errors"
	"testing"
)

// parseRanges 的表驱动用例：锁定"多段 / 忽略 Range / 全越界"三态语义。
func TestParseRangesThreeStates(t *testing.T) {
	const size = 100

	cases := []struct {
		name          string
		header        string
		want          []byteRange
		ignore        bool // 期望 (nil, nil)：忽略 Range 走全量
		wantNoOverlap bool // 期望 errRangeNoOverlap（段全部越界）
		wantAnyErr    bool // 期望任意错误（语法/单位错误）
	}{
		{name: "单段", header: "bytes=0-9", want: []byteRange{{0, 9}}},
		{name: "单段开区间", header: "bytes=90-", want: []byteRange{{90, 99}}},
		{name: "后缀段", header: "bytes=-10", want: []byteRange{{90, 99}}},
		{name: "两段", header: "bytes=0-9,20-29", want: []byteRange{{0, 9}, {20, 29}}},
		{name: "段内空格", header: "bytes= 0-9 , 20-29 ", want: []byteRange{{0, 9}, {20, 29}}},
		{name: "尾部逗号", header: "bytes=0-9,", want: []byteRange{{0, 9}}},
		{name: "空段跳过", header: "bytes=0-9,,20-29", want: []byteRange{{0, 9}, {20, 29}}},
		{name: "大小写不敏感", header: "Bytes=0-9", want: []byteRange{{0, 9}}},
		{name: "起点越界（唯一段）", header: "bytes=200-299", wantNoOverlap: true},
		{name: "多段全越界", header: "bytes=100-199,300-399", wantNoOverlap: true},
		{name: "分段过多忽略 Range（逗号数 ≥32）", header: "bytes=" + repeatRange(33), ignore: true},
		{name: "累计长度超文件忽略 Range", header: "bytes=0-59,60-99,0-59", ignore: true},
		{name: "非 bytes 单位报错", header: "items=0-9", wantAnyErr: true},
		{name: "语法错误报错", header: "bytes=abc", wantAnyErr: true},
		{name: "end 小于 start 报错", header: "bytes=9-0", wantAnyErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRanges(tc.header, size)
			switch {
			case tc.ignore:
				if err != nil || got != nil {
					t.Fatalf("应忽略 Range：(ranges=%v, err=%v)", got, err)
				}
			case tc.wantNoOverlap:
				if !errors.Is(err, errRangeNoOverlap) {
					t.Fatalf("应返回 errRangeNoOverlap，实际 %v", err)
				}
			case tc.wantAnyErr:
				if err == nil {
					t.Fatalf("应报错，实际 ranges=%v", got)
				}
			default:
				if err != nil {
					t.Fatalf("不应报错：%v", err)
				}
				if len(got) != len(tc.want) {
					t.Fatalf("段数=%d，期望 %d（%v）", len(got), len(tc.want), got)
				}
				for i := range got {
					if got[i] != tc.want[i] {
						t.Fatalf("第 %d 段=%v，期望 %v", i, got[i], tc.want[i])
					}
				}
			}
		})
	}
}

func TestByteRangeHelpers(t *testing.T) {
	ra := byteRange{start: 10, end: 19}
	if got := ra.length(); got != 10 {
		t.Fatalf("length=%d，期望 10", got)
	}
	if got := ra.contentRange(100); got != "bytes 10-19/100" {
		t.Fatalf("contentRange=%q", got)
	}
}

func repeatRange(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			out += ","
		}
		out += "0-0"
	}
	return out
}
