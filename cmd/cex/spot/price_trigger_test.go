package spot

import "testing"

// The SDK declares these two policies; actual acceptance remains server-side.
func TestSpotTriggerTimeInForce(t *testing.T) {
	for _, value := range []string{"gtc", "ioc"} {
		if err := validateSpotTriggerTimeInForce(value); err != nil {
			t.Errorf("valid %q: %v", value, err)
		}
	}
	for _, value := range []string{"", "GTC", "fok", "poc", " gtc ", "invalid"} {
		if err := validateSpotTriggerTimeInForce(value); err == nil {
			t.Errorf("accepted invalid %q", value)
		}
	}
}

// Default to the policy proven usable by the test service, rather than an empty/gtc value.
func TestSpotTriggerDefaultTimeInForce(t *testing.T) {
	create, _, err := priceTriggerCmd.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	flag := create.Flags().Lookup("time-in-force")
	if flag == nil || flag.DefValue != "ioc" {
		t.Fatal("trigger command must default time-in-force to ioc")
	}
}
