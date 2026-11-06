---@class fightscript
fightscript = {}

---从YAML代码解析战斗方案
---@param source string 战斗方案YAML格式代码
---@return FightScript | nil
function fightscript.New(source) return {} end

---从战斗方案编辑器目录加载战斗方案
---@param path string 相对 plans 目录的路径，可省略 .yaml 后缀
---@return FightScript | nil
function fightscript.LoadLocal(path) return {} end

return fightscript
