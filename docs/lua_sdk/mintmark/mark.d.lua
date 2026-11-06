---@class MarkAttr
---@field Atk integer 攻击（uint32）
---@field Def integer 防御（uint32）
---@field SpAtk integer 特攻（uint32）
---@field SpDef integer 特防（uint32）
---@field Spd integer 速度（uint32）
---@field Hp integer 体力（uint32）
MarkAttr = {}

---@class MintMark
---@field ID integer 刻印ID（uint32）
---@field ObtainTime integer 刻印获取时间（uint32）
---@field CatchTime integer 绑定精灵获取时间（uint32）
---@field Name string 刻印名称
---@field BindMonID integer 绑定精灵ID（uint32）
---@field BindMoveID integer 绑定技能ID（uint32）
---@field BindGemID integer 绑定宝石ID（uint32）
---@field HoleType integer 是否绑定（uint32）
---@field Attr MarkAttr 刻印属性
---@field ExtraAttr MarkAttr 刻印额外属性
---@field Class integer 刻印系列（uint32）
MintMark = {}

---刻印升级
---@return boolean 是否升级成功
function MintMark:LevelUp() return false end

---刻印一键满级（最多尝试5次升级）
function MintMark:LevelMax() end

---刻印分解（仅泰坦、衡、偃月系列）
---@return boolean 是否分解成功
function MintMark:Resolve() return false end

---重新绑定技能ID
---@param id integer 新技能ID（uint32）
---@return MintMark self 返回新MintMark对象
function MintMark:ReBindMoveID(id) return self end
