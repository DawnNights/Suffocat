---@class PetBag
PetBag = {}

---清空出战背包的精灵
---@return integer 清空的精灵数量
function PetBag:ClearBattleBag() return 0 end

---清空待命背包的精灵
---@return integer 清空的精灵数量
function PetBag:ClearSupportBag() return 0 end

---清空所有背包的精灵（出战 + 待命）
---@return integer 清空的精灵数量
function PetBag:ClearBag() return 0 end

---获取出战背包中的精灵
---@return BagPet[] 出战背包精灵列表
function PetBag:InBattleBag() return {} end

---获取待命背包中的精灵
---@return BagPet[] 待命背包精灵列表
function PetBag:InSupportBag() return {} end

---遍历精灵背包
---回调函数需要返回一个 boolean（为真则加入返回结果）
---@param fn fun(pet: BagPet): boolean
---@return BagPet[]
function PetBag:ForEach(fn) return {} end

---对精灵背包进行排序
---回调函数需要返回一个 number（用于交换索引）
---@param fn fun(pet: BagPet): number
function PetBag:Sort(fn) end

---返回精灵背包文本
---@return string
function PetBag:String() return "" end