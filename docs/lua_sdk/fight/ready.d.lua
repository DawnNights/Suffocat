---@alias PetDeathHandler fun(player: FightPlayer)
---@alias PetActionHandler fun(roundNum: integer, player: FightPlayer, enemy: FightEnemy)

---@class ReadyFight
ReadyFight = {}

---设置精灵死亡回调
---当我方精灵死亡并完成内部处理后触发
---@param fn PetDeathHandler
function ReadyFight:OnPetDeath(fn) end

---设置精灵行动回调
---每回合行动阶段触发
---@param fn PetActionHandler
function ReadyFight:OnPetAction(fn) end

---绑定战斗方案驱动
---此项与 OnPetDeath 和 OnPetAction 相互覆盖
---@param driver FightDriver
function ReadyFight:BindDriver(driver) end

---开始战斗流程（阻塞直到战斗结束）
---@return boolean 是否战斗胜利
function ReadyFight:Start() return false end
