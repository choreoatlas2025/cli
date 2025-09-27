package spec

import "testing"

func TestNormalizeServiceAlias(t *testing.T) {
    cases := map[string]string{
        "order-service":        "orderService",
        "Payment.Service":      "paymentService",
        " Shipping Service ":   "shippingService",
        "":                      "service",
    }
    for in, want := range cases {
        if got := NormalizeServiceAlias(in); got != want {
            t.Errorf("alias(%q)=%q want %q", in, got, want)
        }
    }
}

func TestNormalizeOperationID_HTTP(t *testing.T) {
    cases := map[string]string{
        "GET /v1/users/{id}":       "getV1UsersById",
        "POST /orders":             "postOrders",
        "PUT /items/:sku":          "putItemsBySku",
        "DELETE /v1/a/b/":          "deleteV1AB",
    }
    for in, want := range cases {
        if got := NormalizeOperationID(in); got != want {
            t.Errorf("op(%q)=%q want %q", in, got, want)
        }
    }
}

func TestNormalizeOperationID_RPC(t *testing.T) {
    cases := map[string]string{
        "UserService.Get":      "get",
        "pkg.Order/Reserve":    "reserve",
        "Inventory.Check":      "check",
    }
    for in, want := range cases {
        if got := NormalizeOperationID(in); got != want {
            t.Errorf("rpc op(%q)=%q want %q", in, got, want)
        }
    }
}

func TestNormalizeOperationID_Fallback(t *testing.T) {
    if got := NormalizeOperationID(" custom op 42 "); got != "customOp42" {
        t.Errorf("fallback got %q", got)
    }
}

