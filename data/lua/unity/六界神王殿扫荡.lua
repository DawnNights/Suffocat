---@type table<string, number>
ItemName = {}
ItemName[266] = "红宝石"
ItemName[267] = "绿宝石"
ItemName[268] = "泰坦神石"
ItemName[269] = "护佑神石"
ItemName[270] = "宽恕神石"
ItemName[271] = "轮回神石"

---@param seer Seer
---@param id integer
---@param v any
---@diagnostic disable-next-line: lowercase-global
function buy(seer, id, v)
    local count = tonumber(v)
    if count == nil then
        return
    end

    if count < 1 then
        return
    end

    local body = seer:Send(42395, function(bs)
        bs:WriteU32(109):WriteU32(id)
        bs:WriteU32(assert(count)):WriteU32(0)
    end)

    if not body:IsEmpty() then
        logger.Info(string.format(
            "购买 [%s]x%d 成功，剩余界神印记数量: %d",
            ItemName[id],
            count,
            seer:ItemCount(1721558)
        ))
    end
end

---@type SeerMain
function Main(seer, config)
    -- 开启/关闭扫荡激励
    seer:Send(45767, function(bs)
        if config["double"] then
            bs:WriteU32(39):WriteU32(1)
        else
            bs:WriteU32(39):WriteU32(0)
        end
    end)

    -- 扫荡指定难度
    local body = seer:Send(45767, function(bs)
        if config["level"] == "简单" then
            bs:WriteU32(38):WriteU32(1)
        elseif config["level"] == "普通" then
            bs:WriteU32(38):WriteU32(2)
        else
            bs:WriteU32(38):WriteU32(3)
        end
    end)

    if not body:IsEmpty() then
        logger.Info(string.format("六界神王殿%s模式扫荡成功", config["level"]))
    end

    -- 购买指定商品
    buy(seer, 266, config["gem_red"])
    buy(seer, 267, config["gem_green"])
    buy(seer, 268, config["gem_taitan"])
    buy(seer, 269, config["gem_huyou"])
    buy(seer, 270, config["gem_kuanshu"])
    buy(seer, 271, config["gem_lunhui"])

    return true
end

---@type SeerScript
return {
    Type = "UNITY",
    Name = "六界神王殿扫荡",
    Author = "醉星河",
    Desc = "Unity日常-六界神王殿扫荡",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
        cr:AddSelect("level", "一键扫荡难度", "困难", { "简单", "普通", "困难" })
        cr:AddCheckbox("double", "使用激励模组", true)
        cr:AddNumberInput("gem_red", "兑换红宝石", 0, { 0, 999, 1 })
        cr:AddNumberInput("gem_green", "兑换绿宝石", 0, { 0, 999, 1 })
        cr:AddNumberInput("gem_taitan", "兑换泰坦神石", 0, { 0, 99, 1 })
        cr:AddNumberInput("gem_huyou", "兑换护佑神石", 0, { 0, 99, 1 })
        cr:AddNumberInput("gem_kuanshu", "兑换宽恕神石", 0, { 0, 99, 1 })
        cr:AddNumberInput("gem_lunhui", "兑换轮回神石", 0, { 0, 99, 1 })
    end
}
