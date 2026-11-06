---@type SeerMain
function Main(seer, config)
    local body = seer:Send(41464, function(bs) end)

    local dailyTaskId = body:ReadU32()
    body:Skip(8)
    local weekTaskId = body:ReadU32()

    if not seer:Send(41465, function(bs) bs:WriteU32List(1, dailyTaskId) end):IsEmpty() then
        logger.Info(string.format("已领取今日功勋任务4000点数，当前功勋点数%d/99999", seer:ItemCount(1707511)))
    end

    if not seer:Send(41465, function(bs) bs:WriteU32List(2, weekTaskId) end):IsEmpty() then
        logger.Info(string.format("已领取本周功勋任务15000点数，当前功勋点数%d/99999", seer:ItemCount(1707511)))
    end

    return true
end

---@type SeerScript
return {
    Type = "FLASH",
    Name = "太空站功勋任务",
    Author = "醉星河",
    Desc = "FLASH日常-领取功勋奖励\n领取太空站功勋任务的每日奖励和每周奖励",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
    end
}
