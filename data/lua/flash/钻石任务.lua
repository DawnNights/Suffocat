-- 每日钻石任务
---@param seer Seer
function DailyTask(seer)
    -- 领取任务活跃度
    local body = seer:Send(41410, function(bs) bs:WriteU32(1) end)
    body:Skip(4 + 4 + 4 + 4)

    for _ = 1, 11, 1 do
        local taskId = body:ReadU32()
        local curState = body:Skip(4 + 4):ReadU32()

        if curState == 1 or taskId == 11 then -- 可领取
            seer:Send(41411, function(bs) bs:WriteU32List(1, taskId) end)
        end
    end

    -- 领取活跃度宝箱
    body = seer:Send(41410, function(bs) bs:WriteU32(1) end)
    local liveness = body:ReadU32()
    local rewardState = body:ReadU32()

    for i = 0, 5, 1 do
        if liveness > i * 20 and bit.band(rewardState, bit.lshift(1, i)) == 0 then
            seer:Send(41412, function(bs) bs:WriteU32List(1, i + 1) end)
        end
    end

    -- 显示日志
    local n = seer:Send(41410, function(bs) bs:WriteU32(1) end):Skip(4):ReadU32()
    local count = 0
    while n ~= 0 do
        n = bit.band(n, n - 1) -- 清除最低位的1
        count = count + 1
    end

    logger.Info(string.format("今日钻石任务活跃度: %d/100, 已领取奖励: %d/6", liveness, count))
end

-- 每周钻石任务
---@param seer Seer
function WeekTask(seer)
    -- 领取任务活跃度
    local body = seer:Send(41410, function(bs) bs:WriteU32(2) end)
    body:Skip(4 + 4 + 4 + 4)

    for _ = 1, 12, 1 do
        local taskId = body:ReadU32()
        local curState = body:Skip(4 + 4):ReadU32()

        if curState == 1 then -- 可领取
            seer:Send(41411, function(bs) bs:WriteU32List(2, taskId) end)
        end
    end

    -- 领取活跃度宝箱
    body = seer:Send(41410, function(bs) bs:WriteU32(2) end)
    local liveness = body:ReadU32()
    local rewardState = body:ReadU32()

    for i = 0, 5, 1 do
        if liveness > (i * 40) and bit.band(rewardState, bit.lshift(1, i)) == 0 then
            seer:Send(41412, function(bs) bs:WriteU32List(2, i + 101) end)
        end
    end

    -- 显示日志
    local n = seer:Send(41410, function(bs) bs:WriteU32(2) end):Skip(4):ReadU32()
    local count = 0
    while n ~= 0 do
        n = bit.band(n, n - 1) -- 清除最低位的1
        count = count + 1
    end

    logger.Info(string.format("本周钻石任务活跃度: %d/200, 已领取奖励: %d/6", liveness, count))
end

---@type SeerMain
function Main(seer, config)
    DailyTask(seer)
    WeekTask(seer)
    return true
end

---@type SeerScript
return {
    Type = "FLASH",
    Name = "钻石任务",
    Author = "醉星河",
    Desc = "FLASH日常-钻石任务\n领取每日钻石奖励和每周钻石奖励",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr) end
}
