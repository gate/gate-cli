package earn

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Exercise the actual SDK DTO and reject input before any credential/network work.
func TestAutoInvestAssetTypeParsing(t *testing.T) {
	for _, value := range []int32{1, 2} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			raw := fmt.Sprintf(`{"money":"USDT","asset_type":%d,"items":[{"asset":"BTC","ratio":"70"},{"asset":"ETH","ratio":"30"}]}`, value)
			body, err := parseAutoInvestMinAmount(raw)
			if err != nil {
				t.Fatal(err)
			}
			if body.AssetType != value || body.Money != "USDT" || len(body.Items) != 2 || body.Items[0].Ratio != "70" {
				t.Fatalf("input changed: %+v", body)
			}
			wire, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				AssetType int32 `json:"asset_type"`
			}
			if err = json.Unmarshal(wire, &got); err != nil || got.AssetType != value {
				t.Fatalf("asset_type lost: %s, %v", wire, err)
			}
		})
	}
	for _, raw := range []string{`{}`, `null`, `{`, `{"asset_type":0}`, `{"asset_type":3}`, `{"asset_type":-1}`, `{"asset_type":2147483648}`, `{"asset_type":"1"}`, `{"asset_type":true}`, `{"asset_type":null}`, `{"asset_type":1.5}`, `[]`} {
		t.Run("reject/"+raw, func(t *testing.T) {
			if _, err := parseAutoInvestMinAmount(raw); err == nil {
				t.Fatal("invalid asset_type accepted")
			}
		})
	}
}
