---@type SeerMain
function Main(seer, config)
    -- 判断今日剩余挑战次数
    local times = 5 - seer:Send(46046, function(bs)
        bs:WriteU32(1):WriteU32(18709)
    end):Skip(4):ReadU32()

    if times == 0 then
        logger.Info("今日勇者之塔已完成")
        return true
    end

    -- 开启/关闭激励
    seer:Send(42395, function(bs)
        if config["double"] then
            bs:WriteU32(117):WriteU32(1):WriteU32(1):WriteU32(0)
        else
            bs:WriteU32(117):WriteU32(1):WriteU32(0):WriteU32(0)
        end
    end)

    -- 判断是否扫荡
    if config["speedrun"] then
        for _ = 1, times, 1 do
            seer:Send(42395, function(bs)
                bs:WriteU32(117):WriteU32(2):WriteU32(30):WriteU32(0)
            end)
        end
    else
        local fs = fightscript.LoadLocal(tostring(config["path"]))
        if fs == nil then
            logger.Warn("请设置勇者之塔战斗方案")
            return false
        end

        local driver = seer:ExecFightScript(fs)
        if driver == nil then
            logger.Warn(string.format("战斗方案 [%s] 映射失败", fs.MetaData.Name))
            return false
        end

        for _ = 1, times, 1 do
            if fs.ScriptConfig.HPLimit then
                seer:PetHPLimit()
            else
                seer:PetCure()
            end

            local fight = seer:ReadyFight(42396, function(bs)
                bs:WriteU32List(117, 30, 1)
            end)

            fight:BindDriver(driver)

            if not fight:Start() then
                logger.Warn(string.format("方案 [%s] 挑战失败，请切换战斗方案", fs.MetaData.Name))
                return false
            end
        end
    end

    if not seer:Send(42395, function(bs)
            bs:WriteU32List(117, 4, 0, 0)
        end):IsEmpty() then
        logger.Info(string.format("已领取勇者之塔每日礼包，当前已有%d/200", seer:ItemCount(2300078)))
    end


    return true
end

---@type SeerScript
return {
    Type = "FLASH",
    Name = "勇者之塔",
    Author = "醉星河",
    Desc = "FLASH日常-勇者之塔\n挑战勇者之塔第30层",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddCheckbox("speedrun", "优先扫荡", false)
        cr:AddCheckbox("double", "开启激励", true)
        cr:AddFightScript("path", "战斗方案")
    end
}
