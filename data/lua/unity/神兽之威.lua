---@param seer Seer 通信对象
---@param speedrun boolean 是否扫荡
---@param driver FightDriver | nil 战斗方案执行器
---@param hplimit boolean 是否压血
---@return boolean
function QingLong(seer, speedrun, driver, hplimit)
    if speedrun then
        local empty = seer:Send(41388, function(bs) bs:WriteU32List(58, 3) end):IsEmpty()
        if empty then
            logger.Warn("神兽之威-青龙，困难难度扫荡失败")
        end
        return not empty
    end

    if driver == nil then
        logger.Debug("神兽之威-青龙，未设置扫荡且未选择战斗方案，已跳过")
        return false
    end

    if hplimit then
        seer:PetHPLimit()
    else
        seer:PetCure()
    end

    local fight = seer:ReadyFight(41129, function(bs) bs:WriteU32(18388) end)
    fight:BindDriver(driver)

    return fight:Start()
end

---@param seer Seer 通信对象
---@param speedrun boolean 是否扫荡
---@param driver FightDriver | nil 战斗方案执行器
---@param hplimit boolean 是否压血
---@return boolean
function BaiHu(seer, speedrun, driver, hplimit)
    if speedrun then
        local empty = seer:Send(41388, function(bs) bs:WriteU32List(58, 6) end):IsEmpty()
        if empty then
            logger.Warn("神兽之威-白虎，困难难度扫荡失败")
        end
        return not empty
    end

    if driver == nil then
        logger.Debug("神兽之威-白虎，未设置扫荡且未选择战斗方案，已跳过")
        return false
    end

    if hplimit then
        seer:PetHPLimit()
    else
        seer:PetCure()
    end

    local fight = seer:ReadyFight(41129, function(bs) bs:WriteU32(18391) end)
    fight:BindDriver(driver)

    return fight:Start()
end

---@param seer Seer 通信对象
---@param speedrun boolean 是否扫荡
---@param driver FightDriver | nil 战斗方案执行器
---@param hplimit boolean 是否压血
---@return boolean
function XuanWu(seer, speedrun, driver, hplimit)
    if speedrun then
        local empty = seer:Send(41388, function(bs) bs:WriteU32List(58, 9) end):IsEmpty()
        if empty then
            logger.Warn("神兽之威-玄武，困难难度扫荡失败")
        end
        return not empty
    end

    if driver == nil then
        logger.Debug("神兽之威-玄武，未设置扫荡且未选择战斗方案，已跳过")
        return false
    end

    if hplimit then
        seer:PetHPLimit()
    else
        seer:PetCure()
    end

    local fight = seer:ReadyFight(41129, function(bs) bs:WriteU32(18394) end)
    fight:BindDriver(driver)

    return fight:Start()
end

---@param seer Seer 通信对象
---@param speedrun boolean 是否扫荡
---@param driver FightDriver | nil 战斗方案执行器
---@param hplimit boolean 是否压血
---@return boolean
function ZhuQue(seer, speedrun, driver, hplimit)
    if speedrun then
        local empty = seer:Send(41388, function(bs) bs:WriteU32List(58, 12) end):IsEmpty()
        if empty then
            logger.Warn("神兽之威-朱雀，困难难度扫荡失败")
        end
        return not empty
    end

    if driver == nil then
        logger.Debug("神兽之威-朱雀，未设置扫荡且未选择战斗方案，已跳过")
        return false
    end

    if hplimit then
        seer:PetHPLimit()
    else
        seer:PetCure()
    end

    local fight = seer:ReadyFight(41129, function(bs) bs:WriteU32(18397) end)
    fight:BindDriver(driver)

    return fight:Start()
end

