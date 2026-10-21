# 绘本书架 EhonShelf

一个原生桌面书架阅读器，用于离线浏览国際日本文化研究センター（日文研）公开的
「艶本資料データベース」（https://lapis.nichibun.ac.jp/enp/ ，约 470 部作品）。

从 v0.2.0 起用 Go 和 [MyGo](https://github.com/egoist/mygo) 的原生界面（package `ui`，GPU 绘制，不使用网页视图）重写，
单个可执行文件，启动快、占用小。旧的 Python 版本保留在
[`python-legacy`](https://github.com/mcxen/ehon-shelf/tree/python-legacy) 分支。

## 功能

- 书架：封面由每部作品第一页生成（`cover.jpg`），显示书名、绘师、年代、图数；搜索书名 / 绘师，按绘师筛选；网格按需绘制，作品多也流畅
- 阅读器：默认右至左阅读，可切换左至右；单页 / 双页；方向键翻页，`+` / `-` 缩放，`Esc` 返回；适应窗口；跳转到指定页；记住每部作品读到的位置
- 设置：远程完整目录（约 470 部），勾选或全部下载；当前作品与总体进度；暂停 / 继续；用系统对话框选择数据文件夹；请求间隔（默认 1.2 秒，最小 1 秒）；下载示例数据
- 原生菜单：文件（书架、设置、重新读取书库、退出）、编辑、阅读、窗口

## 数据来源与版权

所有图像与书目数据来自 **国際日本文化研究センター「艶本資料データベース」**，版权及相关权利归其所有。
本仓库只包含程序代码（MIT 许可证），**不包含也不授权任何图像数据**。下载的数据仅供个人学习研究使用，
如需转载、出版或其他用途，请遵循日文研的利用规定并取得许可。

## 安装

到 [Releases](https://github.com/mcxen/ehon-shelf/releases/latest) 下载：

| 平台 | 文件 |
|---|---|
| macOS（Apple Silicon 与 Intel 通用） | `EhonShelf-*-universal.dmg`，或 `.app.zip` |
| Windows x64 | `EhonShelf Setup *.exe` 安装程序，或免安装的 `*-windows-amd64-portable.zip` |
| Linux x64 | `ehonshelf_*_amd64.deb`（`sudo apt install ./ehonshelf_*.deb`），或 `ehonshelf-*-linux-amd64.tar.gz` 配合 `install.sh`（`sh install.sh ehonshelf-*.tar.gz`） |

Linux 需要 GTK 3（多数桌面发行版已自带）。

**未签名提示**：安装包没有经过 Apple / Microsoft 签名。
macOS 首次打开时请在 Finder 中右键点击应用选择「打开」，或执行
`xattr -dr com.apple.quarantine /Applications/EhonShelf.app`；
Windows 出现 SmartScreen 提示时选择「更多信息」→「仍要运行」。

## 获取数据

1. 打开「设置」（书架右上角，或菜单 文件 → 设置，`Ctrl/Cmd + ,`）。
2. 点击「下载示例数据」，从最新发布版获取 3 部示例作品（约 190 MB）。
3. 在「远程目录」中勾选作品后点「下载所选」，或点「下载全部」。
4. 下载可随时「暂停」/「继续」。下载队列保存在数据文件夹的 `queue.json` 中，关闭程序后重新打开可继续；已下载的页面会跳过。

默认数据文件夹为 `~/EhonShelf/data`，配置（数据文件夹、请求间隔、阅读位置）保存在 `~/EhonShelf/config.json`。
数据格式与 Python 版相同（`<编号>/meta.json`、`pages/0001.jpg`…、`cover.jpg`），旧数据可直接使用。
若可执行文件旁有 `sample-data.zip`，首次运行且书库为空时会自动导入。

## 浏览器用户脚本导出（应用无法直接访问网站时）

如果网站的防火墙拒绝应用的请求，可以在自己能正常浏览该网站的浏览器里用用户脚本导出数据，再导入应用。

1. 安装浏览器扩展 [Tampermonkey](https://www.tampermonkey.net/)（Chrome / Edge / Firefox / Safari 均可；Firefox 也可用 Greasemonkey）。
2. 打开脚本地址，Tampermonkey 会提示安装：
   https://raw.githubusercontent.com/mcxen/ehon-shelf/main/userscript/ehonshelf-export.user.js
3. 访问 https://lapis.nichibun.ac.jp/enp/ ，页面右下角会出现「EhonShelf 数据导出」面板：
   - 作品编号：如 `1-50,81`，或填 `all` 导出全部；
   - 每个数据包作品数：每导出这么多部作品就下载一个 `ehonshelf-<起>-<止>.zip`（默认 5，避免占用过多内存）；
   - 点击「开始」；「暂停」后再点「开始」即可继续，已导出的作品记录在浏览器中会自动跳过（可用「清除已导出记录」重置）。
   脚本每秒最多发出 1 个同源请求；遇到 403 / 429 会立即停止，并先把已完成的作品打包下载。
4. 在应用中打开「设置」→「导入本地数据包…」，选择下载的 zip（可多选），已有文件不会被覆盖。

## 礼貌抓取说明

- 所有请求串行发送，默认间隔 1.2 秒，最小 1 秒，不可设置更低
- 读取并遵守 `robots.txt`；网络错误按指数退避重试；`.part` 临时文件写完再改名，中断不会留下残缺页面
- 原图从 PDF 中原样提取内嵌 JPEG，不重新压缩；没有原图时退回缩略图并在 `meta.json` 中标记 `lowres`
- 服务器返回 403 / 429（拒绝或限流）时会大幅放慢，仍被拒绝则暂停整个队列，请过一段时间再继续
- 使用固定的 User-Agent（`NichibunEnpArchiver/1.0 (EhonShelf/0.2; …)`）标识本程序
- 全部作品数据量很大（数十 GB 级），请避免短时间内反复下载

## 开发

需要 Go 1.27+（Linux 运行还需要 GTK 3）：

```bash
go tool mygo dev      # 开发运行，改代码自动重启
go test ./...         # 测试界面（无需窗口）与抓取逻辑（本地假站点）
go tool mygo build    # 打包当前平台，输出在 build/
go tool mygo build -platform darwin/universal,windows/amd64,linux/amd64
```

推送 `v*` 标签（与 `mygo.json` 的 version 一致）会触发 GitHub Actions：在 macOS 与 Ubuntu 上构建
macOS `.app` / `.dmg`、Windows `.exe` 安装程序、Linux `.tar.gz` / `.deb`，发布到 Release，并沿用之前发布版中的 `sample-data.zip`。

## 许可证

代码采用 [MIT License](LICENSE)。数据不在许可范围内。
