package scenekit

// SceneViewOption specifies rendering options for SceneView.

type SceneViewOption int32

const (
	SceneViewOptionAllowsCameraControl         SceneViewOption = 1
	SceneViewOptionAutoenablesDefaultLighting  SceneViewOption = 2
	SceneViewOptionJitteringEnabled            SceneViewOption = 4
	SceneViewOptionTemporalAntialiasingEnabled SceneViewOption = 8
)
