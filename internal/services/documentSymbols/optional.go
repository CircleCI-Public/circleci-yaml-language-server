package documentSymbols

// unlessZero leaves a zero value out of a symbol, as it always has been,
// rather than sending it as "" or false.
func unlessZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
