package reporting

type Result struct {
	SuccessIDs []string
	FailureIDs []string
}

func SuccessResult(ids ...string) *Result {
	return &Result{
		SuccessIDs: ids,
	}
}

func FailureResult(ids ...string) *Result {
	return &Result{
		FailureIDs: ids,
	}
}

func (r *Result) Append(ar *Result) {
	r.SuccessIDs = append(r.SuccessIDs, ar.SuccessIDs...)
	r.FailureIDs = append(r.FailureIDs, ar.FailureIDs...)
}
