//go:build darwin
// +build darwin

// Command music-player demonstrates a rich music player UI with playlist
// navigation, now-playing view, and audio visualizer using SwiftUI from Go.
//
// Background goroutines drive simulated playback (1-second ticks) and
// visualizer bar animations (200ms updates), all through goroutine-safe
// State.Set() calls.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

type track struct {
	title    string
	artist   string
	album    string
	duration int // seconds
	genre    string
}

type playlist struct {
	name   string
	tracks []track
}

type queuedTrack struct {
	playlist int
	track    int
}

var playlists = []playlist{
	{
		name: "Favorites",
		tracks: []track{
			{"Midnight City", "M83", "Hurry Up, We're Dreaming", 244, "Electronic"},
			{"Starlight", "Muse", "Black Holes and Revelations", 240, "Rock"},
			{"Digital Love", "Daft Punk", "Discovery", 301, "Electronic"},
		},
	},
	{
		name: "Chill",
		tracks: []track{
			{"Intro", "The xx", "xx", 128, "Indie"},
			{"Midnight", "Coldplay", "Ghost Stories", 293, "Pop"},
			{"Holocene", "Bon Iver", "Bon Iver", 337, "Folk"},
			{"Re: Stacks", "Bon Iver", "For Emma, Forever Ago", 378, "Folk"},
		},
	},
	{
		name: "Workout",
		tracks: []track{
			{"Stronger", "Kanye West", "Graduation", 312, "Hip-Hop"},
			{"Lose Yourself", "Eminem", "8 Mile", 326, "Hip-Hop"},
			{"Eye of the Tiger", "Survivor", "Eye of the Tiger", 245, "Rock"},
		},
	},
}

// Global playback state, protected by mu.
var (
	mu              sync.Mutex
	currentPlaylist int // index into playlists, -1 = none
	currentTrack    int // index into playlist tracks, -1 = none
	queue           []queuedTrack
)

