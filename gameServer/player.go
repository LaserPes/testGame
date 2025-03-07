package main

import "time"

type PlayerState struct {
	ID          int       `json:"id"`
	PosX        float64   `json:"posX"`
	PosY        float64   `json:"posY"`
	Nickname    string    `json:"nickname"`
	HeroClass   int       `json:"heroClass"`
	DirectionX  float64   `json:"directionX"`
	DirectionY  float64   `json:"directionY"`
	LastAttack  time.Time `json:"lastAttack"`
	IsAttacking bool      `json:"isAttacking"`
	Health      float64   `json:"health"` //
}
type PlayerMovement struct {
	ID         int     `json:"id"`
	DirectionX float64 `json:"directionX"`
	DirectionY float64 `json:"directionY"`
	MovingX    int     `json:"movingX"`
	MovingY    int     `json:"movingY"`
}
type PlayerAttack struct {
	ID         int     `json:"id"`
	DirectionX float64 `json:"directionX"`
	DirectionY float64 `json:"directionY"`
}

type PlayerData struct {
	HeroClass int    `json:"heroClass"`
	Nickname  string `json:"nickname"`
}

type PlayerClass struct {
	ID                 int     `json:"id"`
	MagicResistance    float64 `json:"magicResistance"`
	PhysicalResistance float64 `json:"physicalResistance"`
	Health             float64 `json:"health"`
	Speed              int     `json:"speed"`
	Attack             float64 `json:"attack"`
	AttackRange        float64 `json:"attackRange"`
	AttackSpeed        int     `json:"attackSpeed"`
	AttackType         string  `json:"attackType"`
}

var WarriorClass = PlayerClass{
	ID:                 1,
	MagicResistance:    0,
	PhysicalResistance: 0.5,
	Health:             150,
	Speed:              600,
	Attack:             25,
	AttackRange:        50,
	AttackSpeed:        300,
	AttackType:         "physical",
}
var MageClass = PlayerClass{
	ID:                 2,
	MagicResistance:    0.3,
	PhysicalResistance: 0,
	Speed:              400,
	Health:             100,
	Attack:             30,
	AttackRange:        200,
	AttackSpeed:        500,
	AttackType:         "magic",
}

var classMap = map[int]PlayerClass{WarriorClass.ID: WarriorClass, MageClass.ID: MageClass}
