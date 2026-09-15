# Obsidian Chat

Go + Vue 在线聊天。桌面默认左侧联系人、右侧消息；好友管理、群聊详情和账号设置以第三列展开。布局和黑白灰主题参考 Obsidian Arc。支持深色模式和移动端。

## 已实现

- 首次管理员初始化、注册、登录、退出、昵称和密码修改。
- 用户搜索、双向好友申请、接受/拒绝/撤回、解除好友、私聊。
- 群聊创建、邀请好友、移除成员、退群、群主转让、归档和恢复。
- 持久化消息、历史分页、未读数、多设备已读同步、在线状态、断线恢复、发送重试去重。
- 管理后台：用户角色和停用、群聊归档、注册开关、操作日志、运行指标。
- 消息右键菜单：撤回自己的消息、引用回复、复制、转发到其他会话；移动端可点击消息旁的操作按钮。
- 图片与文件发送、个人头像；管理员分别设置头像、图片和文件大小上限。
- 外观设置：气泡留白、圆角、字号、最大宽度，背景主题和上传壁纸，控件透明度／模糊、背景模糊／压暗；设置保存在当前浏览器。
- 管理员配置 Cloudflare Turnstile，用于注册、登录、好友申请、建群及超阈值消息验证；好友申请和消息频率按账号在服务端限制。
- 管理员聊天记录查询，支持用户、会话、内容筛选与分页，每次读取写入操作日志。

## 运行

需要 Go 1.27、Node.js 22.12+。前端只在构建时使用 Node，运行时不需要。

```powershell
cd E:\Project\ObisidianChat\web
npm ci
npm run build
cd ..
go build -trimpath -ldflags="-s -w" -o bin/obsidianchat.exe ./cmd/obsidianchat
.\bin\obsidianchat.exe
```

打开 <http://127.0.0.1:8090>。首次启动会在终端输出 `setup_token`，填入网页初始化表单并创建管理员。没有预置账号或默认密码。后续用户可自行注册，管理员可关闭注册。

Linux 构建时把输出名改为 `bin/obsidianchat`。二进制已内嵌网页，部署只需程序和数据目录。

开发时先 `go run ./cmd/obsidianchat`，再在 `web` 目录运行 `npm run dev`。Vite 的 `/api` 代理默认指向 `127.0.0.1:8090`。

| 环境变量 | 默认值 | 作用 |
| --- | --- | --- |
| `OC_ADDR` | `127.0.0.1:8090` | HTTP 监听地址 |
| `OC_DATABASE` | `data/chat.db` | SQLite 数据文件 |
| `OC_SETUP_TOKEN` | 每次未初始化启动随机生成 | 首次管理员初始化令牌 |
| `OC_ORIGIN` | 请求自身协议和 Host | 反向代理后的公开来源，如 `https://chat.example.com` |
| `OC_SECURE_COOKIE` | `false` | HTTPS 部署时设为 `true` |

## 检查

```powershell
cd web
npm ci
npm test
npm run build
cd ..
go test ./...
go vet ./...
```

Linux CI 还执行 `go test -race ./...` 和 Docker 镜像构建/健康检查。本机没有 C 编译器，因此未运行 race 检查或 Docker；CI 配置已提供，尚未提交远端执行。

连接测试仅使用本机临时数据库和本机测试服务：

```powershell
$env:OC_LOADTEST = '1'
go test ./internal/chat -run TestLoad1000Connections -v -count=1
Remove-Item Env:OC_LOADTEST
go test ./internal/chat -run '^$' -bench BenchmarkHubFanout100 -benchmem
```

## 部署与边界

已部署实例和维护信息见 [生产部署记录](deploy/PRODUCTION.md)。

见 [部署说明](docs/DEPLOYMENT.md) 和 [架构与实测](docs/ARCHITECTURE.md)。当前面向单实例；SQLite 写入串行执行，实时连接存在进程内。不要对同一个数据库启动多个应用实例或通过负载均衡随机分流长连接。

支持文字、图片和文件消息，以及浏览器打开期间的后台新消息系统通知。通知未开启时，每次打开网站会先通过站内窗口询问，只有用户选择“是”才请求浏览器权限，也可选择“不再提醒”或在账号设置中修改；不包含语音/视频、端到端加密、浏览器关闭后的离线 Push、邮件通知。消息以明文存储在本机数据库，管理员可查看聊天记录。撤回后原文字替换为撤回提示，已经转发的副本不会同步删除。新群成员可查看该群既有历史，退出或被移除后失去会话附件访问权限（本人上传的文件仍可访问）。解除好友会归档原私聊，重新成为好友时恢复历史。

## 验证、上传与反垃圾设置

管理员进入「管理后台 → 功能设置」。Turnstile 默认关闭；填入本站的 Site key 与 Secret key 后，分别开启注册、登录、添加好友、创建群聊的验证。服务端通过 Cloudflare Siteverify 检查令牌、操作名及域名，密钥只保存在服务器数据库中，不返回浏览器；更换时输入新密钥，留空则保留旧值。配置说明见 [Cloudflare 官方文档](https://developers.cloudflare.com/turnstile/get-started/)。

默认头像 2 MiB、图片 10 MiB、文件 25 MiB，可分别调至最高 10／30／100 MiB。图片和头像支持 PNG、JPEG、GIF，最多 4000 万像素；其他类型以附件下载。上传文件保存在数据库同目录的 `uploads` 子目录，备份时需包含它。上传不会自动发送，选好后点击发送，可附带文字或引用。

默认每人每分钟最多 10 次好友申请、120 条消息；创建群聊每分钟最多 10 次。消息验证阈值可按最近一分钟和最近一小时分别配置，0 表示关闭；达到任一阈值后每条新消息需要验证，直到对应时间窗口的消息数下降。消息上限独立生效，通过验证也不能超限。频率记录保存在 SQLite，重启不会清零有效时间窗口。

群聊最多 256 人，每个用户最多 200 个会话、100 条待处理好友申请。每个账号最多 4 条实时连接、10 个登录会话；全局实时连接上限为 10,000，这是资源保护值，**不是实测承载保证**。历史每页 50 条，前端最多保留 200 条，可继续向前翻页。
