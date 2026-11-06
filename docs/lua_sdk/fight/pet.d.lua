-- ----------------------------
-- 出战背包精灵（FightBagPet）
-- ----------------------------

---@class FightBagPet
---@field CatchTime integer 捕获时间（uint32）
---@field ID integer 精灵ID（uint32）
---@field HP integer 当前HP
---@field MaxHP integer 最大HP
---@field Name string 精灵名称
FightBagPet = {}

-- ----------------------------
-- 强化等级（FightLevel）
-- ----------------------------

---@class FightLevel
---@field Atk integer 攻击强化等级（-6~6）
---@field Def integer 防御强化等级
---@field SpAtk integer 特攻强化等级
---@field SpDef integer 特防强化等级
---@field Speed integer 速度强化等级
---@field Accuracy integer 命中强化等级
FightLevel = {}

-- ----------------------------
-- 当前在场精灵（FightActivePet）
-- ----------------------------

---@class FightActivePet : FightBagPet
---@field FightLV FightLevel 强化等级
---@field FightStatus integer[] 状态列表
---@field SkillList PetSkill[] 精灵技能列表
FightActivePet = {}

---判断精灵是否死亡
---@return boolean
function FightActivePet:IsDeath() return false end

-- ----------------------------
-- 精灵技能（PetSkill）
-- ----------------------------

---@class PetSkill
---@field ID integer 技能ID（uint32）
---@field PP integer 剩余使用次数
PetSkill = {}

---获取技能详细信息
---@return PetSkillMoreInfo
function PetSkill:Info() return {} end

-- ----------------------------
-- 技能结算（SkillResult）
-- ----------------------------

---@class SkillResult
---@field Skill PetSkillMoreInfo 技能信息
---@field AskTimes integer 连击次数
---@field IsCrit boolean 是否暴击
---@field Damage integer 红伤
---@field GainDamage integer 粉伤
---@field RealDamage integer 真伤
---@field ZhuijiDamage integer 二段伤
SkillResult = {}
