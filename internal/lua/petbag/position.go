package petbag

const (
	TYPE_STORAGE     = iota // 精灵仓库
	TYPE_BATTLE_BAG         // 出战背包
	TYPE_SUPPORT_BAG        // 待命背包
	TYPE_OTHER              // 其它位置
)

func typeName(bag_type int) string {
	switch bag_type {
	case TYPE_STORAGE:
		return "精灵仓库"
	case TYPE_BATTLE_BAG:
		return "出战背包"
	case TYPE_SUPPORT_BAG:
		return "待命背包"
	default:
		return "其它位置"
	}
}

// 精灵位置
type PetPosition struct {
	Type  int
	Name  string
	Index uint32
}

func parsePosID(src int, dst int) int {
	switch src {
	case TYPE_BATTLE_BAG:
		switch dst {
		case TYPE_STORAGE:
			return 0 // 出战 -> 仓库
		case TYPE_SUPPORT_BAG:
			return 4 // 出战 -> 待命

		}

	case TYPE_STORAGE:
		switch dst {
		case TYPE_BATTLE_BAG:
			return 1 // 仓库 -> 出战
		case TYPE_SUPPORT_BAG:
			return 2 // 仓库 -> 待命
		}

	case TYPE_SUPPORT_BAG:
		switch dst {
		case TYPE_STORAGE:
			return 3 // 待命 -> 仓库
		case TYPE_BATTLE_BAG:
			return 5 // 待命 -> 出战
		}
	}

	return -1
}

func posName(id uint32) string {
	switch id {
	case 0:
		return "精灵仓库"
	case 1:
		return "出战背包"
	case 2:
		return "出战首发"
	case 3:
		return "基地展示"
	case 4:
		return "精英收藏"
	case 5:
		return "基地展示(精英)"
	case 6:
		return "精灵学院"
	case 7:
		return "待命背包"
	case 8:
		return "守护矿藏"
	case 14:
		return "战队任务派遣"
	case 97:
		return "放生仓库"
	case 98:
		return "精灵模拟训练"
	case 99:
		return "精灵探索"
	default:
		return "未知位置"
	}
}
