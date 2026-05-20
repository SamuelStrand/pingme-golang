package alertchannel

import "testing"

func TestNormalizeAndValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		typ       string
		address   string
		wantType  string
		wantValue string
		wantErr   error
	}{
		{
			name:      "telegram allows trimmed value",
			typ:       " Telegram ",
			address:   " 123456 ",
			wantType:  TypeTelegram,
			wantValue: "123456",
		},
		{
			name:      "webhook requires http scheme",
			typ:       "webhook",
			address:   "https://example.com/hook",
			wantType:  TypeWebhook,
			wantValue: "https://example.com/hook",
		},
		{
			name:      "email allows trimmed value",
			typ:       " Email ",
			address:   " team@example.com ",
			wantType:  TypeEmail,
			wantValue: "team@example.com",
		},
		{
			name:    "webhook rejects non http scheme",
			typ:     "webhook",
			address: "ftp://example.com/hook",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "email rejects missing at sign",
			typ:     "email",
			address: "team.example.com",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "email rejects whitespace",
			typ:     "email",
			address: "team @example.com",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "email rejects too short address",
			typ:     "email",
			address: "a@b",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "unknown type is rejected",
			typ:     "sms",
			address: "+123456789",
			wantErr: ErrInvalidType,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotType, gotValue, err := normalizeAndValidate(testCase.typ, testCase.address)
			if testCase.wantErr != nil {
				if err != testCase.wantErr {
					t.Fatalf("err = %v, want %v", err, testCase.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if gotType != testCase.wantType {
				t.Fatalf("type = %q, want %q", gotType, testCase.wantType)
			}
			if gotValue != testCase.wantValue {
				t.Fatalf("value = %q, want %q", gotValue, testCase.wantValue)
			}
		})
	}
}
