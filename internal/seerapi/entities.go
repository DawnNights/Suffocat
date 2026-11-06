package seerapi

import "database/sql"

type stores struct {
	pets          *tableStore[Pet]
	skills        *tableStore[Skill]
	items         *tableStore[Item]
	mintmarks     *tableStore[MintMark]
	suits         *tableStore[Suit]
	equips        *tableStore[Equip]
	factors       *tableStore[Factor]
	factorDesigns *tableStore[FactorDesign]
}

func newStores(db *sql.DB) *stores {
	return &stores{
		pets:          newPetStore(db),
		skills:        newSkillStore(db),
		items:         newItemStore(db),
		mintmarks:     newMintMarkStore(db),
		suits:         newSuitStore(db),
		equips:        newEquipStore(db),
		factors:       newFactorStore(db),
		factorDesigns: newFactorDesignStore(db),
	}
}

func newPetStore(db *sql.DB) *tableStore[Pet] {
	return &tableStore[Pet]{
		db:    db,
		table: "pets",
		columns: []string{
			"id", "name", "resource_id", "catch_rate", "type_id", "gender",
			"atk", "def", "sp_atk", "sp_def", "spd", "hp",
		},
		idFn: func(v Pet) uint32 { return v.ID },
		scanFn: func(r rowScanner) (Pet, error) {
			var v Pet
			err := r.Scan(
				&v.ID, &v.Name, &v.ResourceID, &v.CatchRate, &v.Type, &v.Gender,
				&v.BaseStats.Atk, &v.BaseStats.Def, &v.BaseStats.SpAtk,
				&v.BaseStats.SpDef, &v.BaseStats.Spd, &v.BaseStats.HP,
			)
			return v, err
		},
		valuesFn: func(v Pet) []any {
			return []any{
				v.ID, v.Name, v.ResourceID, v.CatchRate, v.Type, v.Gender,
				v.BaseStats.Atk, v.BaseStats.Def, v.BaseStats.SpAtk,
				v.BaseStats.SpDef, v.BaseStats.Spd, v.BaseStats.HP,
			}
		},
		children: []childTable[Pet]{
			{
				table:        "pet_skills",
				parentColumn: "pet_id",
				columns:      []string{"skill_id", "level", "special", "advanced", "fifth", "position"},
				valuesFn: func(v Pet) [][]any {
					rows := make([][]any, 0, len(v.Skills))
					for i, skill := range v.Skills {
						rows = append(rows, []any{
							skill.ID, skill.Level, skill.Special, skill.Advanced, skill.Fifth, i,
						})
					}
					return rows
				},
				load: func(r rowScanner, v *Pet) error {
					var skill PetSkill
					var position int
					if err := r.Scan(&skill.ID, &skill.Level, &skill.Special, &skill.Advanced, &skill.Fifth, &position); err != nil {
						return err
					}
					v.Skills = append(v.Skills, skill)
					return nil
				},
			},
			{
				table:        "pet_soulmarks",
				parentColumn: "pet_id",
				columns:      []string{"soulmark_id", "position"},
				valuesFn: func(v Pet) [][]any {
					rows := make([][]any, 0, len(v.SoulmarkIDs))
					for i, id := range v.SoulmarkIDs {
						rows = append(rows, []any{id, i})
					}
					return rows
				},
				load: func(r rowScanner, v *Pet) error {
					var id uint32
					var position int
					if err := r.Scan(&id, &position); err != nil {
						return err
					}
					v.SoulmarkIDs = append(v.SoulmarkIDs, id)
					return nil
				},
			},
		},
	}
}

func newSkillStore(db *sql.DB) *tableStore[Skill] {
	return &tableStore[Skill]{
		db:    db,
		table: "skills",
		columns: []string{
			"id", "name", "category_id", "type_id", "power", "max_pp",
			"accuracy", "priority", "crit_rate", "must_hit",
		},
		idFn: func(v Skill) uint32 { return v.ID },
		scanFn: func(r rowScanner) (Skill, error) {
			var v Skill
			err := r.Scan(
				&v.ID, &v.Name, &v.Category, &v.Type, &v.Power, &v.MaxPP,
				&v.Accuracy, &v.Priority, &v.CritRate, &v.MustHit,
			)
			return v, err
		},
		valuesFn: func(v Skill) []any {
			return []any{
				v.ID, v.Name, v.Category, v.Type, v.Power, v.MaxPP,
				v.Accuracy, v.Priority, v.CritRate, v.MustHit,
			}
		},
	}
}

