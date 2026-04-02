//go:build darwin

package a2uiruntime

import (
	"net/url"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/avkit"
)

var loadAVKitOnce sync.Once

func playerView(cache *stateCache, mediaPolicy MediaPolicy, rawURL string, width, height float64) swiftui.View {
	if rawURL == "" {
		return swiftui.Text("Missing media URL").Font(swiftui.FontCaption).ForegroundStyleNamed("secondary").AsView()
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return swiftui.Text("Invalid media URL").Font(swiftui.FontCaption).ForegroundStyleNamed("secondary").AsView()
	}
	if parsed.Scheme != "" && parsed.Scheme != "file" && !mediaPolicy.AllowRemote {
		return swiftui.VStackAlignedSpaced(swiftui.HorizontalAlignmentLeading, 6,
			swiftui.Text("Remote media disabled").Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
			swiftui.Link("Open media", rawURL).ControlSize(swiftui.ControlSizeSmall),
		)
	}
	loadAVKitOnce.Do(func() {
		purego.Dlopen("/System/Library/Frameworks/AVKit.framework/AVKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	})

	key := "player:" + rawURL
	player, ok := cache.getPlayer(key)
	if !ok {
		nsurl := objc.ID(objc.GetClass("NSURL")).Send(
			objc.RegisterName("URLWithString:"),
			newNSString(rawURL),
		)
		playerID := objc.ID(objc.GetClass("AVPlayer")).Send(
			objc.RegisterName("playerWithURL:"),
			nsurl,
		)
		player = uintptr(playerID)
		cache.setPlayer(key, player)
	}
	playerID := objc.ID(player)
	if mediaPolicy.Autoplay {
		playerID.Send(objc.RegisterName("play"))
	} else {
		playerID.Send(objc.RegisterName("pause"))
	}
	viewPtr := avkit.NewVideoPlayer(player)
	return swiftui.ViewFromPointer(viewPtr).Frame(width, height)
}

func newNSString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(
		objc.RegisterName("stringWithUTF8String:"),
		s,
	)
}
