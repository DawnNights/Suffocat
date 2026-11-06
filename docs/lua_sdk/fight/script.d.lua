-- ----------------------------
-- 赛尔配置（SeerSet）
-- ----------------------------

---@class SeerSet
---@field BattleFire integer 火焰等级
---@field Suit integer 套装ID
---@field Equip integer 目镜ID
---@field Title integer 称号ID
---@field EnableEquipments boolean 是否启用套装
---@field EnableTitle boolean 是否启用称号
SeerSet = {}

-- ----------------------------
-- 脚本全局设置（ScriptConfig）
-- ----------------------------

---@class ScriptConfig
---@field FightRoundLimit integer 战斗回合上限
---@field HPLimit boolean 是否压血
---@field ClothMust boolean 战斗回合上限
---@field TitleMust boolean 战斗回合上限
---@field SupportPetsMust boolean 战斗回合上限
ScriptConfig = {}

-- ----------------------------
-- 动作类型（Use）
-- ----------------------------

---@class Use
---@field Type string 动作类型
---@field Arg integer|nil 参数
Use = {}

-- ----------------------------
-- 条件（Condition）
-- ----------------------------

---@class Condition
---@field Left string 左值
---@field LeftArg integer | string | nil 左值参数
---@field Operator string 运算符
---@field Right integer | nil 右值
Condition = {}

-- ----------------------------
-- 动作步骤（Step）
-- ----------------------------

---@class Step
---@field Use Use 动作
---@field If Condition[] 条件列表
Step = {}

-- ----------------------------
-- 精灵附加配置（PetConfig）
-- ----------------------------

---@class PetConfig
---@field FightItems integer[] 战斗道具列表
---@field HpLimit integer 压血
---@field Level integer 等级限制
---@field DeathChange integer 死亡切换索引
PetConfig = {}

-- ----------------------------
-- 阵容精灵（RosterPet）
-- ----------------------------

---@class RosterPet
---@field ID integer 精灵ID
---@field Config PetConfig 精灵配置
---@field ActionSequence Step[][] 行动序列
RosterPet = {}

-- ----------------------------
-- 脚本元数据（MetaData）
-- ----------------------------
---@class MetaData
---@field Name string 脚本名称
MetaData = {}

-- ----------------------------
-- 战斗脚本（FightScript）
-- ----------------------------

---@class FightScript
---@field MetaData MetaData 元数据
---@field SeerSet SeerSet 赛尔配置
---@field ScriptConfig ScriptConfig 脚本设置
---@field RosterSet RosterPet[] 阵容列表
FightScript = {}

-- ----------------------------
-- 战斗驱动（FightDriver）
-- ----------------------------

---@class FightDriver
FightDriver = {}

---精灵死亡回调函数
---@param player FightPlayer
function FightDriver:OnPetDeath(player) end

---精灵精灵行动回调函数
---@param roundNum integer
---@param player FightPlayer
---@param enemy FightEnemy
function FightDriver:OnPetAction(roundNum, player, enemy) end