func newItemStore(db *sql.DB) *tableStore[Item] {
	return &tableStore[Item]{
		db:      db,
		table:   "items",
		columns: []string{"id", "name", "description"},
		idFn:    func(v Item) uint32 { return v.ID },
		scanFn: func(r rowScanner) (Item, error) {
			var v Item
			err := r.Scan(&v.ID, &v.Name, &v.Desc)
			return v, err
		},
		valuesFn: func(v Item) []any {
			return []any{v.ID, v.Name, v.Desc}
		},
	}
}

func newMintMarkStore(db *sql.DB) *tableStore[MintMark] {
	return &tableStore[MintMark]{
		db:    db,
		table: "mintmarks",
		columns: []string{
			"id", "name", "description", "class_id",
			"extra_atk", "extra_def", "extra_sp_atk", "extra_sp_def", "extra_spd", "extra_hp",
		},
		idFn: func(v MintMark) uint32 { return v.ID },
		scanFn: func(r rowScanner) (MintMark, error) {
			var v MintMark
			err := r.Scan(
				&v.ID, &v.Name, &v.Desc, &v.ClassID,
				&v.ExtraAttr.Atk, &v.ExtraAttr.Def, &v.ExtraAttr.SpAtk,
				&v.ExtraAttr.SpDef, &v.ExtraAttr.Spd, &v.ExtraAttr.HP,
			)
			return v, err
		},
		valuesFn: func(v MintMark) []any {
			return []any{
				v.ID, v.Name, v.Desc, v.ClassID,
				v.ExtraAttr.Atk, v.ExtraAttr.Def, v.ExtraAttr.SpAtk,
				v.ExtraAttr.SpDef, v.ExtraAttr.Spd, v.ExtraAttr.HP,
			}
		},
	}
}

func newSuitStore(db *sql.DB) *tableStore[Suit] {
	return &tableStore[Suit]{
		db:      db,
		table:   "suits",
		columns: []string{"id", "name"},
		idFn:    func(v Suit) uint32 { return v.ID },
		scanFn: func(r rowScanner) (Suit, error) {
			var v Suit
			err := r.Scan(&v.ID, &v.Name)
			return v, err
		},
		valuesFn: func(v Suit) []any {
			return []any{v.ID, v.Name}
		},
		children: []childTable[Suit]{
			{
				table:        "suit_equips",
				parentColumn: "suit_id",
				columns:      []string{"equip_id", "position"},
				valuesFn: func(v Suit) [][]any {
					rows := make([][]any, 0, len(v.EquipIDs))
					for i, id := range v.EquipIDs {
						rows = append(rows, []any{id, i})
					}
					return rows
				},
				load: func(r rowScanner, v *Suit) error {
					var id uint32
					var position int
					if err := r.Scan(&id, &position); err != nil {
						return err
					}
					v.EquipIDs = append(v.EquipIDs, id)
					return nil
				},
			},
		},
	}
}

func newEquipStore(db *sql.DB) *tableStore[Equip] {
	return &tableStore[Equip]{
		db:      db,
		table:   "equips",
		columns: []string{"id", "name", "part"},
		idFn:    func(v Equip) uint32 { return v.ID },
		scanFn: func(r rowScanner) (Equip, error) {
			var v Equip
			err := r.Scan(&v.ID, &v.Name, &v.Part)
			return v, err
		},
		valuesFn: func(v Equip) []any {
			return []any{v.ID, v.Name, v.Part}
		},
	}
}

func newFactorStore(db *sql.DB) *tableStore[Factor] {
	return &tableStore[Factor]{
		db:    db,
		table: "factors",
		columns: []string{
			"id", "name", "pet_limit", "pet_consume", "skill_consume",
			"soulmark_consume", "item_id", "design_id", "need_pet_id",
			"pet_id", "skill_id", "soulmark_id",
		},
		idFn: func(v Factor) uint32 { return v.ID },
		scanFn: func(r rowScanner) (Factor, error) {
			var v Factor
			err := r.Scan(
				&v.ID, &v.Name, &v.PetLimit, &v.PetConsume, &v.SkillConsume,
				&v.SoulmarkConsume, &v.ItemID, &v.DesignID, &v.NeedPetID,
				&v.PetID, &v.SkillID, &v.SoulmarkID,
			)
			return v, err
		},
		valuesFn: func(v Factor) []any {
			return []any{
				v.ID, v.Name, v.PetLimit, v.PetConsume, v.SkillConsume,
				v.SoulmarkConsume, v.ItemID, v.DesignID, v.NeedPetID,
				v.PetID, v.SkillID, v.SoulmarkID,
			}
		},
	}
}

