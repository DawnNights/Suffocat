---@class json
json = {}

---将数据解析为json字符串
---@param obj table
---@param indent? integer 缩进空格数，不传或为0时输出紧凑JSON
---@return string jsonString 序列化后的JSON字符串
function json.dumps(obj, indent) return "" end

---将JSON字符串解析为Lua table
---@param str string JSON字符串
---@return table|nil result 解析后的Lua table，失败时返回nil
---@return string? err 解析失败时的错误信息
function json.loads(str) return nil, "" end
