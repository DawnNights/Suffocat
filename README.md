# Suffocat

> 项目名 **Suffocat** 取自本人在赛尔号中最喜欢的精灵「萨芙凯特」（精灵 ID：4604）。

![萨芙凯特](docs/assets/suffocat.png)

赛尔号脱机任务工具。项目由 AI 从 `MWHelper` 中的脱机功能迁移而来，保留协议实现、Lua 接口绑定、战斗方案编辑器与 Lua 脚本工作台，并做了以下调整：

- 数据源由 `mwdb` 改为 [SeerAPI](https://github.com/SeerAPI/seerapi)，本地使用 SQLite 缓存，按需请求并写入数据库。
- 脚本改为明文 `.lua`，不再加密、打包为 `.sfc`。
- 战斗方案不再嵌入脚本，脚本只从战斗方案编辑器目录读取 YAML 方案。
- 因子任务由 Go 原生执行，不再为每个因子生成重复脚本；难度、扫荡、失败次数与逐关端口/方案保存在 `data/config/factors.json`。

## 目录结构

```text
Suffocat/
├── main.go                 程序入口
├── internal/               实现代码
│   ├── common/             通用工具（ByteStream、HTTP、Lua 转换、协议算法）
│   ├── config/             运行路径配置
│   ├── protocol/           游戏协议
│   │   ├── command/        协议命令常量
│   │   └── schema/         协议数据结构（封包解析结果）
│   ├── seerapi/            SeerAPI 客户端、数据模型、SQLite 缓存
│   ├── lua/                Lua 接口绑定
│   │   ├── builtin/        Lua 内置库
│   │   ├── fight/          战斗相关绑定
│   │   ├── mintmark/       刻印绑定
│   │   └── petbag/         精灵背包绑定
│   └── web/                HTTP 服务与网页资源
│       ├── fight_editor/   战斗方案编辑器
│       └── studio/         Suffocat Studio（Lua 脚本工作台）
├── docs/lua_sdk/           Lua 类型注释（IDE 补全用）
├── data/                   全部运行时数据
│   ├── db/                 SeerAPI SQLite 缓存
│   ├── config/             脚本配置、启用状态、排序、因子设置
│   ├── lua/                明文 Lua 脚本
│   │   ├── flash/          FLASH 端脚本
│   │   └── unity/          UNITY 端脚本
│   └── scripts/            战斗方案 YAML（编辑器唯一目录）
├── go.mod
├── go.sum
└── README.md
```

### 代码目录说明

| 目录 | 职责 |
|---|---|
| `internal/config` | 运行路径配置 |
| `internal/protocol` | 游戏协议、封包、命令与数据结构，替代原 `seer/` |
| `internal/seerapi` | SeerAPI 请求、数据模型与 SQLite 缓存 |
| `internal/lua` | Lua 接口绑定，替代原 `luabind/` |
| `internal/web` | HTTP 接口与网页资源，替代原 `web/` |
| `internal/common` | ByteStream、HTTP、Lua 转换等跨包工具，替代原 `utils/` |

## 快速开始

```bash
go mod tidy
go run .
```

默认监听 `http://127.0.0.1:4604`：

- Lua 脚本工作台：`http://127.0.0.1:4604/studio/`
- 战斗方案管理器：`http://127.0.0.1:4604/fight_manager`
- 战斗方案编辑器：`http://127.0.0.1:4604/fight_editor?file=xxx.yaml`

### 启动参数

| 参数 | 说明 |
|---|---|
| `-addr :4604` | HTTP 监听地址 |
| `-root .` | 项目根目录，默认为当前工作目录；运行时数据位于其下的 `data/` |
| `-reset-cache` | 启动前清空 SQLite 缓存 |

### 数据获取策略

程序启动时**不会**全量拉取游戏数据：

1. **本地优先**：宠物详情、技能、道具、刻印、套装、部件都先查 `data/db/seerapi.db`，命中直接返回。
2. **未命中才请求**：本地没有时调用 SeerAPI，成功后写入 SQLite 再返回。
3. **静态数据不自动刷新**：技能、道具、刻印、套装、部件视为不会变化的数据；一旦缓存就不再过期刷新。
4. **精灵名称索引**：战斗方案编辑器的精灵下拉框需要完整名称列表，因此单独下载一个约 100KB 的 `name-index.json`（仅 ID -> 名称），不拉取全部精灵详情。
5. **游戏配置版本检查**：启动时请求 `https://seer.61.com/version/zzz_config.txt` 并保存版本号；版本变化时读取精灵分片索引的 `by_id`，对比本地缺失的精灵 ID，只下载包含这些 ID 的分片，同时重新同步因子道具与关卡数据。
6. **显式批量刷新**：需要一次性预填充或重新导入时，使用界面中的“数据同步”，或调用 `POST /studio/data/sync` 在后台批量刷新。

清空缓存：

```bash
go run . -reset-cache
# 或直接删除 data/db/seerapi.db
```

## 脚本

脚本放在 `data/lua/` 下的 `flash/` 或 `unity/` 目录，文件为 UTF-8 明文 `.lua`。程序会根据脚本中的 `Type` 自动归档；`UNITY-`、`UNITY_`、`FLASH-`、`FLASH_` 文件名前缀会自动去掉。导入或保存时如果脚本放错目录，也会自动移动到正确目录：

```lua
---@type SeerMain
function Main(seer, config)
    logger.Info("Hello Suffocat")
    return true
end

---@type SeerScript
return {
    Type = "UNITY", -- 或 "FLASH"
    Name = "示例脚本",
    Author = "匿名",
    Desc = "脚本说明",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddCheckbox("enable", "开关", true)
    end,
}
```

面板支持新建、编辑、重命名、删除、导入 `.lua`、启用/禁用、拖拽排序、单脚本运行与批量运行。

`Setup` 可选；不提供时脚本没有配置项。脚本配置保存在 `data/config/scripts/`，按文件名区分。

## 战斗方案

- 战斗方案编辑器只管理 `data/scripts/` 目录（编辑器根目录）。
- 脚本中通过 `fightscript.LoadLocal("子目录/方案.yaml")` 读取方案；省略 `.yaml` 后缀也可以。
- 方案路径被限制在 `data/scripts/` 内，禁止 `..`、绝对路径与目录穿越。
- 原 `fightscript.LoadPrivate` 已移除，脚本不再从 `.sfc` 中读取内嵌方案。

战斗方案选择器（`AddFightScript`）的值同样是相对于 `data/scripts/` 的路径。

## SeerAPI 数据与缓存

`internal/seerapi/` 包提供以下数据结构与接口：

| 数据类型 | SeerAPI 端点 | SQLite 表 |
|---|---|---|
| 精灵 | `/v1/pet/{id}` | `pets` |
| 技能 | `/v1/skill/{id}` | `skills` |
| 道具 | `/v1/item/{id}` | `items` |
| 刻印 | `/v1/mintmark/{id}` | `mintmarks` |
| 套装 | `/v1/suit/{id}` | `suits` |
| 套装部件 | `/v1/equip/{id}` | `equips` |
| 因子道具 | `/v1/factor/{id}` | `factors` |
| 因子关卡 | `/v1/factor_design/{id}` | `factor_designs` |

SQLite 数据库位于 `data/db/seerapi.db`，数据按字段存储：

- 实体表：`pets`、`skills`、`items`、`mintmarks`、`suits`、`equips`、`factors`、`factor_designs`；
- 子表：`pet_skills`、`pet_soulmarks`、`suit_equips` 及因子关卡的难度、BOSS、规则、禁用精灵子表，按 `position` 保持数据源顺序；
- 每张实体表包含 `id`、`name`、`updated_at` 及各业务字段，`updated_at` 用于标记该行是否已写入详情；
- `pets` 名称索引行只写入 `id`/`name`，`updated_at` 为 0，首次读取详情时再补全。

精灵数据与因子数据都会在启动时根据 `zzz_config.txt` 的版本号判断是否需要更新。因子数据通过 `/v1/factor/`、`/v1/factor_design/` 分页同步，并使用一个事务整体替换，避免出现半套数据。

旧版数据库把整条记录以 JSON 原文存入 `data` 列。启动时检测到这种表会在事务中自动迁移为字段表，迁移完成后执行 `VACUUM` 回收空间。旧版 JSONL 缓存仍会在对应表为空时自动导入。

## 因子数据

因子列表、名称、兑换材料、各难度关卡、BOSS、规则、禁用精灵以及进度变量全部来自 SeerAPI：

- `/v1/factor/`：因子道具资源；
- `/v1/factor_design/`：因子关卡资源，包含 `time_value`、`progress_value`、`fail_times`。

两者在启动时根据 `https://seer.61.com/version/zzz_config.txt` 的版本变化决定是否重新同步；同一版本号也会触发精灵数据的批量更新，并保留 `data/config/factors.json` 中的本地勾选与关卡设置。

当前接口未返回每日挑战次数 `Times`，代码按游戏固定值 `3` 处理；如果后续接口增加 `times` 字段，会自动使用接口值。

因子任务不生成脚本，直接在 Go 中登录、检查进度并按关卡执行：

- 列表按因子设计 ID 从大到小排列，条目名称随难度变化（`简单-xxx`、`普通-xxx`、`困难-xxx`）。
- 所有因子默认不勾选；勾选后点击“运行已选因子”执行所有已勾选的因子。
- 每条因子的设置包括难度、优先扫荡、允许失败次数（默认 10），以及每一关的登录端口和挑战方案。
- 设置保存在 `data/config/factors.json`，挑战方案仍只从 `data/scripts` 读取。
- 每次登录后首次需要切换时读取一次角色称号与套装并缓存；后续关卡若状态相同则跳过切换，重新登录后缓存自动失效。

## 说明

- 账号密码由前端提交给本地后端用于游戏登录；与原始面板一致，前端会把账号密码保存在浏览器 `localStorage` 中，请在共用电脑上自行清理。
- 后端不会把账号密码写入日志或数据文件。
- 协议层的封包加密、心跳与危险命令拦截保持原实现。
- Lua 类型注释位于 `docs/lua_sdk/`，可用于 VS Code + Lua Language Server 补全。

## 特别感谢

感谢 [坚果](https://github.com/Nattsu39) 和他的项目 [SeerAPI](https://github.com/SeerAPI/seerapi)为本项目的数据来源提供支持。
