package operators

// OpIf реализация тернарного оператора условия.
// При выполнении условия будет возвращен второй переданный аргумент.
func OpIf[T comparable](cond bool, a T, b T) T {
	if cond {
		return a
	}

	return b
}
