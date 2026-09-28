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

func TestStaleCheckReportsRejection(t *testing.T) {
	accepted, generationChanged := StaleCheck()
	if accepted {
		t.Fatal("过期句柄必须被拒绝：stale_accepted 应为 false")
	}
	if !generationChanged {
		t.Fatal("回收再生成后代次必须变化：generation_changed 应为 true")
	}
}

func TestLongRunConservation(t *testing.T) {
	const frames = 100000
	const capacity = 256
	const spawnPerFrame = 4
	sim := NewSim(capacity)
	stats := sim.Run(frames, spawnPerFrame)
	if stats.GhostWrites != 0 {
		t.Fatalf("幽灵写入必须为 0：%d", stats.GhostWrites)
	}
	if !stats.Consistent {
		t.Fatal("长跑后不变量必须成立（空闲栈无重复、计数不越界）")
	}
	if stats.Spawned-stats.Released != stats.Alive {
		t.Fatalf("spawned-released 必须等于 alive：%d-%d!=%d",
			stats.Spawned, stats.Released, stats.Alive)
	}
	if stats.Alive != spawnPerFrame*30 {
		t.Fatalf("稳态 alive 应为每帧生成数 × 寿命（120）：%d", stats.Alive)
	}
	if stats.Alive+stats.Free != capacity || stats.Free > capacity || stats.Alive < 0 {
		t.Fatalf("free 必须等于 capacity-alive 且不越界：alive=%d free=%d cap=%d",
			stats.Alive, stats.Free, capacity)
	}
	if stats.Dropped != 0 {
		t.Fatalf("容量充足时不该有丢弃：%d", stats.Dropped)
	}
	if stats.Spawned != frames*spawnPerFrame {
		t.Fatalf("每帧都应成功生成：spawned=%d", stats.Spawned)
	}
}

func TestDroppedAccountedWhenFull(t *testing.T) {
	const capacity = 8
	const spawnPerFrame = 20
	sim := NewSim(capacity)
	stats := sim.Run(100, spawnPerFrame)
	if !stats.Consistent {
		t.Fatal("高负载下不变量仍必须成立")
	}
	if stats.GhostWrites != 0 {
		t.Fatalf("任何负载下幽灵写入都必须为 0：%d", stats.GhostWrites)
	}
	if stats.Dropped == 0 {
		t.Fatal("池满后多余的生成必须计入 dropped")
	}
	if stats.Alive < 0 || stats.Alive > capacity || stats.Free < 0 || stats.Free > capacity {
		t.Fatalf("计数越界：alive=%d free=%d", stats.Alive, stats.Free)
	}
	if stats.Spawned-stats.Released != stats.Alive {
		t.Fatalf("守恒关系必须成立：%d-%d!=%d",
			stats.Spawned, stats.Released, stats.Alive)
	}
	if stats.Alive+stats.Free != capacity {
		t.Fatalf("alive+free 必须等于容量：%d+%d!=%d",
			stats.Alive, stats.Free, capacity)
	}
}

func TestNoDoubleReleaseFromEffects(t *testing.T) {
	sim := NewSim(64)
	for frame := 0; frame < 120; frame++ {
		sim.Step(4)
		stats := sim.Stats()
		if !stats.Consistent {
			t.Fatalf("第 %d 帧出现重复归还或计数越界：%+v", frame, stats)
		}
		if stats.GhostWrites != 0 {
			t.Fatalf("第 %d 帧出现幽灵写入：%d", frame, stats.GhostWrites)
		}
		if stats.Spawned-stats.Released != stats.Alive {
			t.Fatalf("第 %d 帧统计不自洽：%+v", frame, stats)
		}
	}
}
