---@class bit
bit = {}

---按位与，对多个整数执行按位与运算
---@param a integer
---@param ... integer
---@return integer
function bit.band(a, ...) return 0 end

---按位或，对多个整数执行按位或运算
---@param a integer
---@param ... integer
---@return integer
function bit.bor(a, ...) return 0 end

---按位异或，对多个整数执行按位异或运算
---@param a integer
---@param ... integer
---@return integer
function bit.bxor(a, ...) return 0 end

---按位非，对整数执行按位取反运算（32位）
---@param a integer
---@return integer
function bit.bnot(a) return 0 end

---逻辑左移
---@param a integer
---@param n integer 移位的位数
---@return integer
function bit.lshift(a, n) return 0 end

---逻辑右移
---@param a integer
---@param n integer 移位的位数
---@return integer
function bit.rshift(a, n) return 0 end

---算术右移（保留符号位）
---@param a integer
---@param n integer 移位的位数
---@return integer
function bit.arshift(a, n) return 0 end
