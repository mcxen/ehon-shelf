# 复用规范

加组件或加能力之前，先看开源里已经怎么做。查过仍要自己写的，只写外壳现在用得到的那一点。

## 三条

1. 组件和能力先调研开源里的做法。先查 [MyGo](https://github.com/egoist/mygo)、[Onevra](https://github.com/oneo-me/onevra)、[Kero](https://github.com/egoist/kero)，再决定写不写。
2. 禁止过度设计，禁止过度防御。不为想象中的第二种用法加接口、抽象层或提前的错误分支。错误只处理真的会发生的那一种。
3. 禁止重复造轮子。下表已有的直接用。表里没有、但上游已经有的，先用上游，不要在 GodGo 里再写一份。

Onevra 自己的做法也是这三条：微调 MyGo 已有的，只补 MyGo 没有的；MyGo 官方有了就删掉自己的。GodGo 同样。不要把 Onevra 整库提前接进来，等到表里那一项真的要用，再 `go get github.com/oneo-me/onevra/source`。

## 先查哪里

| 项目 | 拿什么 | 不拿什么 |
|---|---|---|
| MyGo 0.6 | 窗口、基础控件、插件、更新、打包、无窗口测试 | 不把 `package ui` 包一层自己的控件库 |
| Onevra | 主题控件、窗口壳、成套 Lucide、命令面板、图表、聊天 | 外壳还用不到的不要先依赖 |
| Kero | 侧栏和设置窗口的层次，见 `view.go`、`settings_view.go`、`theme.go` | 终端、daemon、PTY、会话和分屏 |

## 常用，直接用

这些是外壳会碰到的。没列进去的，用到时再查上游，查完补一行。

| 要做的事 | 用 |
|---|---|
| 按钮、输入、开关、分段、弹出菜单、滚动 | MyGo 的 `ui` 基础控件。外壳已有的 `button`、`segmented`、`popup`、`toggle` 只是 Kero 的样式，不另起一套行为 |
| Markdown | `github.com/egoist/mygo/ui/markdown` 的 `View`。规范页已经这么渲染 |
| 滚动边缘和材质 | `plugins/glass` |
| 签名更新、检查、自动下载 | `plugins/updater`，配合 `mygo keygen` 和 `mygo build -upload` |
| 菜单、快捷键、单实例、登录时打开、通知、用系统打开文件 | `mygo.App`、`mygo.Shell` |
| 不打开窗口的界面测试 | `ui.NewTester` |
| 打包、签名、公证、按标签发布 | `mygo.json` 和 MyGo 文档里的 GitHub Actions。发布工作流保持现在这份，不自写打包器 |
| 本地数据库 | 真要存业务数据时用 `plugins/sqlite`。外壳的设置是一个 JSON 文件，不要为它上数据库 |
| 终端 | `plugins/terminal`，以及 Kero 已经接好的做法。外壳不是终端，不要为了以后也许要而接进来 |
| 另一套侧栏或普通窗口壳 | Onevra 的 `shells/sidebar`、`shells/plain`。现在这份侧栏留着；要换壳时换掉，不要两套并存 |
| 主题按钮、选择、表格、对话框 | Onevra 的 `source/components`。MyGo 基础控件够用时不要换 |
| 完整命令面板 | Onevra 的 `components/command`。现在的前往面板够跳转模块和设置；要换成它时删掉自己的，不要两套并存 |
| 代码高亮 | 先继续用 MyGo 的 Markdown。要高亮再看 Onevra，不要自己写高亮器 |
| 图表、流程图、聊天 | Onevra 的 `chart`、`graph`、`treemap`、`flow`、`chat`。外壳没有这些界面，不要预建 |
| 成套图标 | 侧栏继续用 `resources/icons` 里那十几个。要整套 Lucide 时用 Onevra 的 `icons/lucide`，不要再拷一套 SVG |
| 启动器、菜单栏、文档窗口 | MyGo 文档里的 Launchers、菜单和 Documents，或 Onevra 对应的 example。外壳不是这三种应用，不要预建 |

## 什么时候才自己写

MyGo、Onevra、Kero 都没有，而且外壳这一次就要用，才写在 GodGo 里。写完在上面的表加一行，写明为什么上游没有。

自己写的时候：

- 放进已经存在的包，不新建通用框架。
- 不做可选策略，不做只为了以后好换的接口。
- 不把同一份规范抄进 Go 字符串。

## 禁止

- 同一种控件留下 MyGo、Onevra、自己写的三份。
- 先设计扩展点，再找谁来用。
- 为每个返回值包一层哨兵错误，或写调用方还用不上的校验。
- 把上游 README 整篇抄进仓库。这张表只留外壳会碰到的常用项。
