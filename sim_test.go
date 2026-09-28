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

func TestTenThousandFramesScenario(t *testing.T) {
	sim := NewSim(256)
	stats := sim.Run(10_000, 4)
	if !stats.Consistent {
		t.Fatalf("长跑后池必须自洽：%+v", stats)
	}
	if stats.GhostWrites != 0 {
		t.Fatalf("幽灵写入必须为 0：%d", stats.GhostWrites)
	}
	if stats.Dropped != 0 {
		t.Fatalf("容量足够时不该有丢弃：%d", stats.Dropped)
	}
	if stats.Alive < 100 || stats.Alive > 140 {
		t.Fatalf("alive 应该稳定在 120 左右：%d", stats.Alive)
	}
	if stats.Free != 256-stats.Alive {
		t.Fatalf("free 应该等于 capacity-alive：%d != %d", stats.Free, 256-stats.Alive)
	}
	if stats.Spawned-stats.Released != stats.Alive {
		t.Fatalf("spawned-released 必须等于 alive：%d-%d != %d",
			stats.Spawned, stats.Released, stats.Alive)
	}
}

func TestDroppedCountedWhenPoolFull(t *testing.T) {
	sim := NewSim(4)
	stats := sim.Run(20, 8)
	if !stats.Consistent {
		t.Fatalf("池满场景也必须自洽：%+v", stats)
	}
	if stats.Dropped == 0 {
		t.Fatal("容量装不下每帧生成量时应该记录丢弃")
	}
	if stats.Spawned-stats.Released != stats.Alive {
		t.Fatalf("丢弃不计入 spawned，守恒仍应成立：%d-%d != %d",
			stats.Spawned, stats.Released, stats.Alive)
	}
	if stats.Free != 4-stats.Alive {
		t.Fatalf("free 应该等于 capacity-alive：%d", stats.Free)
	}
}

func TestLongSimConservation(t *testing.T) {
	sim := NewSim(4096)
	stats := sim.Run(100_000, 16)
	if !stats.Consistent {
		t.Fatalf("十万帧后池必须自洽：%+v", stats)
	}
	if stats.GhostWrites != 0 {
		t.Fatalf("幽灵写入必须为 0：%d", stats.GhostWrites)
	}
	if stats.Spawned-stats.Released != stats.Alive {
		t.Fatalf("守恒不成立：%d-%d != %d", stats.Spawned, stats.Released, stats.Alive)
	}
	if stats.Free+stats.Alive != 4096 {
		t.Fatalf("free+alive 必须等于容量：%d+%d", stats.Free, stats.Alive)
	}
}

func TestStaleCheckRejectsOldHandle(t *testing.T) {
	accepted, generationChanged := StaleCheck()
	if accepted {
		t.Fatal("回收后旧句柄必须被拒绝：stale_accepted 应为 false")
	}
	if !generationChanged {
		t.Fatal("回收再复用后槽位代次必须变化：generation_changed 应为 true")
	}
}
