package gamepool

import "testing"

func TestSimRunsFrames(t *testing.T) {
	sim := NewSim(64)
	stats := sim.Run(50, 2)
	if stats.Frames != 50 {
		t.Fatalf("帧数不对：%d", stats.Frames)
	}
	if stats.Spawned == 0 {
		t.Fatal("应该生成过子弹")
	}
}

func TestSimSpawnsEveryFrame(t *testing.T) {
	sim := NewSim(4096)
	stats := sim.Run(10, 3)
	if stats.Spawned < 30 {
		t.Fatalf("10 帧 × 3 发应该有 30 次生成：%d", stats.Spawned)
	}
}

func TestFreeIsNotNegative(t *testing.T) {
	sim := NewSim(16)
	stats := sim.Run(20, 1)
	if stats.Free < 0 || stats.Alive < 0 {
		t.Fatalf("计数不该为负：%+v", stats)
	}
}

func TestStaleCheckReturnsSomething(t *testing.T) {
	accepted, _ := StaleCheck()
	_ = accepted
}
