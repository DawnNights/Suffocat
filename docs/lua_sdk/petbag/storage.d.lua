---@class PetStorage
PetStorage = {}

---获取位于仓库中的精灵
---@return BagPet[] 仓库中的精灵列表
function PetStorage:InStorage() return {} end

---获取不在仓库中的精灵
---@return BagPet[] 其它位置的精灵列表
function PetStorage:InOther() return {} end

---遍历精灵仓库
------回调函数需要返回一个 boolean（为真则加入返回结果）
---@param fn fun(pet: BagPet): boolean
---@return BagPet[]
function PetStorage:ForEach(fn) return {} end
