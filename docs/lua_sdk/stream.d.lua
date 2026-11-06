---@class ByteStream
local ByteStream = {}

---@param n integer
---@return ByteStream
function ByteStream:WriteU8(n) return {} end

---@param n integer
---@return ByteStream
function ByteStream:WriteU16(n) return {} end

---@param n integer
---@return ByteStream
function ByteStream:WriteU32(n) return {} end

---@param ... integer
---@return ByteStream
function ByteStream:WriteU32List(...) return {} end

---@param hex string
---@return ByteStream
function ByteStream:WriteHex(hex) return {} end

---@return integer
function ByteStream:ReadU8() return 0 end

---@return integer
function ByteStream:ReadU16() return 0 end

---@return integer
function ByteStream:ReadU32() return 0 end

---@return integer
function ByteStream:Length() return 0 end

function ByteStream:Clear() return 0 end

---@return string
function ByteStream:Hex() return "" end

---@return boolean
function ByteStream:IsEmpty() return false end

---@param len integer
---@return ByteStream
function ByteStream:Skip(len) return {} end

return ByteStream
