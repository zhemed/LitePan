package httpx

import "litepan/internal/buildinfo"

// AppName 是程序品牌名。
const AppName = "LitePan"

// AppVersion / DefaultUserAgent 由**全站唯一真值** buildinfo.Version 派生，不再各自持有字面量。
//
// 历史上这里有一份上游版本号的副本，并注释"保持与前端 version.ts 一致" —— 靠人工保持的一致性
// 必然漂移（前端那份已删除，改由 /api/public/system-config 运行期读取）。
// 注意：必须是 var 而非 const —— buildinfo.Version 需要可被 `-ldflags -X` 覆盖。
// 常量性由调用方承担（drivers/115_Open/upload.go 的 ossUserAgent 已随之改为 var）。
var (
	AppVersion       = buildinfo.Version
	DefaultUserAgent = AppName + "/" + AppVersion
)
