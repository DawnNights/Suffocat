---@class BagPet
---@field ID integer 精灵ID（uint32）
---@field Name string 精灵名称
---@field CatchTime integer 精灵捕获时间（uint32）
---@field Position PetPosition 精灵所在位置
BagPet = {}

---获取精灵的详细信息
---@return PetInfo 精灵详细信息
function BagPet:PetInfo() return {} end

---移至出战背包
---@return boolean 是否成功
function BagPet:GotoBattleBag() return false end

---移至待命背包
---@return boolean 是否成功
function BagPet:GotoSupportBag() return false end

---移至仓库
---@return boolean 是否成功
function BagPet:GotoStorage() return false end

---与另一个精灵交换位置
---@param pet BagPet 目标精灵
---@return boolean 是否成功
function BagPet:Swap(pet) return false end

---使用指定 ID 的道具
---@param item_id integer 道具ID（uint32）
---@return boolean 是否使用成功
function BagPet:UseItem(item_id) return false end

---装备指定孔位刻印
---@param idx integer 刻印孔位(1~3)
---@param mark MintMark 刻印对象
---@return boolean 是否替换成功
function BagPet:EquipMark(idx, mark) return false end

---卸下指定孔位刻印
---@param idx integer 刻印孔位(1~3)
function BagPet:UnEquipMark(idx) return end

---替换指定位置的技能
---@param idx integer 被替换技能位置(1~5)
---@param skill_id integer 替换技能ID / 技能石ID(若为特攻则ID开头改为2)
---@return boolean 是否替换成功
function BagPet:SetSkill(idx, skill_id) return false end

---@class PetPosition
---@field Type PositionType 位置类别
---@field Name PositionName 位置名称
---@field Index integer 当精灵处于背包时的位置索引（范围：1~12）
---用于表示该位置中的具体槽位编号
---例如：
---- 出战背包中的第 3 个位置：Index = 3
---- 待命背包中的第 1 个位置：Index = 7
PetPosition = {}
