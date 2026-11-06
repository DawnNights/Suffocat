---@class MarkStorage
---@field marks MintMark[] 刻印列表
MarkStorage = {}

---遍历过滤刻印列表
---@param fn fun(mark: MintMark): boolean 回调函数，返回boolean
---@return MintMark[] 过滤后的刻印列表
function MarkStorage:ForEach(fn)
    return {}
end