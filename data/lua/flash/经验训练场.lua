---@type SeerMain
function Main(seer, config)
    -- 判断今日剩余挑战次数
    local times = 2 - seer:Send(42397, function(bs)
        bs:WriteU32(116)
    end):Skip(8):ReadU32()

    if times == 0 then
        logger.Info("今日经验训练场已完成")
        return true
    end

    -- 开启/关闭激励
    seer:Send(42395, function(bs)
        if config["double"] then
            bs:WriteU32(116):WriteU32(1):WriteU32(1):WriteU32(0)
        else
            bs:WriteU32(116):WriteU32(1):WriteU32(0):WriteU32(0)
        end
    end)

    if not config["speedrun"] then
        FS = fightscript.LoadLocal(tostring(config["path"]))
        if FS == nil then
            logger.Warn("请设置经验训练场战斗方案")
            return false
        end

        Driver = seer:ExecFightScript(FS)
        if Driver == nil then
            logger.Warn(string.format("战斗方案 [%s] 映射失败", FS.MetaData.Name))
            return false
        end
    end


    for _ = 1, times, 1 do
        -- 判断是否扫荡
        if config["speedrun"] then
            seer:Send(42395, function(bs)
                bs:WriteU32(116):WriteU32(2):WriteU32(6):WriteU32(0)
            end)
        else
            if FS.ScriptConfig.HPLimit then
                seer:PetHPLimit()
            else
                seer:PetCure()
            end

            local fight = seer:ReadyFight(42396, function(bs)
                bs:WriteU32List(116, 6, 1)
            end)

            fight:BindDriver(assert(Driver))

            if not fight:Start() then
                logger.Warn(string.format("方案 [%s] 挑战失败，请切换战斗方案", FS.MetaData.Name))
                return false
            end
        end
    end

    if not seer:Send(42395, function(bs)
            bs:WriteU32List(116, 3, 0, 0)
        end):IsEmpty() then
        logger.Info(string.format("已领取经验训练场每日礼包，当前已有%d/200", seer:ItemCount(2300005)))
    end


    return true
end

---@type SeerScript
return {
    Type = "FLASH",
    Name = "经验训练场",
    Author = "醉星河",
    Desc = "FLASH日常-经验训练场\n挑战经验训练场第6层",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddCheckbox("speedrun", "优先扫荡", false)
        cr:AddCheckbox("double", "开启激励", false)
        cr:AddFightScript("path", "战斗方案")
    end
}
