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
			name:    "webhook rejects non http scheme",
			typ:     "webhook",
			address: "ftp://example.com/hook",
			wantErr: ErrInvalidAddress,
		},
		{
			name:    "unknown type is rejected",
			typ:     "email",
			address: "test@example.com",
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
