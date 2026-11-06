---@class PetInfo
---@field ID integer 精灵ID（uint32）
---@field UserID integer 玩家ID（uint32）
---@field Generation integer 未知
---@field DV integer 个体值
---@field Nature integer 性格ID
---@field AbilityType integer 特性ID
---@field Level integer 等级
---@field Exp integer 当前经验值
---@field LvExp integer 当前等级经验
---@field NextLvExp integer 下一级所需经验
---@field HP integer 当前生命值
---@field MaxHP integer 最大生命值
---@field Attack integer 攻击力
---@field Defence integer 防御力
---@field SA integer 特殊攻击
---@field SD integer 特殊防御
---@field Speed integer 速度
---@field EvHP integer 学习力：HP
---@field EvAttack integer 学习力：攻击
---@field EvDefence integer 学习力：防御
---@field EvSA integer 学习力：特攻
---@field EvSD integer 学习力：特防
---@field EvSP integer 学习力：速度
---@field Index integer 队伍索引（出战顺序）
---@field SkillInfos PetSkillInfo[] 技能列表（最多5个）
---@field CatchTime integer 捕获时间戳（uint32）
---@field CatchMap integer 捕获地图ID
---@field CatchRect integer 捕获区域
---@field CatchLevel integer 捕获时等级
---@field AbilityMark integer 第一刻印
---@field SkillMark integer 第二刻印
---@field CommonMark integer 第三刻印
---@field CommonMarkActived integer 第三刻印孔是否开启（0/1）
---@field EffectCount integer 效果数量
---@field EffectList PetEffectInfo[] 效果列表
---@field ResistanceInfo PetResistanceInfo 抗性信息
---@field SkinID integer 皮肤ID
---@field AssistMoveID integer 辅助技能ID
---@field BaseCurHP integer 当前基础HP
---@field PveCurHP integer PvE模式当前HP
---@field PvpCurHP integer PvP模式当前HP
---@field PetValueInfos PetValueInfo[] 属性值信息（六维）
PetInfo = {}


---@class PetSkillInfo
---@field ID integer 技能ID（uint32）
---@field PP integer 技能剩余使用次数（Power Point）
PetSkillInfo = {}

---获取技能详细信息
---@return PetSkillMoreInfo
function PetSkillInfo:Info() return {} end

-- ----------------------------
-- 精灵技能详细信息（PetSkillMoreInfo）
-- ----------------------------
---@class PetSkillMoreInfo
---@field ID integer 技能ID
---@field Name string 技能名称
---@field Type integer 技能类型
---@field Power integer 技能威力
---@field MaxPP integer 技能最大使用次数
---@field Accuracy integer 命中率
---@field Priority integer 技能先制等级
SkillInfo = {}

---@class PetEffectInfo
---@field ItemID integer 效果来源道具ID
---@field Status integer 效果状态（启用、禁用等）
---@field LeftCount integer 剩余回合数
---@field EffectID integer 效果ID
---@field Args string 效果附加参数
---@field ArgsAll table<integer, integer> 附加参数总表（map[uint32]uint32）
---@field Args6 string 附加参数（第6个）
---@field AddArray integer[] 加成数值数组
---@field LevelArray integer[] 等级修正数组
PetEffectInfo = {}


---@class PetResistanceInfo
---@field ResistState integer 抗性状态标志位
---@field RedGem integer 红色宝石抗性数值
---@field GreenGem integer 绿色宝石抗性数值
---@field Reserve integer 预留字段
---@field ResistAll integer 总抗性
---@field HurtResistarr integer 伤害减免抗性
---@field EffResistarr integer 异常状态抗性
---@field Cirt integer 暴击概率
---@field CirtAdj integer 暴击修正值
---@field Regular integer 命中率
---@field RegularAdj integer 命中修正值
---@field Precent integer 效果触发率
---@field PrecentAdj integer 效果触发修正值
---@field Ctl1Idx integer 控制类异常1索引
---@field Ctl1 integer 控制类异常1数值
---@field Ctl1Adj integer 控制类异常1修正
---@field Ctl2Idx integer 控制类异常2索引
---@field Ctl2 integer 控制类异常2数值
---@field Ctl2Adj integer 控制类异常2修正
---@field Ctl3Idx integer 控制类异常3索引
---@field Ctl3 integer 控制类异常3数值
---@field Ctl3Adj integer 控制类异常3修正
---@field Weak1Idx integer 弱化类异常1索引
---@field Weak1 integer 弱化类异常1数值
---@field Weak1Adj integer 弱化类异常1修正
---@field Weak2Idx integer 弱化类异常2索引
---@field Weak2 integer 弱化类异常2数值
---@field Weak2Adj integer 弱化类异常2修正
---@field Weak3Idx integer 弱化类异常3索引
---@field Weak3 integer 弱化类异常3数值
---@field Weak3Adj integer 弱化类异常3修正
PetResistanceInfo = {}


---@class PetValueInfo
---@field BaseCur integer 当前基础值
---@field PveCur integer PvE模式当前值
---@field PvpCur integer PvP模式当前值
PetValueInfo = {}
