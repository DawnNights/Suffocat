---@type SeerMain
function Main(seer, config)
    -- 开启/关闭扫荡激励
    seer:Send(42395, function(bs)
        if config["double"] then
            bs:WriteU32(104):WriteU32(4):WriteU32(1):WriteU32(0)
        else
            bs:WriteU32(104):WriteU32(4):WriteU32(0):WriteU32(0)
        end
    end)

    -- 扫荡指定难度
    local body = seer:Send(42395, function(bs)
        if config["level"] == "简单" then
            bs:WriteU32(104):WriteU32(6):WriteU32(1):WriteU32(0)
        elseif config["level"] == "普通" then
            bs:WriteU32(104):WriteU32(6):WriteU32(2):WriteU32(0)
        else
            bs:WriteU32(104):WriteU32(6):WriteU32(3):WriteU32(0)
        end
    end)

    if not body:IsEmpty() then
        logger.Info(string.format("泰坦矿洞%s模式扫荡成功", config["level"]))
    end
    return true
end

---@type SeerScript
return {
    Type = "UNITY",
    Name = "泰坦矿洞扫荡",
    Author = "醉星河",
    Desc = "Unity日常-泰坦矿洞扫荡",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddSelect("level", "一键扫荡难度", "困难", { "简单", "普通", "困难" })
        cr:AddCheckbox("double", "使用激励模组", true)
    end
}
