package main

type PlayerState struct {
	ID   int     `json:"id"`
	PosX float64 `json:"posX"`
	PosY float64 `json:"posY"`
	// pos  pixel.Vec `json:"pos"`
	// speed        float64        `json:"speed"`
	// radius       float64        `json:"radius"`
	// imd          *imdraw.IMDraw `json:"-"` // Skip serialization
	// bounds       pixel.Rect     `json:"-"` // Skip serialization
	Nickname   string  `json:"nickname"`
	HeroClass  int     `json:"heroClass"`
	DirectionX float64 `json:"directionX"`
	DirectionY float64 `json:"directionY"`
	// projectiles  []*Projectile  `json:"-"` // Skip serialization
	LastAttack  float64 `json:"lastAttack"`
	IsAttacking bool    `json:"isAttacking"`
	// meleeEffects []*MeleeEffect `json:"-"`      // Skip serialization
	// explosions   []*Explosion   `json:"-"`      // Skip serialization
	Health int `json:"health"` //
}
type PlayerMovement struct {
	ID         int     `json:"id"`
	DirectionX float64 `json:"directionX"`
	DirectionY float64 `json:"directionY"`
	// HeroClass  int     `json:"heroClass"`
	// Nickname   string  `json:"nickname"`
	MovingX int `json:"movingX"`
	MovingY int `json:"movingY"`
}
type PlayerAttack struct {
	ID         int     `json:"id"`
	DirectionX float64 `json:"directionX"`
	DirectionY float64 `json:"directionY"`
	Nickname   string  `json:"nickname"`
}
type PlayerClass struct {
	ID                 int     `json:"id"`
	MagicResistance    float64 `json:"magicResistance"`
	PhysicalResistance float64 `json:"physicalResistance"`
	Health             int     `json:"health"`
	Speed              int     `json:"speed"`
	Attack             int     `json:"attack"`
	AttackRange        float64 `json:"attackRange"`
	AttackSpeed        int     `json:"attackSpeed"`
	AttackType         string  `json:"attackType"`
}

var WarriorClass = PlayerClass{
	ID:                 1,
	MagicResistance:    0,
	PhysicalResistance: 0.5,
	Health:             150,
	Attack:             20,
	AttackRange:        50,
	AttackSpeed:        500,
	AttackType:         "physical",
}
var MageClass = PlayerClass{
	ID:                 2,
	MagicResistance:    0.3,
	PhysicalResistance: 0,
	Health:             100,
	Attack:             30,
	AttackRange:        200,
	AttackSpeed:        200,
	AttackType:         "magic",
}

var classMap = map[int]PlayerClass{WarriorClass.ID: WarriorClass, MageClass.ID: MageClass}
