package debug

import (
    "encoding/json"
    "fmt"
)

// 1. Define a struct using a Generic Type Parameter [T]
type Processor[T any] struct {
	Identifier string
	Payload    T // This can hold *appsv1alpha.DemoApp or any other type
}

// 2. Create an initialization function that accepts the generic type
func NewProcessor[T any](id string, data T) *Processor[T] {
	return &Processor[T]{
		Identifier: id,
		Payload:    data,
	}
}

// 3. Methods can handle the data generically
func (p *Processor[T]) Debug_App() {
	// You can print it or pass it to reflection engines
	b, err := json.MarshalIndent(p.Payload, "", "  ")
    if err != nil {
        fmt.Println("error:", err)
    }

	fmt.Printf("ID: %s, Debug: %v\n", p.Identifier, string(b))
}

