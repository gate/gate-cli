package toolargs

import "errors"

const infoKlineMaxBars = 500

func validateInfoMarkettrendGetKline(arguments map[string]interface{}) error {
	return validateInfoKlineSizeLimit(arguments)
}

func validateInfoMarketdetailGetKline(arguments map[string]interface{}) error {
	return validateInfoKlineSizeLimit(arguments)
}

func validateInfoKlineSizeLimit(arguments map[string]interface{}) error {
	for _, key := range []string{"size", "limit"} {
		if n, ok := intArg(arguments, key); ok && n > infoKlineMaxBars {
			return errors.New("invalid arguments: " + key + " must be <= 500 (use a smaller window to limit agent stdout/token cost)")
		}
	}
	return nil
}
