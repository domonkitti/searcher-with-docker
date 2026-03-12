package data

import (
	"encoding/json"
	"os"
)

type RuleInput struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Type    string  `json:"type"`
	Unit    string  `json:"unit"`
	Default float64 `json:"default"`
}

type RuleItem struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Enabled bool    `json:"enabled"`
	Op      string  `json:"op"` // >, >=, <, <=, ==, !=
	Value   float64 `json:"value"`
}

type RuleConfig struct {
	Inputs          []RuleInput `json:"inputs"`
	Rules           []RuleItem  `json:"rules"`
	BudgetAllTrue   string      `json:"budgetAllTrue"`
	BudgetOtherwise string      `json:"budgetOtherwise"`
	LogicNote       string      `json:"logicNote"`
}

func defaultRuleConfig() RuleConfig {
	return RuleConfig{
		Inputs: []RuleInput{
			{Key: "price", Label: "ราคาต่อชิ้น", Type: "number", Unit: "บาท", Default: 0},
			{Key: "lifespanYears", Label: "อายุการใช้งาน", Type: "number", Unit: "ปี", Default: 0},
		},
		Rules: []RuleItem{
			{Key: "price", Label: "ราคาต่อชิ้น", Enabled: true, Op: ">=", Value: 10000},
			{Key: "lifespanYears", Label: "อายุการใช้งาน", Enabled: true, Op: ">=", Value: 5},
			{Key: "dummy_1", Label: "เงื่อนไขอื่นๆ (dummy)", Enabled: false, Op: ">=", Value: 0},
		},
		BudgetAllTrue:   "ใช้งบลงทุน",
		BudgetOtherwise: "งบดำเนินการ",
		LogicNote:       "wait for more information",
	}
}

func LoadRuleConfig(path string) RuleConfig {
	if _, err := os.Stat(path); err != nil {
		return defaultRuleConfig()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return defaultRuleConfig()
	}
	var cfg RuleConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return defaultRuleConfig()
	}

	d := defaultRuleConfig()
	if len(cfg.Inputs) == 0 {
		cfg.Inputs = d.Inputs
	}
	if len(cfg.Rules) == 0 {
		cfg.Rules = d.Rules
	}
	if cfg.BudgetAllTrue == "" {
		cfg.BudgetAllTrue = d.BudgetAllTrue
	}
	if cfg.BudgetOtherwise == "" {
		cfg.BudgetOtherwise = d.BudgetOtherwise
	}
	if cfg.LogicNote == "" {
		cfg.LogicNote = d.LogicNote
	}
	return cfg
}

func EvalRules(cfg RuleConfig, inputs map[string]float64) (budget string, allTrue bool, conditions []map[string]any) {
	conditions = make([]map[string]any, 0, len(cfg.Rules))
	allTrue = true

	for _, r := range cfg.Rules {
		if !r.Enabled {
			continue
		}
		x := inputs[r.Key]
		ok := compare(x, r.Op, r.Value)
		conditions = append(conditions, map[string]any{
			"key": r.Key, "label": r.Label, "op": r.Op, "value": r.Value, "ok": ok,
		})
		if !ok {
			allTrue = false
		}
	}

	if len(conditions) == 0 {
		allTrue = false
	}

	if allTrue {
		return cfg.BudgetAllTrue, allTrue, conditions
	}
	return cfg.BudgetOtherwise, allTrue, conditions
}

func compare(x float64, op string, v float64) bool {
	switch op {
	case ">":
		return x > v
	case ">=":
		return x >= v
	case "<":
		return x < v
	case "<=":
		return x <= v
	case "==":
		return x == v
	case "!=":
		return x != v
	default:
		return false
	}
}
