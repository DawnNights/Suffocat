---@class FightEnemy
---@field PetBag FightBagPet[] 战斗中的精灵列表
---@field ActivePet FightActivePet 当前出战精灵
---@field LastResult SkillResult 上一次技能结算结果
FightEnemy = {}


---@class FightPlayer : FightEnemy
---@field UserID integer 玩家ID（uint32）
---@field Name string 玩家名称
---@field Clothes integer[] 穿戴信息
FightPlayer = {}

---使用指定技能ID
---@param skillID integer 技能ID
function FightPlayer:UseSkill(skillID) end

---按技能索引使用技能
---@param skillIdx integer 技能索引（1开始）
function FightPlayer:UseSkillByIdx(skillIdx) end

---按技能名称使用技能（匹配包含字符串）
---@param skillName string 技能名称
function FightPlayer:UseSkillByName(skillName) end

---使用指定道具
---@param itemID integer 道具ID
function FightPlayer:UseItem(itemID) end

---放弃操作
function FightPlayer:UseSkip() end

---切换出战精灵
---@param pet FightBagPet
function FightPlayer:SwitchPet(pet) end

---按精灵ID切换出战精灵
---@param petID integer 精灵ID
function FightPlayer:SwitchPetByID(petID) end

---按精灵名称切换出战精灵（匹配包含字符串）
---@param petName string 精灵名称
function FightPlayer:SwitchPetByName(petName) end

---切换下一个可用精灵（非当前精灵且HP>0）
function FightPlayer:SwitchNextPet() end

---逃跑
function FightPlayer:Escape() end
