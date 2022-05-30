package ptrs

func Int(n int) *int             { return &n }
func Int32(n int32) *int32       { return &n }
func Float64(f float64) *float64 { return &f }
func String(s string) *string    { return &s }
