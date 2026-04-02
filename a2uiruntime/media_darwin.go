//go:build darwin

package a2uiruntime

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/avkit"
)

var loadAVKitOnce sync.Once

func playerView(cache *stateCache, url string, width, height float64) swiftui.View {
	if url == "" {
		return swiftui.Text("Missing media URL").Font(swiftui.FontCaption).ForegroundStyleNamed("secondary").AsView()
	}
	loadAVKitOnce.Do(func() {
		purego.Dlopen("/System/Library/Frameworks/AVKit.framework/AVKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	})

	key := "player:" + url
	player, ok := cache.getPlayer(key)
	if !ok {
		nsurl := objc.ID(objc.GetClass("NSURL")).Send(
			objc.RegisterName("URLWithString:"),
			newNSString(url),
		)
		playerID := objc.ID(objc.GetClass("AVPlayer")).Send(
			objc.RegisterName("playerWithURL:"),
			nsurl,
		)
		player = uintptr(playerID)
		cache.setPlayer(key, player)
	}
	playerID := objc.ID(player)
	playerID.Send(objc.RegisterName("pause"))
	viewPtr := avkit.NewVideoPlayer(player)
	return swiftui.ViewFromPointer(viewPtr).Frame(width, height)
}

func newNSString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(
		objc.RegisterName("stringWithUTF8String:"),
		s,
	)
}