func newFactorDesignStore(db *sql.DB) *tableStore[FactorDesign] {
	return &tableStore[FactorDesign]{
		db:    db,
		table: "factor_designs",
		columns: []string{
			"id", "times", "time_value", "progress_value", "fail_times",
			"achievement_id", "reward_id", "mintmark_reward_id",
		},
		idFn: func(v FactorDesign) uint32 { return v.ID },
		scanFn: func(r rowScanner) (FactorDesign, error) {
			var v FactorDesign
			err := r.Scan(
				&v.ID, &v.Times, &v.TimeValue, &v.ProgressValue, &v.FailTimes,
				&v.AchievementID, &v.RewardID, &v.MintmarkRewardID,
			)
			return v, err
		},
		valuesFn: func(v FactorDesign) []any {
			return []any{
				v.ID, v.Times, v.TimeValue, v.ProgressValue, v.FailTimes,
				v.AchievementID, v.RewardID, v.MintmarkRewardID,
			}
		},
		children: []childTable[FactorDesign]{
			{
				table:        "factor_difficulties",
				parentColumn: "design_id",
				columns:      []string{"position", "level", "reward_count"},
				valuesFn: func(v FactorDesign) [][]any {
					rows := make([][]any, 0, len(v.Difficulties))
					for i, difficulty := range v.Difficulties {
						rows = append(rows, []any{i, difficulty.Level, difficulty.RewardCount})
					}
					return rows
				},
				load: func(r rowScanner, v *FactorDesign) error {
					var position int
					var difficulty FactorDifficulty
					if err := r.Scan(&position, &difficulty.Level, &difficulty.RewardCount); err != nil {
						return err
					}
					v.Difficulties = append(v.Difficulties, difficulty)
					return nil
				},
			},
			{
				table:        "factor_tasks",
				parentColumn: "design_id",
				orderBy:      "difficulty, position",
				columns:      []string{"difficulty", "position", "description", "is_single_pet", "boss_pet_id"},
				valuesFn: func(v FactorDesign) [][]any {
					var rows [][]any
					for i, difficulty := range v.Difficulties {
						for j, task := range difficulty.Tasks {
							rows = append(rows, []any{i, j, task.Desc, task.IsSinglePet, task.BossPetID})
						}
					}
					return rows
				},
				load: func(r rowScanner, v *FactorDesign) error {
					var difficulty, position int
					var task FactorTask
					if err := r.Scan(&difficulty, &position, &task.Desc, &task.IsSinglePet, &task.BossPetID); err != nil {
						return err
					}
					if difficulty >= 0 && difficulty < len(v.Difficulties) {
						v.Difficulties[difficulty].Tasks = append(v.Difficulties[difficulty].Tasks, task)
					}
					return nil
				},
			},
			{
				table:        "factor_rules",
				parentColumn: "design_id",
				orderBy:      "difficulty, position",
				columns:      []string{"difficulty", "position", "rule_id", "tips", "fail_tips"},
				valuesFn: func(v FactorDesign) [][]any {
					var rows [][]any
					for i, difficulty := range v.Difficulties {
						for j, rule := range difficulty.Rules {
							rows = append(rows, []any{i, j, rule.ID, rule.Tips, rule.FailTips})
						}
					}
					return rows
				},
				load: func(r rowScanner, v *FactorDesign) error {
					var difficulty, position int
					var rule FactorRule
					if err := r.Scan(&difficulty, &position, &rule.ID, &rule.Tips, &rule.FailTips); err != nil {
						return err
					}
					if difficulty >= 0 && difficulty < len(v.Difficulties) {
						v.Difficulties[difficulty].Rules = append(v.Difficulties[difficulty].Rules, rule)
					}
					return nil
				},
			},
			{
				table:        "factor_banned_pets",
				parentColumn: "design_id",
				orderBy:      "difficulty, position",
				columns:      []string{"difficulty", "position", "pet_id"},
				valuesFn: func(v FactorDesign) [][]any {
					var rows [][]any
					for i, difficulty := range v.Difficulties {
						for j, petID := range difficulty.BannedPets {
							rows = append(rows, []any{i, j, petID})
						}
					}
					return rows
				},
				load: func(r rowScanner, v *FactorDesign) error {
					var difficulty, position int
					var petID uint32
					if err := r.Scan(&difficulty, &position, &petID); err != nil {
						return err
					}
					if difficulty >= 0 && difficulty < len(v.Difficulties) {
						v.Difficulties[difficulty].BannedPets = append(v.Difficulties[difficulty].BannedPets, petID)
					}
					return nil
				},
			},
		},
	}
}
