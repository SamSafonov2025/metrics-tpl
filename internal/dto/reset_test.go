package dto

import (
	"testing"
)

func TestResetableStruct_Reset(t *testing.T) {
	str := "test string"
	child := &ResetableStruct{
		i:   42,
		str: "child",
	}

	rs := &ResetableStruct{
		i:     10,
		str:   "hello",
		strP:  &str,
		s:     []int{1, 2, 3, 4, 5},
		m:     map[string]string{"key1": "value1", "key2": "value2"},
		child: child,
	}

	// Вызываем Reset()
	rs.Reset()

	// Проверяем примитивы
	if rs.i != 0 {
		t.Errorf("Expected i to be 0, got %d", rs.i)
	}
	if rs.str != "" {
		t.Errorf("Expected str to be empty, got %s", rs.str)
	}

	// Проверяем указатель на примитив
	if rs.strP == nil {
		t.Error("Expected strP to be non-nil")
	} else if *rs.strP != "" {
		t.Errorf("Expected *strP to be empty, got %s", *rs.strP)
	}

	// Проверяем слайс
	if len(rs.s) != 0 {
		t.Errorf("Expected slice length to be 0, got %d", len(rs.s))
	}
	if cap(rs.s) == 0 {
		t.Error("Expected slice capacity to be preserved")
	}

	// Проверяем мапу
	if len(rs.m) != 0 {
		t.Errorf("Expected map length to be 0, got %d", len(rs.m))
	}

	// Проверяем вложенную структуру
	if rs.child == nil {
		t.Error("Expected child to be non-nil")
	} else {
		if rs.child.i != 0 {
			t.Errorf("Expected child.i to be 0, got %d", rs.child.i)
		}
		if rs.child.str != "" {
			t.Errorf("Expected child.str to be empty, got %s", rs.child.str)
		}
	}
}

func TestComplexStruct_Reset(t *testing.T) {
	count := int64(100)
	score := float64(99.5)
	parent := &ComplexStruct{
		ID:   999,
		Name: "parent",
	}

	cs := &ComplexStruct{
		ID:       1,
		Name:     "test",
		Active:   true,
		Tags:     []string{"tag1", "tag2", "tag3"},
		Metadata: map[string]interface{}{"key": "value"},
		Parent:   parent,
		Count:    &count,
		Score:    &score,
	}

	// Вызываем Reset()
	cs.Reset()

	// Проверяем примитивы
	if cs.ID != 0 {
		t.Errorf("Expected ID to be 0, got %d", cs.ID)
	}
	if cs.Name != "" {
		t.Errorf("Expected Name to be empty, got %s", cs.Name)
	}
	if cs.Active != false {
		t.Errorf("Expected Active to be false, got %v", cs.Active)
	}

	// Проверяем слайс
	if len(cs.Tags) != 0 {
		t.Errorf("Expected Tags length to be 0, got %d", len(cs.Tags))
	}

	// Проверяем мапу
	if len(cs.Metadata) != 0 {
		t.Errorf("Expected Metadata length to be 0, got %d", len(cs.Metadata))
	}

	// Проверяем указатели на примитивы
	if cs.Count == nil {
		t.Error("Expected Count to be non-nil")
	} else if *cs.Count != 0 {
		t.Errorf("Expected *Count to be 0, got %d", *cs.Count)
	}

	if cs.Score == nil {
		t.Error("Expected Score to be non-nil")
	} else if *cs.Score != 0 {
		t.Errorf("Expected *Score to be 0, got %f", *cs.Score)
	}

	// Проверяем Parent
	if cs.Parent == nil {
		t.Error("Expected Parent to be non-nil")
	} else {
		if cs.Parent.ID != 0 {
			t.Errorf("Expected Parent.ID to be 0, got %d", cs.Parent.ID)
		}
		if cs.Parent.Name != "" {
			t.Errorf("Expected Parent.Name to be empty, got %s", cs.Parent.Name)
		}
	}
}

func TestSimpleStruct_Reset(t *testing.T) {
	ss := &SimpleStruct{
		Value: 42,
		Label: "test",
	}

	// Вызываем Reset()
	ss.Reset()

	if ss.Value != 0 {
		t.Errorf("Expected Value to be 0, got %d", ss.Value)
	}
	if ss.Label != "" {
		t.Errorf("Expected Label to be empty, got %s", ss.Label)
	}
}

func TestReset_NilReceiver(t *testing.T) {
	var rs *ResetableStruct
	// Не должно паниковать
	rs.Reset()

	var cs *ComplexStruct
	cs.Reset()

	var ss *SimpleStruct
	ss.Reset()
}
