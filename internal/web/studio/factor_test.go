package studio

import (
	"testing"

	"suffocat/internal/seerapi"
)

func testFactorDesign() *seerapi.FactorDesign {
	return &seerapi.FactorDesign{
		ID: 5,
		Difficulties: []seerapi.FactorDifficulty{
			{
				Level:       "easy",
				RewardCount: 12,
				Tasks: []seerapi.FactorTask{
					{Desc: "【简单敌人一】"},
					{Desc: "【简单敌人二】"},
				},
			},
			{
				Level:       "normal",
				RewardCount: 18,
				Tasks: []seerapi.FactorTask{
					{Desc: "【普通敌人一】"},
					{Desc: "【普通敌人二】"},
					{Desc: "【普通敌人三】"},
				},
				Rules: []seerapi.FactorRule{
					{ID: 2, Tips: "六回合内击败"},
					{ID: 1, Tips: "不可重复上阵"},
				},
			},
			{
				Level:       "hard",
				RewardCount: 36,
				Tasks: []seerapi.FactorTask{
					{Desc: "【困难敌人一】"},
				},
			},
		},
	}
}

func TestBuildFactorDesc(t *testing.T) {
	design := testFactorDesign()

	got := buildFactorDesc(design, 2)
	want := "1. 六回合内击败\n2. 不可重复上阵\n\n第 1 关【普通敌人一】\n第 2 关【普通敌人二】\n第 3 关【普通敌人三】"
	if got != want {
		t.Fatalf("说明文本不一致:\n got=%q\nwant=%q", got, want)
	}

	if desc := buildFactorDesc(design, 3); desc != "第 1 关【困难敌人一】" {
		t.Fatalf("困难难度说明错误: %q", desc)
	}
}

func TestNormalizedFactorSettings(t *testing.T) {
	design := testFactorDesign()

	settings := normalizedFactorSettings(design, nil)
	if settings.Mode != 1 || settings.FailTimes != 10 {
		t.Fatalf("默认设置错误: %+v", settings)
	}
	if len(settings.Stages) != 2 {
		t.Fatalf("简单难度关卡数量错误: %d", len(settings.Stages))
	}
	if settings.Stages[0].Port != "UNITY" {
		t.Fatalf("默认端口错误: %+v", settings.Stages[0])
	}

	saved := &FactorSettings{
		Mode:      3,
		Speedrun:  true,
		FailTimes: 7,
		Stages: []FactorStageConfig{
			{Port: "FLASH", Plan: "a.yaml"},
			{Port: "UNITY", Plan: "b.yaml"},
		},
	}
	settings = normalizedFactorSettings(design, saved)
	if settings.Mode != 3 || !settings.Speedrun || settings.FailTimes != 7 {
		t.Fatalf("保存设置未生效: %+v", settings)
	}
	if len(settings.Stages) != 1 || settings.Stages[0].Port != "FLASH" || settings.Stages[0].Plan != "a.yaml" {
		t.Fatalf("困难难度关卡设置错误: %+v", settings.Stages)
	}
}

func TestFactorName(t *testing.T) {
	factor := &seerapi.Factor{ID: 2400021, Name: "墨纪拉因子"}
	if name := factorName(factor); name != "墨纪拉" {
		t.Fatalf("因子名称错误: %q", name)
	}
	if name := factorName(nil); name != "" {
		t.Fatalf("空因子名称错误: %q", name)
	}
}
