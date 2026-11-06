---@class Seer
---@field UserID integer 用户米米号
---@field Type SeerType 用户登录端口
Seer = {}

-- ================= 发包相关 =================

--- 发送封包并等待响应包
---@param cmd integer
---@param fn fun(bs: ByteStream)
---@return ByteStream 响应包包体
function Seer:Send(cmd, fn) return {} end

--- 接收指定封包
---@param cmd integer
---@return ByteStream 响应包包体
function Seer:Recv(cmd) return {} end

--- 发送战斗封包并返回战斗处理结构
---@param cmd integer
---@param fn fun(bs: ByteStream)
---@return ReadyFight
function Seer:ReadyFight(cmd, fn) return {} end

-- ================= 精灵相关 =================

--- 返回精灵背包
---@return PetBag
function Seer:PetBag() return {} end

--- 设置精灵背包
---@param battlePets integer[]  出战精灵ID列表（最多6个）
---@param supportPets integer[] 待命精灵ID列表（最多6个）
---@return integer[] notFoundList 未找到/失败的精灵ID列表
function Seer:SetPetBag(battlePets, supportPets) return {} end

--- 返回精灵仓库
---@return PetStorage
function Seer:PetStorage() return {} end

--- 返回刻印仓库
---@return MarkStorage
function Seer:MarkStorage() return {} end

--- 切换背包阵容
---@param idx integer
function Seer:SwitchBag(idx) end

--- 判断是否拥有指定ID精灵
---@param pet_id integer
function Seer:HasPet(pet_id) end

-- ================= 用户相关 =================

--- 统计指定道具数量
---@param item_id integer 道具ID
---@return integer 道具数量
function Seer:ItemCount(item_id) return 0 end

--- 购买指定赛尔豆道具
---@param item_id integer 道具ID
---@param num integer 道具数量
---@return boolean 是否购买成功
function Seer:ItemBuy(item_id, num) return false end

--- 设置称号
---@param title_id TitleID 称号ID
---@return boolean 是否设置成功
function Seer:SetTitle(title_id) return false end

--- 设置套装
---@param suit_id SuitID 套装ID
---@param equip_id EquipID 目镜ID
---@return boolean 是否设置成功
function Seer:SetCloth(suit_id, equip_id) return false end

-- 获取火焰
---@param fire_type FireType 火焰类型
---@return boolean 是否获取成功
function Seer:GetFire(fire_type) return false end

-- 治疗背包精灵
function Seer:PetCure() return end

-- 开关自动治疗背包精灵
---@param open boolean 是否开启
function Seer:PetCureAuto(open) return end

---背包精灵压血（治疗 + 关闭自动治疗 + 购买并使用压血道具）
function Seer:PetHPLimit() return end

---获取指定用户信息
---@param userID integer 用户米米号
---@return UserInfo
function Seer:GetUserInfo(userID) return {} end

---执行战斗方案脚本
---@param fs FightScript 战斗方案
---@return FightDriver | nil
function Seer:ExecFightScript(fs) return {} end

---@class UserInfo
---@field UserID integer 用户米米号
---@field NickName string 用户昵称
---@field Vip integer VIP类型
---@field VipLevel integer VIP等级
---@field MapType integer 当前所在地图类型
---@field MapID integer 当前所在地图实例ID
---@field TeamID integer 所在战队ID
---@field FireBuff integer 火焰Buff
---@field Title integer 佩戴称号ID
---@field Clothes integer[] 当前穿戴套装部件列表
UserInfo = {}

return Seer
