-- 多值查询辅助函数（使用 GET_MULTI_FOREVER 46046）
---@param seer Seer
---@param ids integer[]
---@return integer[]
local function multiValue(seer, ids)
    local body = seer:Send(46046, function(bs)
        bs:WriteU32(#ids)
        for _, id in ipairs(ids) do
            bs:WriteU32(id)
        end
    end)
    body:ReadU32() -- skip header
    local results = {}
    for _ = 1, #ids do
        table.insert(results, body:ReadU32())
    end
    return results
end

-- 查询剩余刻印抽奖次数
---@param seer Seer
---@return integer
local function getMarkDrawRemaining(seer)
    local vals = multiValue(seer, { 16577 })
    return math.max(0, 1 - vals[1])
end

-- 查询可许愿次数（基于在线时长）
---@param seer Seer
---@return integer maxTimes
---@return integer remainingTimes
local function getWishTimes(seer)
    local vals = multiValue(seer, { 12462 })
    local totalMinutes = vals[1]
    -- 计算可许愿次数（基于登录时长）
    local maxTimes = 0
    local timesMap = { { 120, 10 }, { 90, 7 }, { 60, 5 }, { 30, 3 }, { 15, 2 }, { 5, 1 }, { 0, 0 } }
    for _, entry in ipairs(timesMap) do
        if totalMinutes >= entry[1] then
            maxTimes = entry[2]
            break
        end
    end
    -- 已需要次数
    local wishedVals = multiValue(seer, { 12231 })
    local remainingTimes = math.max(0, maxTimes - wishedVals[1])
    return maxTimes, remainingTimes
end

-- 查询星愿签到是否可领
---@param seer Seer
---@return boolean
local function canWishSign(seer)
    local vals = multiValue(seer, { 201345 })
    return vals[1] == 0
end

-- 查询星愿签到天数
---@param seer Seer
---@return integer
local function getWishSignDay(seer)
    local vals = multiValue(seer, { 20235 })
    return vals[1]
end

---@type SeerMain
function Main(seer, config)
    -- 刻印抽奖
    if config["mark_draw"] then
        local remaining = getMarkDrawRemaining(seer)
        if remaining > 0 then
            seer:Send(46301, function(bs)
                bs:WriteU32(1):WriteU32(0)
            end)
            logger.Info("刻印抽奖完成")
        else
            logger.Info("刻印抽奖: 今日已抽过")
        end
    end

    -- 许愿
    if config["wish_bottle"] then
        local maxTime, remaining = getWishTimes(seer)
        if remaining > 0 then
            for i = 1, remaining do
                if seer:Send(45801, function(bs)
                        bs:WriteU32(2):WriteU32(1)
                    end):IsEmpty() then
                    break
                end
                logger.Info(string.format("许愿培养 %d/%d 完成", i + maxTime - remaining, maxTime))
            end
        else
            logger.Info("许愿: 无可用次数")
        end
    end

    -- 星愿签到
    if config["wish_sign"] then
        if canWishSign(seer) then
            local day = getWishSignDay(seer)
            seer:Send(45801, function(bs)
                bs:WriteU32(1):WriteU32(day + 1)
            end)
            logger.Info(string.format("星愿签到完成: 第%d天", day + 1))
        else
            logger.Info("星愿签到: 今日已签")
        end
    end

    return true
end

---@type SeerScript
return {
    Type = "FLASH",
    Name = "刻印抽奖+星愿好礼",
    Author = "醉星河",
    Desc = [[FLASH日常-刻印抽奖+星愿好礼+星愿签到]],
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddCheckbox("mark_draw", "刻印抽奖", true)
        cr:AddCheckbox("wish_bottle", "星愿好礼", true)
        cr:AddCheckbox("wish_sign", "星愿签到", true)
    end
}
