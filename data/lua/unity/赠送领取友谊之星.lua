---@type SeerMain
function Main(seer, config)
    local body = seer:Send(47334, function(_) end)

    local body2 = seer:Send(47348, function(bs)
        local friendNum = body:Skip(8):ReadU32()
        if friendNum > 20 then
            friendNum = 20
        end

        bs:WriteU32List(4, 0, friendNum)
        for _ = 1, friendNum, 1 do
            bs:WriteU32(body:ReadU32())
            body:Skip(16 + 4 + 4 + 4 + 4 + 4 + 4 + 1 + 4)
        end
    end)

    if body2:IsEmpty() then
        logger.Info("今日已赠送并领取过友谊之星")
    else
        local starNum = body2:ReadU32()
        local friendNum = body2:ReadU32()
        logger.Info(string.format("已成功向%d位好友赠送并收获%d颗友谊之星，当前友谊之星数量%d/100", friendNum, starNum, seer:ItemCount(1400154)))
    end

    return true
end

---@type SeerScript
return {
    Type = "UNITY",
    Name = "赠送领取友谊之星",
    Author = "醉星河",
    Desc = "UNITY日常-赠送领取友谊之星",
    Version = "0.0.1",
    Main = Main,
    Setup = function(cr)
    end
}
