---@class ConfigRegister
ConfigRegister = {}

--- 注册勾选框
---@param key string
---@param desc string
---@param default boolean
function ConfigRegister:AddCheckbox(key, desc, default) end

--- 注册文本输入框
---@param key string
---@param desc string
---@param default string
function ConfigRegister:AddInput(key, desc, default) end

--- 注册数值输入框
---@param key string
---@param desc string
---@param default number
---@param options [number, number, number] @[最小值, 最大值, 步长] 数值范围约束
function ConfigRegister:AddNumberInput(key, desc, default, options) end

--- 注册下拉框
---@param key string
---@param desc string
---@param default string
---@param options string[]
function ConfigRegister:AddSelect(key, desc, default, options) end

--- 注册多选框
---@param key string
---@param desc string
---@param default string[]
---@param options string[]
function ConfigRegister:AddMultiSelect(key, desc, default, options) end

--- 注册战斗方案选择器，值为战斗方案文件相对路径
---@param key string
---@param desc string
function ConfigRegister:AddFightScript(key, desc) end
