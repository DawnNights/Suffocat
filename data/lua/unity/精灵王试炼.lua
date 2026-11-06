---@type SeerMain
function Main(seer, config)
    -- 精灵王ID，默认15
    local elfKingId = ELF_KINGS_LIST[config["elfKing"]]
    if not elfKingId then
        logger.Warn("无效的精灵王ID，请检查配置")
        return false
    end

    -- 获取困难难度解锁情况
    -- elfKingId 1-9 用108105，10-18用108106
    local levelStageId = elfKingId >= 10 and 108106 or 108105
    local levelStage = seer:Send(46046, function(bs) bs:WriteU32(1):WriteU32(levelStageId) end):Skip(4):ReadU32()
    local stageElfId = (elfKingId - 1) % 9 * 3 + 1
    local unlockHard = (math.floor(levelStage / (2 ^ stageElfId)) % 2) == 1

    if not unlockHard then
        logger.Warn(string.format("精灵王-%s未解锁困难难度试炼，无法挑战", config["elfKing"]))
        return false
    end

    -- 获取剩余挑战次数（总6次）
    local remainingTimes = 6 - seer:Send(46046, function(bs) bs:WriteU32(1):WriteU32(18745) end):Skip(4):ReadU32()

    -- 根据配置开关激励
    seer:Send(42395, function(bs)
        if config["double"] then
            bs:WriteU32List(106, 6, 1, 0)
        else
            bs:WriteU32List(106, 6, 0, 0)
        end
    end)

    -- 扫荡或挑战
    if config["speedrun"] then
        for _ = 1, remainingTimes do
            seer:Send(42395, function(bs) bs:WriteU32List(106, 5, elfKingId, 3) end)
        end
    else
        if remainingTimes > 0 then
            local fs = fightscript.LoadLocal(tostring(config["fightscript"]))
            if fs == nil then
                logger.Warn("请设置精灵王的试炼战斗方案")
                return false
            end

            local driver = seer:ExecFightScript(fs)
            if driver == nil then
                logger.Warn(string.format("战斗方案 [%s] 映射失败", fs.MetaData.Name))
                return false
            end

            for _ = 1, remainingTimes do
                if fs.ScriptConfig.HPLimit then
                    seer:PetHPLimit()
                else
                    seer:PetCure()
                end

                local fight = seer:ReadyFight(42396, function(bs)
                    bs:WriteU32List(106, elfKingId, 3)
                end)

                fight:BindDriver(driver)

                if not fight:Start() then
                    logger.Warn(string.format("方案 [%s] 挑战失败", fs.MetaData.Name))
                    return false
                end
            end
        end
    end

    if remainingTimes > 0 then
        logger.Info(string.format(
            "精灵王-%s困难试炼%s完成，当前%s数量: %d/1000",
            config["elfKing"],
            config["speedrun"] and "扫荡" or "挑战",
            ELF_KINGS_GEM_LIST[1721742 + elfKingId],
            seer:ItemCount(1721742 + elfKingId)
        ))
    end

    -- 获取本周挑战次数
    local weeklyChallengeCount = seer:Send(46046, function(bs) bs:WriteU32(1):WriteU32(20134) end):Skip(4):ReadU32()
    if weeklyChallengeCount >= 30 then
        if not seer:Send(42395, function(bs) bs:WriteU32List(106, 3, 0, 0) end):IsEmpty() then
            logger.Info(string.format("已领取精灵王试炼礼包，当前数量%d/200", seer:ItemCount(2300007)))
        end
    else
        logger.Info(string.format("本周精灵王试炼礼包领取进度: %d/30", weeklyChallengeCount))
    end

    return true
end

-- 精灵王列表
ELF_KINGS_LIST = {
    ["火王"] = 1,
    ["光王"] = 2,
    ["冰王"] = 3,
    ["次元王"] = 4,
    ["飞王"] = 5,
    ["龙王"] = 6,
    ["秘王"] = 7,
    ["水王"] = 8,
    ["电王"] = 9,
    ["地王"] = 10,
    ["暗王"] = 11,
    ["超能王"] = 12,
    ["战王"] = 13,
    ["邪灵鸭"] = 14,
    ["草王"] = 15,
    ["普通王"] = 16,
    ["自然王"] = 17,
    ["远古王"] = 18,
}

-- 精灵王兑换晶石列表
ELF_KINGS_GEM_LIST = {
    [1721743] = "盛焱晶石",
    [1721744] = "耀光晶石",
    [1721745] = "极冻晶石",
    [1721746] = "星振晶石",
    [1721747] = "闪逸晶石",
    [1721748] = "龙脉晶石",
    [1721749] = "静谧晶石",
    [1721750] = "碧涛晶石",
    [1721751] = "怒霆晶石",
    [1721752] = "地岩晶石",
    [1721753] = "蚀影晶石",
    [1721754] = "威能晶石",
    [1721755] = "狂战晶石",
    [1721756] = "堕落晶石",
    [1721757] = "复苏晶石",
    [1721758] = "凌绝晶石",
    [1721759] = "生灵晶石",
    [1721760] = "斑驳晶石",
}

---@type SeerScript
return {
    Type = "UNITY",
    Name = "精灵王的试炼",
    Author = "醉星河",
    Desc = "UNITY日常-精灵王的试炼\n挑战困难难度精灵王",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddSelect("elfKing", "挑战精灵王", "草王",
            {
                "火王", "光王", "冰王", "次元王", "飞王", "龙王",
                "秘王", "水王", "电王", "地王", "暗王", "超能王",
                "战王", "邪灵鸭", "草王", "普通王", "自然王", "远古王"
            }
        )
        cr:AddCheckbox("double", "使用激励模组", false)
        cr:AddCheckbox("speedrun", "优先扫荡", false)
        cr:AddFightScript("fightscript", "对战方案")
    end
}