---@param seer Seer 通信对象
---@param speedrun boolean 是否扫荡
---@param driver FightDriver | nil 战斗方案执行器
---@param hplimit boolean 是否压血
---@return boolean
function QiLing(seer, speedrun, driver, hplimit)
    if speedrun then
        local empty = seer:Send(41388, function(bs) bs:WriteU32List(58, 15) end):IsEmpty()
        if empty then
            logger.Warn("神兽之威-麒麟，困难难度扫荡失败")
        end
        return not empty
    end

    if driver == nil then
        logger.Debug("神兽之威-麒麟，未设置扫荡且未选择战斗方案，已跳过")
        return false
    end

    if hplimit then
        seer:PetHPLimit()
    else
        seer:PetCure()
    end

    local fight = seer:ReadyFight(41129, function(bs) bs:WriteU32(18400) end)
    fight:BindDriver(driver)

    return fight:Start()
end

---@type SeerMain
function Main(seer, config)
    local body = seer:Send(45866, function(bs)
        bs:WriteU32List(0, 0, 6, 121555, 201512, 201513, 201514, 201515, 201516)
    end):Skip(16)

    local qinglongTimes = 5 - body:ReadU32()
    local baihuTimes = 5 - body:ReadU32()
    local xuanwuTimes = 5 - body:ReadU32()
    local zhuqueTimes = 5 - body:ReadU32()
    local qilingTimes = 5 - body:ReadU32()

    local allZero = (qinglongTimes == 0) and (baihuTimes == 0) and (xuanwuTimes == 0) and (zhuqueTimes == 0) and
    (qilingTimes == 0)
    if allZero then
        logger.Info("今日神兽之威已经完成")
        return true
    else
        logger.Info(string.format(
            "今日剩余挑战次数: 青龙-%d | 白虎-%d | 玄武-%d | 朱雀-%d | 麒麟-%d",
            qinglongTimes, baihuTimes, xuanwuTimes, zhuqueTimes, qilingTimes
        ))
    end

    GeneralDriver = nil
    local needHpLimit = false
    if not config["qinglong_speedrun"] or not config["baihu_speedrun"] or not config["xuanwu_speedrun"] or not config["zhuque_speedrun"] or not config["qiling_speedrun"] then
        local fs = fightscript.LoadLocal(tostring(config["fightscript"]))
        if fs ~= nil then
            GeneralDriver = seer:ExecFightScript(fs)
            needHpLimit = fs.ScriptConfig.HPLimit
            if GeneralDriver == nil then
                logger.Warn(string.format("战斗方案 [%s] 映射失败", fs.MetaData.Name))
                return false
            end
        end
    end

    seer:Send(41388, function(bs)
        if config["double"] then bs:WriteU32List(58, 21) else bs:WriteU32List(58, 20) end
    end)

    for _ = 1, qinglongTimes, 1 do
        if not QingLong(seer, config["qinglong_speedrun"] == true, GeneralDriver, needHpLimit) then
            break
        end
    end

    for _ = 1, baihuTimes, 1 do
        if not BaiHu(seer, config["baihu_speedrun"] == true, GeneralDriver, needHpLimit) then
            break
        end
    end

    for _ = 1, xuanwuTimes, 1 do
        if not XuanWu(seer, config["xuanwu_speedrun"] == true, GeneralDriver, needHpLimit) then
            break
        end
    end

    for _ = 1, zhuqueTimes, 1 do
        if not ZhuQue(seer, config["zhuque_speedrun"] == true, GeneralDriver, needHpLimit) then
            break
        end
    end

    for _ = 1, qilingTimes, 1 do
        if not QiLing(seer, config["qiling_speedrun"] == true, GeneralDriver, needHpLimit) then
            break
        end
    end

    return true
end

---@type SeerScript
return {
    Type = "UNITY",
    Name = "神兽之威",
    Author = "醉星河",
    Desc = "Unity日常-神兽之威困难难度",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddCheckbox("double", "挑战激励", true)
        cr:AddFightScript("fightscript", "挑战方案")
        cr:AddCheckbox("qinglong_speedrun", "青龙-优先扫荡", false)
        cr:AddCheckbox("baihu_speedrun", "白虎-优先扫荡", false)
        cr:AddCheckbox("xuanwu_speedrun", "玄武-优先扫荡", false)
        cr:AddCheckbox("zhuque_speedrun", "朱雀-优先扫荡", false)
        cr:AddCheckbox("qiling_speedrun", "麒麟-优先扫荡", false)
    end
}
