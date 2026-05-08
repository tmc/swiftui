package swiftui

import "testing"

func TestLifecycleCallbackRegistration(t *testing.T) {
	var launched, activated, resigned, terminated bool
	shouldAllow := true

	lc := AppLifecycle{
		OnLaunched:     func() { launched = true },
		OnActivate:     func() { activated = true },
		OnResignActive: func() { resigned = true },
		ShouldTerminate: func() bool {
			return shouldAllow
		},
		OnTerminate: func() { terminated = true },
	}

	plan := scenePlan{lifecycle: &lc}

	// Build the run plan lifecycle section.
	rlc := &sceneRunPlanLifecycle{}
	if plan.lifecycle.OnLaunched != nil {
		rlc.DidFinishLaunchingCallbackID = uint64(registerCallback(plan.lifecycle.OnLaunched))
	}
	if plan.lifecycle.OnActivate != nil {
		rlc.DidBecomeActiveCallbackID = uint64(registerCallback(plan.lifecycle.OnActivate))
	}
	if plan.lifecycle.OnResignActive != nil {
		rlc.DidResignActiveCallbackID = uint64(registerCallback(plan.lifecycle.OnResignActive))
	}
	if plan.lifecycle.ShouldTerminate != nil {
		fn := plan.lifecycle.ShouldTerminate
		rlc.ShouldTerminateCallbackID = uint64(registerCommandCallback(func() int32 {
			if fn() {
				return 1
			}
			return 0
		}))
	}
	if plan.lifecycle.OnTerminate != nil {
		rlc.WillTerminateCallbackID = uint64(registerCallback(plan.lifecycle.OnTerminate))
	}

	// Verify all IDs are non-zero.
	if rlc.DidFinishLaunchingCallbackID == 0 {
		t.Error("DidFinishLaunchingCallbackID should be non-zero")
	}
	if rlc.DidBecomeActiveCallbackID == 0 {
		t.Error("DidBecomeActiveCallbackID should be non-zero")
	}
	if rlc.DidResignActiveCallbackID == 0 {
		t.Error("DidResignActiveCallbackID should be non-zero")
	}
	if rlc.ShouldTerminateCallbackID == 0 {
		t.Error("ShouldTerminateCallbackID should be non-zero")
	}
	if rlc.WillTerminateCallbackID == 0 {
		t.Error("WillTerminateCallbackID should be non-zero")
	}

	// Dispatch fire-and-forget callbacks via the button trampoline.
	buttonCallbackTrampoline(uintptr(rlc.DidFinishLaunchingCallbackID))
	if !launched {
		t.Error("OnLaunched callback did not fire")
	}
	buttonCallbackTrampoline(uintptr(rlc.DidBecomeActiveCallbackID))
	if !activated {
		t.Error("OnActivate callback did not fire")
	}
	buttonCallbackTrampoline(uintptr(rlc.DidResignActiveCallbackID))
	if !resigned {
		t.Error("OnResignActive callback did not fire")
	}
	buttonCallbackTrampoline(uintptr(rlc.WillTerminateCallbackID))
	if !terminated {
		t.Error("OnTerminate callback did not fire")
	}

	// Dispatch shouldTerminate via the command trampoline.
	if got := commandCallbackTrampoline(uintptr(rlc.ShouldTerminateCallbackID)); got != 1 {
		t.Errorf("ShouldTerminate returned %d, want 1", got)
	}
	shouldAllow = false
	if got := commandCallbackTrampoline(uintptr(rlc.ShouldTerminateCallbackID)); got != 0 {
		t.Errorf("ShouldTerminate returned %d, want 0", got)
	}
}

func TestLifecycleNilCallbacks(t *testing.T) {
	lc := AppLifecycle{} // all nil
	plan := scenePlan{lifecycle: &lc}

	rlc := &sceneRunPlanLifecycle{}
	if plan.lifecycle.OnLaunched != nil {
		rlc.DidFinishLaunchingCallbackID = uint64(registerCallback(plan.lifecycle.OnLaunched))
	}
	if plan.lifecycle.OnActivate != nil {
		rlc.DidBecomeActiveCallbackID = uint64(registerCallback(plan.lifecycle.OnActivate))
	}

	if rlc.DidFinishLaunchingCallbackID != 0 {
		t.Error("nil OnLaunched should produce ID 0")
	}
	if rlc.DidBecomeActiveCallbackID != 0 {
		t.Error("nil OnActivate should produce ID 0")
	}
}
