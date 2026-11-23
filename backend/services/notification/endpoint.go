package notification

import (
	"context"
	"encoding/base64"
	"expense-tracker/backend/types"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

func validateSubscription(ctx context.Context, input types.WebPushSubscriptionInput) error {
	parsed, err := url.Parse(strings.TrimSpace(input.Endpoint))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" || parsed.Port() != "" && parsed.Port() != "443" || !supportedPushHost(parsed.Hostname()) {
		return types.ErrInvalidWebPushSubscription
	}
	if err := ensurePublicHost(ctx, parsed.Hostname()); err != nil {
		return types.ErrInvalidWebPushSubscription
	}
	p256dh, err := base64.RawURLEncoding.DecodeString(input.P256DH)
	if err != nil || len(p256dh) != 65 || p256dh[0] != 4 {
		return types.ErrInvalidWebPushSubscription
	}
	authKey, err := base64.RawURLEncoding.DecodeString(input.Auth)
	if err != nil || len(authKey) != 16 {
		return types.ErrInvalidWebPushSubscription
	}
	return nil
}

func supportedPushHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "fcm.googleapis.com" || host == "updates.push.services.mozilla.com" || host == "push.apple.com" || strings.HasSuffix(host, ".push.apple.com")
}

func ensurePublicHost(ctx context.Context, host string) error {
	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupNetIP(lookupCtx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return types.ErrInvalidWebPushSubscription
	}
	for _, address := range addresses {
		if !isPublicAddress(address) {
			return types.ErrInvalidWebPushSubscription
		}
	}
	return nil
}

func isPublicAddress(address netip.Addr) bool {
	return address.IsValid() && !address.IsPrivate() && !address.IsLoopback() &&
		!address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast() && !address.IsUnspecified()
}
