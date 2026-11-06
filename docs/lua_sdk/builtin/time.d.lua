---@class Date
---@field Year integer   年
---@field Mouth integer  月
---@field Day integer    日
---@field Hour integer   时
---@field Minute integer 分
---@field Second integer 秒

---@class time
time = {}

---获取当前日期时间
---@return Date date 当前年月日时分秒
function time.Date() return {} end

---休眠指定秒数
---@param second integer 等待秒数
function time.Sleep(second) end

---获取当前 Unix 时间戳（秒）
---@return integer unix 当前时间戳（秒）
function time.Unix() return 0 end

---获取当前 Unix 时间戳（毫秒）
---@return integer unixMilli 当前时间戳（毫秒）
function time.UnixMilli() return 0 end
