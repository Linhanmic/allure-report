# allure-report

Gauge 测试框架的 **Allure 3** 报告插件。执行规格后会把 Suite 结果转换成 Allure 结果文件（`*-result.json`），并优先使用插件内置的 Allure 3 生成 Awesome HTML 报告（离线可用，仅需本机 Node.js）。

## 功能

- 作为 Gauge Execution 插件接入，监听 `SuiteExecutionResult`
- 将 Specification / Scenario / Step / Concept / 数据表场景映射为 Allure 用例与步骤
- 支持截图、自定义消息、Hook、标签、失败堆栈
- 写出兼容 Allure 2/3 的 `allure-results`
- 插件包内置 Allure 3 CLI，离线环境只需 Node.js 即可生成 HTML（默认单文件 `index.html`）

## 安装

从源码编译并安装到 Gauge 插件目录：

```bash
go run build/make.go --install
```

或先打包 zip，再离线安装：

```bash
make distro
gauge install allure-report --file deploy/allure-report-0.1.1-<os>.<arch>.zip
```

跨平台打包：

```bash
make distro-all
```

发布到 GitHub Release（推送 `v*` 标签后自动上传 `deploy/*.zip`）：

```bash
git tag v0.1.0
git push origin v0.1.0
```

在 Gauge 项目的 `manifest.json` 中加入插件：

```json
{
  "Language": "js",
  "Plugins": ["html-report", "allure-report"]
}
```

## 使用

```bash
gauge run specs
```

默认输出：

| 路径 | 内容 |
| --- | --- |
| `reports/allure-results/` | Allure 原始结果 |
| `reports/allure-report/` | Allure 3 HTML 报告 |

### 离线报告生成

插件 zip 内已包含 `bundled/node_modules/allure`（Allure 3.16.1）。安装插件后，只要本机有 **Node.js**（`node` 在 PATH 中），即可在完全离线环境下生成 HTML，无需联网下载或全局安装 `allure` / `npx`。

生成优先级：

1. 插件内置 Allure（`node bundled/generate.mjs`）
2. 系统 `allure` CLI（若已安装）
3. `npx allure@3`（需联网）

若未安装 Node.js，仍会保留 `allure-results`，可稍后在有 Node 的机器上生成：

```bash
npx --yes allure@3 awesome reports/allure-results --output reports/allure-report --single-file --report-language zh
```

## 配置

在 `env/default/default.properties` 中设置：

```properties
# Gauge 通用
gauge_reports_dir = reports
overwrite_reports = true

# Allure 插件
allure_report_generate = true
allure_report_single_file = true
allure_report_language = zh
allure_report_theme = auto
allure_report_name = Gauge Allure Report
allure_results_dir =
allure_report_dir =
allure_issue_pattern = https://jira.example.com/browse/%s
allure_tms_pattern = https://tms.example.com/case/%s
```

| 属性 | 说明 | 默认值 |
| --- | --- | --- |
| `gauge_reports_dir` | 报告根目录 | `reports` |
| `overwrite_reports` | `true` 覆盖最近一次报告；`false` 按时间戳保留历史 | Gauge 项目设置 |
| `allure_report_generate` | 是否尝试生成 Allure 3 HTML | `true` |
| `allure_report_single_file` | 生成单文件 HTML，便于直接打开 | `true` |
| `allure_report_language` | 报告语言 | `zh` |
| `allure_report_theme` | `light` / `dark` / `auto` | `auto` |
| `allure_report_name` | 报告标题 | `Gauge Allure Report` |
| `allure_results_dir` | 结果目录（相对项目根或绝对路径） | `<reports>/allure-results` |
| `allure_report_dir` | HTML 目录 | `<reports>/allure-report` |
| `allure_issue_pattern` | Issue 链接模板，`%s` 为标签值 | 空 |
| `allure_tms_pattern` | TMS 链接模板 | 空 |

## 标签约定

规格或场景上的标签会映射为 Allure labels / links：

| Gauge 标签 | Allure |
| --- | --- |
| `severity:critical` 或 `critical` | severity |
| `owner:alice` | owner |
| `epic:交易` | epic |
| `feature:登录` | feature |
| `story:记住密码` | story |
| `issue:BUG-12` | issue 链接 |
| `tms:TC-1` | tms 链接 |
| `layer:e2e` | layer |
| `link:https://example.com\|文档` | 自定义链接 |
| 其他标签 | tag |

示例：

```markdown
# 用户认证
tags: epic:账户, feature:登录

## 管理员可以登录
tags: severity:critical, owner:qa, issue:BUG-12
* 打开首页
* 输入用户名 "admin"
* 登录成功
```

## 映射规则

- Specification → Allure suite / feature
- Scenario → Allure test
- Concept → 嵌套 step
- 数据表行 → Allure parameters + HTML 表格附件
- Context / Teardown → 分组步骤
- before/after hook → container fixtures
- 断言失败 → `failed`；验证/Hook 异常 → `broken`；跳过 → `skipped`
- Gauge 截图文件与自定义消息 → attachments / log steps

## 示例项目

仓库内提供了不依赖浏览器的 Gauge JS 示例：

```bash
make install
cd examples/gauge-js
gauge run specs
```

示例会生成包含通过、失败、跳过、Concept、数据表、标签与截图附件的 Allure 报告。

## 开发

```bash
go test ./...
make build
make example
```

要求 Go 1.22+。打包前会在 `bundled/` 执行 `npm ci` 下载 Allure 3 依赖；生成 HTML 报告需要 Node.js（优先使用内置 Allure）。