func formatDuration(secs int) string {
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func allTracks(pl int) []track {
	if pl < 0 || pl >= len(playlists) {
		return nil
	}
	return playlists[pl].tracks
}

func trackInfo(pl, ti int) (track, bool) {
	tracks := allTracks(pl)
	if ti < 0 || ti >= len(tracks) {
		return track{}, false
	}
	return tracks[ti], true
}

func playbackKey(pl, ti int) int {
	if pl < 0 || ti < 0 {
		return -1
	}
	return pl<<16 | ti
}

func currentTrackInfo() (track, bool) {
	mu.Lock()
	defer mu.Unlock()
	return trackInfo(currentPlaylist, currentTrack)
}

func main() {
	// State for current track index (-1 = none).
	trackState := swiftui.NewIntState(-1)
	// Playing state: 0=paused, 1=playing.
	playingState := swiftui.NewIntState(0)
	// Playback position in seconds.
	positionState := swiftui.NewFloatState(0)
	// Sheet presented state: 0=hidden, 1=shown.
	sheetState := swiftui.NewIntState(0)
	// Popover for queue: 0=hidden, 1=shown.
	queuePopoverState := swiftui.NewIntState(0)
	// Volume.
	volumeState := swiftui.NewIntState(75)
	// Visualizer bars (8 bars).
	var vizBars [8]*swiftui.FloatState
	for i := range vizBars {
		vizBars[i] = swiftui.NewFloatState(0.1)
	}
	// Tick counter for visualizer redraw.
	vizTick := swiftui.NewIntState(0)
	// Track name for now-playing bar (rebuilt via DynamicView on trackState).
	// Queue counter for triggering queue view rebuilds.
	queueCount := swiftui.NewIntState(0)

	mu.Lock()
	currentPlaylist = 0
	currentTrack = 0
	for j := 1; j < len(playlists[0].tracks); j++ {
		queue = append(queue, queuedTrack{playlist: 0, track: j})
	}
	initialQueue := len(queue)
	mu.Unlock()
	trackState.Set(playbackKey(0, 0))
	queueCount.Set(initialQueue)

	// Playback ticker goroutine.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if playingState.Get() != 1 {
				continue
			}
			pos := positionState.Get() + 1
			t, ok := currentTrackInfo()
			if !ok {
				continue
			}
			if pos >= float64(t.duration) {
				// Auto-advance.
				mu.Lock()
				advanced := false
				if len(queue) > 0 {
					next := queue[0]
					queue = queue[1:]
					currentPlaylist = next.playlist
					currentTrack = next.track
					advanced = true
				} else if currentPlaylist >= 0 {
					next := currentTrack + 1
					tracks := allTracks(currentPlaylist)
					if next < len(tracks) {
						currentTrack = next
						advanced = true
					}
				}
				cp := currentPlaylist
				ct := currentTrack
				qc := len(queue)
				mu.Unlock()
				if advanced {
					positionState.Set(0)
					trackState.Set(playbackKey(cp, ct))
					queueCount.Set(qc)
				} else {
					playingState.Set(0)
					positionState.Set(0)
				}
				continue
			}
			positionState.Set(pos)
		}
	}()

	// Visualizer goroutine.
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if playingState.Get() != 1 {
				for i := range vizBars {
					vizBars[i].Set(0.05)
				}
				vizTick.Set(vizTick.Get() + 1)
				continue
			}
			for i := range vizBars {
				vizBars[i].Set(0.1 + rand.Float64()*0.9)
			}
			vizTick.Set(vizTick.Get() + 1)
		}
	}()

	selectTrack := func(pl, ti int) {
		mu.Lock()
		currentPlaylist = pl
		currentTrack = ti
		// Build queue: remaining tracks after selected.
		queue = nil
		tracks := allTracks(pl)
		for j := ti + 1; j < len(tracks); j++ {
			queue = append(queue, queuedTrack{playlist: pl, track: j})
		}
		qc := len(queue)
		mu.Unlock()
		positionState.Set(0)
		trackState.Set(playbackKey(pl, ti))
		playingState.Set(1)
		sheetState.Set(1)
		queueCount.Set(qc)
	}

	togglePlay := func() {
		if playingState.Get() == 0 {
			playingState.Set(1)
		} else {
			playingState.Set(0)
		}
	}

	prevTrack := func() {
		mu.Lock()
		if currentPlaylist >= 0 && currentTrack > 0 {
			currentTrack--
		}
		cp := currentPlaylist
		ct := currentTrack
		mu.Unlock()
		positionState.Set(0)
		trackState.Set(playbackKey(cp, ct))
	}

	nextTrack := func() {
		mu.Lock()
		advanced := false
		if len(queue) > 0 {
			next := queue[0]
			queue = queue[1:]
			currentPlaylist = next.playlist
			currentTrack = next.track
			advanced = true
		} else if currentPlaylist >= 0 {
			next := currentTrack + 1
			tracks := allTracks(currentPlaylist)
			if next < len(tracks) {
				currentTrack = next
				advanced = true
			}
		}
		cp := currentPlaylist
		ct := currentTrack
		qc := len(queue)
		mu.Unlock()
		if advanced {
			positionState.Set(0)
			trackState.Set(playbackKey(cp, ct))
			queueCount.Set(qc)
		}
	}

	addToQueue := func(pl, ti int) {
		mu.Lock()
		queue = append(queue, queuedTrack{playlist: pl, track: ti})
		qc := len(queue)
		mu.Unlock()
		queueCount.Set(qc)
	}

	// Build playlist browse content.
	var playlistCards []swiftui.Viewable
	for pi, pl := range playlists {
		pi, pl := pi, pl
		lead := pl.tracks[0]
		playlistCards = append(playlistCards,
			swiftui.GroupBox(pl.name,
				swiftui.VStackSpaced(8,
					infoLine("Tracks", fmt.Sprintf("%d", len(pl.tracks))),
					infoLine("Lead", lead.title),
					infoLine("Style", lead.genre),
					swiftui.HStackSpaced(8,
						swiftui.Button("Play", func() {
							selectTrack(pi, 0)
						}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
						swiftui.Button("Queue All", func() {
							for ti := range pl.tracks {
								addToQueue(pi, ti)
							}
						}).ButtonStyle(swiftui.ButtonStyleBordered),
					),
				).Padding(10),
			).MaxFrame(-1, 0),
		)
	}

	browseContent := swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			swiftui.HStack(
				swiftui.VStackSpaced(4,
					swiftui.HStack(
						swiftui.Text("Browse").
							Font(swiftui.FontTitle).
							FontWeight(swiftui.WeightBold),
						swiftui.Spacer(),
					),
					swiftui.HStack(
						swiftui.Text("A compact music surface with playlists, queue context, and a persistent now-playing bar.").
							Font(swiftui.FontCallout).
							ForegroundStyleNamed("secondary"),
						swiftui.Spacer(),
					),
				).MaxFrame(-1, 0),
			),
			swiftui.DynamicView(trackState, func(_ int) swiftui.View {
				t, ok := currentTrackInfo()
				if !ok {
					return swiftui.Text("").AsView().Frame(0, 0)
				}
				return swiftui.HStackSpaced(16,
					swiftui.ZStack(
						swiftui.RoundedRectangle(18).
							Fill(swiftui.RGBA(0.15, 0.18, 0.28, 1.0)).
							Frame(150, 150).
							AsView(),
						swiftui.Image("music.note.list").
							ForegroundStyle(swiftui.RGBA(0.35, 0.65, 1.0, 1.0)).
							ImageScale(swiftui.ImageScaleLarge),
					),
					swiftui.VStackSpaced(8,
						swiftui.HStack(
							swiftui.Text("Current Selection").
								Font(swiftui.FontCaption).
								ForegroundStyleNamed("secondary"),
							swiftui.Spacer(),
						),
						swiftui.HStack(
							swiftui.Text(t.title).
								Font(swiftui.FontTitle2).
								FontWeight(swiftui.WeightBold),
							swiftui.Spacer(),
						),
						swiftui.HStack(
							swiftui.Text(t.artist+" • "+t.album).
								Font(swiftui.FontBody).
								ForegroundStyleNamed("secondary"),
							swiftui.Spacer(),
						),
						swiftui.HStackSpaced(12,
							musicBadge("Genre", t.genre),
							musicBadge("Length", formatDuration(t.duration)),
						),
						swiftui.HStackSpaced(8,
							swiftui.Button("Play", func() {
								playingState.Set(1)
								sheetState.Set(1)
							}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
							swiftui.Button("Open Queue", func() {
								queuePopoverState.Set(1)
							}).ButtonStyle(swiftui.ButtonStyleBordered),
						),
					).MaxFrame(-1, 0),
				).Padding(14).
					Background(swiftui.RGBA(0.18, 0.19, 0.23, 0.7)).
					CornerRadius(16)
			}),
			swiftui.HStackSpaced(12,
				musicStatCard("Playlists", fmt.Sprintf("%d", len(playlists)), "Collections"),
				musicStatCard("Queue", fmt.Sprintf("%d", queueCount.Get()), "Up next"),
				musicStatCard("Volume", fmt.Sprintf("%d%%", volumeState.Get()), "Output"),
			),
			swiftui.HStackSpaced(12,
				swiftui.GroupBox("Up Next",
					swiftui.DynamicView(queueCount, func(_ int) swiftui.View {
						mu.Lock()
						q := make([]queuedTrack, len(queue))
						copy(q, queue)
						mu.Unlock()
						if len(q) == 0 {
							return swiftui.Text("Queue empty").
								Font(swiftui.FontCaption).
								ForegroundStyleNamed("secondary").
								Padding(10).
								AsView()
						}
						limit := len(q)
						if limit > 4 {
							limit = 4
						}
						rows := make([]swiftui.Viewable, 0, limit)
						for i := 0; i < limit; i++ {
							t, ok := trackInfo(q[i].playlist, q[i].track)
							if !ok {
								continue
							}
							rows = append(rows, swiftui.HStack(
								swiftui.Text(t.title),
								swiftui.Spacer(),
								swiftui.Text(t.artist).
									Font(swiftui.FontCaption).
									ForegroundStyleNamed("secondary"),
							))
						}
						return swiftui.VStackSpaced(8, rows...).Padding(10)
					}),
				).MaxFrame(-1, 0),
				swiftui.GroupBox("Playback",
					swiftui.VStackSpaced(10,
						infoLine("State", playbackStateLabel(playingState.Get())),
						infoLine("Current", currentTrackLabel()),
						infoLine("Visualizer", "8 animated bars"),
						infoLine("Sheet", "Detailed now-playing surface"),
					).Padding(10),
				).MaxFrame(-1, 0),
			),
			swiftui.HStackSpaced(12, playlistCards...),
		).Padding(20),
	)

	// Visualizer view.
	visualizer := func() swiftui.View {
		var bars []swiftui.Viewable
		for i := range vizBars {
			i := i
			bars = append(bars, swiftui.DynamicView(vizTick, func(_ int) swiftui.View {
				h := vizBars[i].Get() * 80
				if h < 4 {
					h = 4
				}
				return swiftui.RoundedRectangle(3).
					Fill(swiftui.RGBA(0.3, 0.7, 1.0, 0.8)).
					Frame(16, h).
					AsView()
			}))
		}
		return swiftui.HStackSpaced(4, bars...).Frame(0, 80)
	}

	// Now-playing sheet content.
	sheetContent := swiftui.DynamicView(trackState, func(ti int) swiftui.View {
		t, ok := currentTrackInfo()
		if !ok {
			return swiftui.Text("No track selected").AsView()
		}
		return swiftui.VStackSpaced(12,
			swiftui.Spacer(),
			// Album art placeholder.
			swiftui.ZStack(
				swiftui.RoundedRectangle(20).
					Fill(swiftui.RGBA(0.15, 0.15, 0.3, 1.0)).
					Frame(200, 200).
					AsView().
					Shadow(swiftui.RGBA(0, 0, 0, 0.3), 10, 0, 5),
				swiftui.Image("music.note").
					ForegroundStyle(swiftui.RGBA(1, 1, 1, 0.4)).
					ImageScale(swiftui.ImageScaleLarge),
			),
			// Track info.
			swiftui.Text(t.title).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.Text(t.artist).
				Font(swiftui.FontBody).
				ForegroundStyleNamed("secondary"),
			swiftui.Text(t.album).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			// Visualizer.
			visualizer(),
			// Seek slider.
			swiftui.FloatSlider("", positionState, 0, float64(t.duration), func() {}),
			swiftui.HStack(
				swiftui.DynamicView(swiftui.NewIntState(0), func(_ int) swiftui.View {
					pos := positionState.Get()
					return swiftui.Text(formatDuration(int(pos))).
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary").
						AsView()
				}),
				swiftui.Spacer(),
				swiftui.Text(formatDuration(t.duration)).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
			),
			// Playback controls.
			swiftui.HStackSpaced(24,
				swiftui.ButtonWithImage("shuffle", func() {}).
					ForegroundStyleNamed("secondary"),
				swiftui.ButtonWithImage("backward.fill", func() { prevTrack() }).
					ImageScale(swiftui.ImageScaleLarge),
				swiftui.DynamicView(playingState, func(p int) swiftui.View {
					icon := "play.fill"
					if p == 1 {
						icon = "pause.fill"
					}
					return swiftui.ButtonWithImage(icon, func() { togglePlay() }).
						ImageScale(swiftui.ImageScaleLarge).
						Font(swiftui.FontTitle)
				}),
				swiftui.ButtonWithImage("forward.fill", func() { nextTrack() }).
					ImageScale(swiftui.ImageScaleLarge),
				swiftui.ButtonWithImage("repeat", func() {}).
					ForegroundStyleNamed("secondary"),
			),
			// Volume.
			swiftui.HStackSpaced(8,
				swiftui.Image("speaker.fill").
					ForegroundStyleNamed("secondary").
					ImageScale(swiftui.ImageScaleSmall),
				swiftui.Slider("", volumeState, 0, 100, func() {}),
				swiftui.Image("speaker.wave.3.fill").
					ForegroundStyleNamed("secondary").
					ImageScale(swiftui.ImageScaleSmall),
			),
			swiftui.Spacer(),
		).Padding(24).
			BackgroundStyle("ultraThinMaterial")
	})

	// Queue popover content.
	queuePopover := swiftui.DynamicView(queueCount, func(qc int) swiftui.View {
		mu.Lock()
		q := make([]queuedTrack, len(queue))
		copy(q, queue)
		mu.Unlock()
		if len(q) == 0 {
			return swiftui.Text("Queue empty").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				Padding(12).AsView()
		}
		var rows []swiftui.Viewable
		limit := 3
		if len(q) < limit {
			limit = len(q)
		}
		for i := 0; i < limit; i++ {
			entry := q[i]
			t, ok := trackInfo(entry.playlist, entry.track)
			if !ok {
				continue
			}
			subtitle := t.artist
			if entry.playlist >= 0 && entry.playlist < len(playlists) {
				subtitle = fmt.Sprintf("%s • %s", t.artist, playlists[entry.playlist].name)
			}
			rows = append(rows, swiftui.HStack(
				swiftui.Text(t.title).Font(swiftui.FontCaption),
				swiftui.Spacer(),
				swiftui.Text(subtitle).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
			).Padding(4))
		}
		return swiftui.VStackSpaced(2, rows...).Padding(12).Frame(250, 0)
	})

	// Now-playing bar (bottom).
	nowPlayingBar := swiftui.DynamicView(trackState, func(ti int) swiftui.View {
		t, ok := currentTrackInfo()
		if !ok {
			return swiftui.Spacer().Frame(0, 0)
		}
		return swiftui.VStack(
			swiftui.Divider(),
			swiftui.HStack(
				swiftui.ButtonWithImage("music.note", func() {
					sheetState.Set(1)
				}).ForegroundStyle(swiftui.RGBA(0.3, 0.7, 1.0, 1.0)),
				swiftui.VStack(
					swiftui.Text(t.title).
						Font(swiftui.FontCaption).
						FontWeight(swiftui.WeightSemibold),
					swiftui.Text(t.artist).
						Font(swiftui.FontCaption2).
						ForegroundStyleNamed("secondary"),
				),
				swiftui.Spacer(),
				swiftui.DynamicView(playingState, func(p int) swiftui.View {
					icon := "play.fill"
					if p == 1 {
						icon = "pause.fill"
					}
					return swiftui.ButtonWithImage(icon, func() { togglePlay() })
				}),
				swiftui.ButtonWithImage("list.bullet", func() {
					if queuePopoverState.Get() == 0 {
						queuePopoverState.Set(1)
					} else {
						queuePopoverState.Set(0)
					}
				}).Popover(queuePopoverState, queuePopover),
			).Padding(8),
			swiftui.FloatProgressView(positionState, func() float64 {
				t, ok := currentTrackInfo()
				if !ok {
					return 1
				}
				return float64(t.duration)
			}()).Tint(swiftui.RGBA(0.3, 0.7, 1.0, 1.0)),
		)
	})
	if err :=

		// Main layout.
		swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
			Title:  "Music Player",
			Width:  600,
			Height: 760,
			Root: swiftui.VStack(
				browseContent,
				nowPlayingBar,
			).Sheet(sheetState, sheetContent),
		}}}); err != nil {
		log.Fatal(err)
	}
}

func musicBadge(label, value string) swiftui.View {
	return swiftui.VStackSpaced(2,
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary"),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightSemibold),
	).Padding(8).
		Background(swiftui.RGBA(1, 1, 1, 0.05)).
		CornerRadius(8)
}

func musicStatCard(label, value, note string) swiftui.View {
	return swiftui.VStackSpaced(6,
		swiftui.HStack(
			swiftui.Text(label).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(value).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		),
		swiftui.HStack(
			swiftui.Text(note).
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
			swiftui.Spacer(),
		),
	).Padding(12).
		Background(swiftui.RGBA(0.18, 0.19, 0.23, 0.62)).
		CornerRadius(10)
}

func playbackStateLabel(v int) string {
	if v == 1 {
		return "Playing"
	}
	return "Paused"
}

func currentTrackLabel() string {
	t, ok := currentTrackInfo()
	if !ok {
		return "None"
	}
	return t.title
}

func infoLine(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).
			ForegroundStyleNamed("secondary"),
		swiftui.Spacer(),
		swiftui.Text(value).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightMedium),
	)
}
