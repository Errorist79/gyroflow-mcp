package gyroflow

import (
	"context"
	"testing"
)

// Real captured strings from codesign --display --entitlements - --verbose
// on the App Store build of Gyroflow. Pinned so the test breaks if the
// string matching regresses.

const sandboxedEntitlement = `[Key] com.apple.security.app-sandbox`
const appStoreAuthority = `Authority=Apple Mac OS Application Signing`

func TestClassifyEntitlementsDetectsSandbox(t *testing.T) {
	if !classifyEntitlements(sandboxedEntitlement) {
		t.Fatal("expected sandboxed=true for App Store entitlement")
	}
}

func TestClassifyEntitlementsNotSandboxed(t *testing.T) {
	if classifyEntitlements("some other entitlement output") {
		t.Fatal("expected sandboxed=false for non-sandboxed output")
	}
}

func TestClassifyAuthorityDetectsAppStore(t *testing.T) {
	if !classifyAuthority(appStoreAuthority) {
		t.Fatal("expected appStore=true for App Store authority")
	}
}

func TestClassifyAuthorityNotAppStore(t *testing.T) {
	if classifyAuthority("Authority=Developer ID Application: Some Dev (XXXXXXXXXX)") {
		t.Fatal("expected appStore=false for Developer ID authority")
	}
}

func TestAppBundlePathFindsApp(t *testing.T) {
	got := appBundlePath("/Applications/Gyroflow.app/Contents/MacOS/gyroflow")
	want := "/Applications/Gyroflow.app"
	if got != want {
		t.Fatalf("appBundlePath = %q, want %q", got, want)
	}
}

func TestAppBundlePathNoApp(t *testing.T) {
	in := "/usr/local/bin/gyroflow"
	got := appBundlePath(in)
	if got != in {
		t.Fatalf("appBundlePath(%q) = %q, want input unchanged", in, got)
	}
}

func TestAppBundlePathNoAppOpt(t *testing.T) {
	in := "/opt/gyroflow/gyroflow"
	got := appBundlePath(in)
	if got != in {
		t.Fatalf("appBundlePath(%q) = %q, want input unchanged", in, got)
	}
}

// TestInspectSandboxSignatureReturnsError verifies the (SandboxResult, error)
// signature exists and compiles. The non-darwin path always returns nil error.
func TestInspectSandboxSignatureReturnsError(t *testing.T) {
	_, err := InspectSandbox(context.TODO(), "/usr/local/bin/gyroflow")
	_ = err // nil on non-darwin; compile-check for return signature
}
