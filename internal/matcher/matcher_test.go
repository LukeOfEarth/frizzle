package matcher

import (
	"encoding/json"
	"testing"
)

func TestMatches(t *testing.T) {
	tests := []struct {
		name    string
		event   string
		pattern string
		want    bool
	}{
		{
			name:    "exact source match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":["myapp.orders"]}`,
			want:    true,
		},
		{
			name:    "exact source mismatch",
			event:   `{"source":"myapp.users","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":["myapp.orders"]}`,
			want:    false,
		},
		{
			name:    "prefix match",
			event:   `{"source":"myapp.orders.created","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":[{"prefix":"myapp.orders"}]}`,
			want:    true,
		},
		{
			name:    "prefix mismatch",
			event:   `{"source":"myapp.users.login","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":[{"prefix":"myapp.orders"}]}`,
			want:    false,
		},
		{
			name:    "multiple matchers OR logic",
			event:   `{"source":"myapp.users","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":[{"prefix":"myapp.orders"},"myapp.users"]}`,
			want:    true,
		},
		{
			name:    "exists true",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"orderId":"123"}}`,
			pattern: `{"detail":{"orderId":[{"exists":true}]}}`,
			want:    true,
		},
		{
			name:    "exists false",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"detail":{"orderId":[{"exists":false}]}}`,
			want:    true,
		},
		{
			name:    "exists false but field present",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"orderId":"123"}}`,
			pattern: `{"detail":{"orderId":[{"exists":false}]}}`,
			want:    false,
		},
		{
			name:    "numeric greater than",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"amount":150}}`,
			pattern: `{"detail":{"amount":[{"numeric":[">",100]}]}}`,
			want:    true,
		},
		{
			name:    "numeric not greater than",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"amount":50}}`,
			pattern: `{"detail":{"amount":[{"numeric":[">",100]}]}}`,
			want:    false,
		},
		{
			name:    "numeric range",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"amount":75}}`,
			pattern: `{"detail":{"amount":[{"numeric":[">=",50,"<=",100]}]}}`,
			want:    true,
		},
		{
			name:    "numeric range fail",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"amount":25}}`,
			pattern: `{"detail":{"amount":[{"numeric":[">=",50,"<=",100]}]}}`,
			want:    false,
		},
		{
			name:    "suffix match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"filename":"report.pdf"}}`,
			pattern: `{"detail":{"filename":[{"suffix":".pdf"}]}}`,
			want:    true,
		},
		{
			name:    "wildcard match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"filename":"report_2024.pdf"}}`,
			pattern: `{"detail":{"filename":[{"wildcard":"report_*.pdf"}]}}`,
			want:    true,
		},
		{
			name:    "equals-ignore-case",
			event:   `{"source":"myapp.orders","detail-type":"OrderPLACED","detail":{}}`,
			pattern: `{"detail-type":[{"equals-ignore-case":"orderplaced"}]}`,
			want:    true,
		},
		{
			name:    "anything-but array",
			event:   `{"source":"myapp.users","detail-type":"UserLoggedIn","detail":{}}`,
			pattern: `{"source":[{"anything-but":["myapp.orders","myapp.payments"]}]}`,
			want:    true,
		},
		{
			name:    "anything-but array excluded",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":[{"anything-but":["myapp.orders","myapp.payments"]}]}`,
			want:    false,
		},
		{
			name:    "anything-but prefix",
			event:   `{"source":"myapp.users","detail-type":"UserLoggedIn","detail":{}}`,
			pattern: `{"source":[{"anything-but":{"prefix":"com."}}]}`,
			want:    true,
		},
		{
			name:    "anything-but prefix excluded",
			event:   `{"source":"com.external.svc","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":[{"anything-but":{"prefix":"com."}}]}`,
			want:    false,
		},
		{
			name:    "nested detail match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"customer":{"name":"John","tier":"premium"}}}`,
			pattern: `{"detail":{"customer":{"tier":["premium"]}}}`,
			want:    true,
		},
		{
			name:    "dot notation detail match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"customer":{"name":"John","tier":"premium"}}}`,
			pattern: `{"detail":{"customer.tier":["premium"]}}`,
			want:    true,
		},
		{
			name:    "multiple conditions all must match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"amount":150}}`,
			pattern: `{"source":[{"prefix":"myapp.orders"}],"detail-type":["OrderPlaced"],"detail":{"amount":[{"numeric":[">",100]}]}}`,
			want:    true,
		},
		{
			name:    "multiple conditions one fails",
			event:   `{"source":"myapp.orders","detail-type":"OrderCancelled","detail":{"amount":150}}`,
			pattern: `{"source":[{"prefix":"myapp.orders"}],"detail-type":["OrderPlaced"],"detail":{"amount":[{"numeric":[">",100]}]}}`,
			want:    false,
		},
		{
			name:    "empty pattern matches everything",
			event:   `{"source":"anything","detail-type":"anything","detail":{"foo":"bar"}}`,
			pattern: `{}`,
			want:    true,
		},
		{
			name:    "missing source field",
			event:   `{"detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"source":["myapp.orders"]}`,
			want:    false,
		},
		{
			name:    "exact boolean match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"isTest":true}}`,
			pattern: `{"detail":{"isTest":[true]}}`,
			want:    true,
		},
		{
			name:    "exact null match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"optionalField":null}}`,
			pattern: `{"detail":{"optionalField":[null]}}`,
			want:    true,
		},
		{
			name:    "cidr match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"ip":"10.0.0.5"}}`,
			pattern: `{"detail":{"ip":[{"cidr":"10.0.0.0/24"}]}}`,
			want:    true,
		},
		{
			name:    "cidr mismatch",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"ip":"192.168.1.1"}}`,
			pattern: `{"detail":{"ip":[{"cidr":"10.0.0.0/24"}]}}`,
			want:    false,
		},
		{
			name:    "resources match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","resources":["arn:aws:ec2:us-east-1:123456789012:instance/i-12345"],"detail":{}}`,
			pattern: `{"resources":["arn:aws:ec2:us-east-1:123456789012:instance/i-12345"]}`,
			want:    true,
		},
		{
			name:    "resources no match",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","resources":["arn:aws:lambda:us-east-1:123456789012:function:myfunc"],"detail":{}}`,
			pattern: `{"resources":["arn:aws:ec2:us-east-1:123456789012:instance/i-12345"]}`,
			want:    false,
		},
		{
			name:    "single value not in array",
			event:   `{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{}}`,
			pattern: `{"detail-type":"OrderPlaced"}`,
			want:    true,
		},
		{
			name:    "numeric string in detail vs numeric pattern",
			event:   `{"source":"test","detail-type":"test","detail":{"count":"42"}}`,
			pattern: `{"detail":{"count":[42]}}`,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event, pattern map[string]interface{}
			if err := json.Unmarshal([]byte(tt.event), &event); err != nil {
				t.Fatalf("bad event JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.pattern), &pattern); err != nil {
				t.Fatalf("bad pattern JSON: %v", err)
			}
			got := Matches(event, pattern)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
