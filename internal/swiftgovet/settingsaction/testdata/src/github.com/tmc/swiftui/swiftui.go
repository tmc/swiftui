package swiftui

type IntState struct{}

func NewIntState(int) *IntState { return &IntState{} }

func (s *IntState) Get() int { return 0 }

func (s *IntState) Set(int) {}

func (s *IntState) SetAnimatedWith(int, int) {}

func Button(string, func()) any { return nil }
