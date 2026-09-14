# allure-report

Gauge 测试框架的 **Allure 3** 报告插件。执行规格后会把 Suite 结果转换成 Allure 结果文件（`*-result.json`），并在本机有 Allure 3 / Node.js 时生成 Awesome HTML 报告。

## 功能

- 作为 Gauge Execution 插件接入，监听 `SuiteExecutionResult`
- 将 Specification / Scenario / Step / Concept / 数据表场景映射为 Allure 用例与步骤
- 支持截图、自定义消息、Hook、标签、失败堆栈
- 写出兼容 Allure 2/3 的 `allure-results`
- 自动调用 Allure 3 生成 HTML 报告（默认单文件 `index.html`）

## 安装

从源码编译并安装到 Gauge 插件目录：

```bash
go run build/make.go --install
```

或先打包 zip，再离线安装：

```bash
go run build/make.go --distro
gauge install allure-report --file deploy/allure-report-0.1.0-<os>.<arch>.zip
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

如果本机已安装 Allure 3 CLI（`allure`）或 Node.js（`npx`），插件会自动生成 HTML。否则仍会保留 `allure-results`，可稍后执行：

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

## 开发

```bash
go test ./...
go run build/make.go
```

要求 Go 1.22+。生成 HTML 报告需要 Allure 3 或 Node.js。
