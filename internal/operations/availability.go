package operations

// IsDisabled reports release-level CLI availability, independently of API auth.
// The catalog remains a complete protocol reference: disabling CLI access must
// not change ordinary API permissions or the separate Intel News service.
func IsDisabled(method, path string) (bool, error) {
	op, err := Lookup(method, path)
	if err != nil || op == nil {
		return false, err
	}
	switch op.Module {
	case "p2p", "crossex", "coupon", "ann", "launch":
		return true, nil
	case "rewards":
		// Rewards is shared. Only the reviewed welfare/activity operations retire;
		// future unrelated products must not be disabled by a broad URL prefix.
		switch op.OperationID {
		case "getUserIdentity", "getBeginnerTaskList", "claimTask", "claimReward",
			"listActivities", "listActivityTypes", "getMyActivityEntry":
			return true, nil
		}
	}
	return false, nil
}
