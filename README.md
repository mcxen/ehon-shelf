# 绘本书架 Ehon Shelf

一个本地运行的书架式阅读器，用于离线浏览国際日本文化研究センター（日文研）公开的
「艶本資料データベース」（https://lapis.nichibun.ac.jp/enp/ ，约 470 部作品）。

- 书架界面：封面由每部作品首页高清图生成，支持搜索书名、绘师筛选
- 阅读器：右至左 / 左至右、单页 / 双页、缩放、键盘翻页，自动记住阅读位置
- 设置页：获取远程完整目录、勾选或全部下载、进度显示、暂停 / 继续、设置数据文件夹和请求间隔、下载示例数据

## 数据来源与版权

所有图像与书目数据来自 **国際日本文化研究センター「艶本資料データベース」**，版权及相关权利归其所有。
本仓库只包含程序代码（MIT 许可证），**不包含也不授权任何图像数据**。下载的数据仅供个人学习研究使用，
如需转载、出版或其他用途，请遵循日文研的利用规定并取得许可。

## 安装

### 方式一：下载安装包
到 [Releases](https://github.com/mcxen/ehon-shelf/releases/latest) 下载对应平台的文件：

| 平台 | 文件 |
|---|---|
| macOS (Apple Silicon) | `EhonShelf-macos-arm64.dmg` 或 `.zip` |
| Windows x64 | `EhonShelf-windows-x64.zip`，解压后运行 `EhonShelf.exe` |
| Linux x64 | `EhonShelf-linux-x64.tar.gz`，解压后运行 `./EhonShelf` |

启动后会自动在浏览器中打开 `http://127.0.0.1:8765/`。

macOS 提示：安装包未经 Apple 签名公证，首次打开时请在 Finder 中右键点击应用选择「打开」，
或执行 `xattr -dr com.apple.quarantine /Applications/EhonShelf.app`。
Windows 可能出现 SmartScreen 提示，选择「更多信息」→「仍要运行」。

### 方式二：从源码运行
需要 Python 3.9+：
```bash
git clone https://github.com/mcxen/ehon-shelf.git
cd ehon-shelf
pip install -r requirements.txt
python app.py            # 可选参数 --port 8765 --no-browser
```

## 获取数据

1. 打开书架右上角的「设置」。
2. 点击「下载示例数据」可从最新发布版获取 3 部示例作品（约 190 MB）。
3. 在「远程目录」中点击「获取目录列表」，勾选作品后点「下载所选」，或点「下载全部」。
4. 下载过程可随时「暂停」/「继续」；关闭程序后再次下载同一作品会跳过已完成的页面。

默认数据文件夹为 `~/EhonShelf/data`（配置保存在 `~/EhonShelf/config.json`），可在设置页修改。
全部作品数据量较大（数十 GB 级），请预留足够磁盘空间。

## 礼貌抓取说明

- 所有请求串行发送，默认间隔 1.2 秒，最小 1 秒，不可设置更低
- 启动时读取并遵守 `robots.txt`；遇到错误按指数退避重试
- 使用固定的 User-Agent 标识本程序
- 请避免在短时间内反复下载全部数据，以免给日文研服务器造成负担

## 构建

```bash
pip install pyinstaller -r requirements.txt
python scripts/build.py   # 输出在 dist/
```
推送 `v*` 标签会触发 GitHub Actions，自动构建 macOS / Windows / Linux 安装包并发布到 Release。

## 许可证

代码采用 [MIT License](LICENSE)。数据不在许可范围内。
