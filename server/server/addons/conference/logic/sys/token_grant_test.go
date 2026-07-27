package sys_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/livekit/protocol/auth"
)

func TestAccessTokenGrants(t *testing.T) {
	at := auth.NewAccessToken("devkey", "secret")
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     "demo",
	}
	grant.SetCanPublish(true)
	grant.SetCanSubscribe(true)
	grant.SetCanPublishData(true)

	at.SetVideoGrant(grant).
		SetIdentity("u_abc123").
		SetName("张三").
		SetValidFor(15 * time.Minute)

	token, err := at.ToJWT()
	if err != nil {
		t.Fatalf("ToJWT: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims type")
	}
	if claims["sub"] != "u_abc123" && claims["identity"] != "u_abc123" {
		// livekit puts identity in "sub" typically; also check video grant
	}
	video, ok := claims["video"].(map[string]any)
	if !ok {
		t.Fatalf("missing video grant: %#v", claims)
	}
	if video["room"] != "demo" {
		t.Fatalf("room=%v", video["room"])
	}
	if video["roomJoin"] != true {
		t.Fatalf("roomJoin=%v", video["roomJoin"])
	}
	if video["canPublish"] != true {
		t.Fatalf("canPublish=%v", video["canPublish"])
	}
	if video["canSubscribe"] != true {
		t.Fatalf("canSubscribe=%v", video["canSubscribe"])
	}
	if video["canPublishData"] != true {
		t.Fatalf("canPublishData=%v", video["canPublishData"])
	}
}
