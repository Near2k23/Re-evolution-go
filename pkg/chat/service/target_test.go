package chat_service

import "testing"

func TestTarget(t *testing.T) {
	for _, tc := range []struct {
		in   BodyStruct
		want string
	}{
		{BodyStruct{Chat: "573001112233@s.whatsapp.net"}, "573001112233@s.whatsapp.net"},
		{BodyStruct{Chat: "573001112233:12@s.whatsapp.net"}, "573001112233@s.whatsapp.net"},
		{BodyStruct{Number: "573001112233"}, "573001112233@s.whatsapp.net"},
		{BodyStruct{Chat: "120363000000000000@g.us"}, "120363000000000000@g.us"},
	} {
		got, err := target(&tc.in)
		if err != nil || got.String() != tc.want {
			t.Errorf("%+v → %v %v", tc.in, got, err)
		}
	}
	if _, err := target(&BodyStruct{}); err == nil {
		t.Error("empty chat must fail")
	}
}
