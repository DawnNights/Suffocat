---@type SeerMain
function Main(seer, config)
    -- 获取今日已挑战次数
    local body = seer:Send(46046, function(bs)
        bs:WriteU32(2):WriteU32(18751):WriteU32(18752)
    end)

    body:ReadU32()
    local normalTimes = 9 - body:ReadU32() -- 普通模式剩余挑战次数
    local hardTimes = 4 - body:ReadU32()   -- 困难模式剩余挑战次数

    if normalTimes == 0 and hardTimes == 0 then
        logger.Info("今日SPT帕诺星系已完成")
    else
        logger.Info(string.format("当前SPT普通模式可挑战次数: %d, 困难模式可挑战次数: %d", normalTimes, hardTimes))
    end

    -- 进行普通模式扫荡
    seer:Send(42395, function(bs)
        bs:WriteU32(111):WriteU32(1):WriteU32(1):WriteU32(1)
    end)

    for _ = 1, normalTimes, 1 do
        seer:Send(42395, function(bs)
            if config["normalMode"] == "普通1-1" then
                bs:WriteU32(111):WriteU32(6):WriteU32(101):WriteU32(1)
            else
                bs:WriteU32(111):WriteU32(6):WriteU32(102):WriteU32(5)
            end
        end)
    end

    -- 进行困难模式扫荡
    if config["hardMode"] then
        seer:Send(42395, function(bs)
            bs:WriteU32(111):WriteU32(1):WriteU32(1):WriteU32(2)
        end)

        for _ = 1, hardTimes, 1 do
            seer:Send(42395, function(bs)
                bs:WriteU32(111):WriteU32(6):WriteU32(101):WriteU32(1)
            end)
        end
    end

    if config["item1"] > 0 and config["item1"] <= 5 then
        if not seer:Send(42395, function(bs) bs:WriteU32List(109, 617, config["item1"], 0) end):IsEmpty() then
            logger.Info(string.format("已购买长效体力药剂x%d, 当前剩余SPT奖章: %d", config["item1"], seer:ItemCount(1725115)))
        end
    end

    if config["item2"] > 0 then
        if not seer:Send(42395, function(bs) bs:WriteU32List(109, 90, config["item2"], 0) end):IsEmpty() then
            logger.Info(string.format("已购买减伤能量珠x%d, 当前剩余先锋旗帜: %d", config["item2"], seer:ItemCount(1722506)))
        else
            logger.Warn(string.format("购买减伤能量珠x%d失败", config["item2"]))
        end
    end

    if config["item3"] > 0 then
        if not seer:Send(42395, function(bs) bs:WriteU32List(109, 92, config["item3"], 0) end):IsEmpty() then
            logger.Info(string.format("已购买不灭能量珠药剂x%d, 当前剩余先锋旗帜: %d", config["item3"], seer:ItemCount(1722506)))
        else
            logger.Warn(string.format("购买不灭能量珠x%d失败", config["item3"]))
        end
    end

    return true
end

---@type SeerScript
return {
    Type = "UNITY",
    Name = "SPT帕诺星系",
    Author = "醉星河",
    Desc = "Unity日常-SPT帕诺星系\n普通1-1：SPT奖章x45; 普通1-4-1：先锋旗帜x270\n困难1-1：SPT奖章x40",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddSelect("normalMode", "普通模式", "普通1-1", { "普通1-1", "普通1-4-1" })
        cr:AddCheckbox("hardMode", "困难1-1", true)
        cr:AddNumberInput("item1", "SPT奖章兑换-长效体力药剂", 0, { 0, 5, 1 })
        cr:AddNumberInput("item2", "先锋旗帜兑换-减伤能量珠Ω", 0, { 0, 50, 1 })
        cr:AddNumberInput("item3", "先锋旗帜兑换-不灭能量珠", 0, { 0, 50, 1 })
    end
}
