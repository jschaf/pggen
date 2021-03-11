package gotype

// HasCompositeType returns true if t or any of t's descendants (for array and
// composite types) is a composite type.
func HasCompositeType(t Type) bool {
	switch t := t.(type) {
	case CompositeType:
		return true
	case ArrayType:
		return HasCompositeType(t.Elem)
	default:
		return false
	}
}

// HasArrayType returns true if t or any of t's descendants (for array and
// composite types) is an array type.
func HasArrayType(t Type) bool {
	switch t := t.(type) {
	case CompositeType:
		for _, typ := range t.FieldTypes {
			if ok := HasArrayType(typ); ok {
				return true
			}
		}
		return false
	case ArrayType:
		return true
	default:
		return false
	}
}
