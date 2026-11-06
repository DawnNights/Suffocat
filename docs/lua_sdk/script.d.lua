---@alias SeerMain fun(seer: Seer, config: table<string, boolean|string|string[]|nil>): boolean
---@alias SeerSetup fun(cr: ConfigRegister)


---@class SeerScript
---@field Type SeerType 脚本登录端口
---@field Main SeerMain 脚本主函数
---@field Setup SeerSetup 脚本设置函数
---@field Name string 脚本名称
---@field Author string 脚本作者
---@field Desc string 脚本简介
---@field Version string 脚本版本
