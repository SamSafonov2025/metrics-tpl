package dto

//go:generate go run ../../cmd/reset

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}

// generate:reset
type ComplexStruct struct {
	ID       int
	Name     string
	Active   bool
	Tags     []string
	Metadata map[string]interface{}
	Parent   *ComplexStruct
	Count    *int64
	Score    *float64
}

// generate:reset
type SimpleStruct struct {
	Value int
	Label string
}

// Обычная структура без комментария - не должна генерироваться
type NormalStruct struct {
	Field string
}
