package buildinfo

// Version 是 LitePan 自己的版本号（**全站唯一真值**）。
//
// 它可在构建时通过 -ldflags "-X litepan/internal/buildinfo.Version=..." 覆盖；
// 前端不在源码里保存版本号，而是经 GET /api/public/system-config 运行期读取本值
// （见 web/src/stores/appInfo.ts）。发版时改这一处即可。
var Version = "v0.0.44"
