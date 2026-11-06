---@type SeerMain
function Main(seer, config)
    seer:SwitchBag(10)
    return true
end

---@type SeerScript
return {
    Type = "UNITY",
    Name = "切换10号背包",
    Author = "醉星河",
    Desc = "UNITY日常-切换10号背包\n切换脚本专用背包(与重聚和XM一致)",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
    end
}
