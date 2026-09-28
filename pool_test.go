package gamepool

import "testing"

func TestSpawnAndGet(t *testing.T) {
	pool := New(4)
	handle, ok := pool.Spawn(Entity{Kind: "bullet", ID: 7, X: 1, Life: 5})
	if !ok {
		t.Fatal("空池应该能生成")
	}
	entity, ok := pool.Get(handle)
	if !ok || entity.ID != 7 || entity.Kind != "bullet" {
		t.Fatalf("取回的实体不对：%+v %v", entity, ok)
	}
}

func TestCapacityIsRespected(t *testing.T) {
	pool := New(3)
	for index := 0; index < 3; index++ {
		if _, ok := pool.Spawn(Entity{Kind: "bullet", ID: index}); !ok {
			t.Fatalf("第 %d 次生成不该失败", index)
		}
	}
	if _, ok := pool.Spawn(Entity{Kind: "bullet", ID: 99}); ok {
		t.Fatal("池满之后不该再生成成功")
	}
}

func TestReleaseLetsSlotBeReused(t *testing.T) {
	pool := New(1)
	handle, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	pool.Release(handle)
	again, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2})
	if !ok || again.Index != handle.Index {
		t.Fatalf("回收后应该能复用同一个槽：%+v %v", again, ok)
	}
}

func TestGetRejectsOutOfRange(t *testing.T) {
	pool := New(2)
	if _, ok := pool.Get(Handle{Index: 9}); ok {
		t.Fatal("越界下标应该被拒绝")
	}
}

func TestAliveCountsSpawns(t *testing.T) {
	pool := New(4)
	pool.Spawn(Entity{Kind: "bullet"})
	pool.Spawn(Entity{Kind: "bullet"})
	if pool.Alive() != 2 {
		t.Fatalf("在场实体数不对：%d", pool.Alive())
	}
}

func TestGenerationRejectsStaleHandle(t *testing.T) {
	pool := New(2)
	first, ok := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	if !ok {
		t.Fatal("空池应该能生成")
	}
	oldGeneration := first.Generation
	if !pool.Release(first) {
		t.Fatal("正常归还应该成功")
	}
	if pool.GenerationOf(first.Index) == oldGeneration {
		t.Fatal("回收后槽位代次必须加一")
	}
	second, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2})
	if !ok {
		t.Fatal("回收后应该能复用槽位")
	}
	if second.Index != first.Index {
		t.Fatal("单槽场景下应复用同一个下标")
	}
	if second.Generation == first.Generation {
		t.Fatal("复用时返回的句柄必须带新代次")
	}
	if entity, ok := pool.Get(first); ok {
		t.Fatalf("过期句柄必须被拒绝，却读到了实体 %+v", entity)
	}
	if _, ok := pool.Get(second); !ok {
		t.Fatal("新代次的句柄应该能正常访问")
	}
	if pool.Release(first) {
		t.Fatal("过期句柄归还必须被拒绝，不能动计数")
	}
	if pool.Alive() != 1 || pool.Free() != 1 || !pool.Consistent() {
		t.Fatalf("拒绝过期归还后状态必须不变：alive=%d free=%d consistent=%t",
			pool.Alive(), pool.Free(), pool.Consistent())
	}
}

func TestDoubleReleaseIsIgnored(t *testing.T) {
	pool := New(4)
	handle, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	if !pool.Release(handle) {
		t.Fatal("第一次归还应该成功")
	}
	if pool.Release(handle) {
		t.Fatal("同一个槽重复归还必须被拒绝")
	}
	if pool.Alive() != 0 || pool.Free() != 4 {
		t.Fatalf("重复归还不能再减计数、不能再入栈：alive=%d free=%d",
			pool.Alive(), pool.Free())
	}
	reused, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2})
	if !ok || reused.Index != handle.Index {
		t.Fatalf("空闲栈无重复时槽位应正常复用：%+v %v", reused, ok)
	}
	if !pool.Consistent() {
		t.Fatal("重复归还被忽略后不变量必须成立")
	}
}

func TestFreeStackHasNoDuplicates(t *testing.T) {
	pool := New(8)
	var handles []Handle
	for index := 0; index < 8; index++ {
		handle, ok := pool.Spawn(Entity{Kind: "bullet", ID: index})
		if !ok {
			t.Fatalf("第 %d 次生成不该失败", index)
		}
		handles = append(handles, handle)
	}
	for index, handle := range handles {
		if !pool.Release(handle) {
			t.Fatalf("第 %d 次正常归还不该失败", index)
		}
		if pool.Release(handle) {
			t.Fatalf("第 %d 个句柄重复归还必须失败", index)
		}
		if !pool.Consistent() {
			t.Fatalf("第 %d 次归还后空闲栈出现重复或计数越界", index)
		}
	}
	if pool.Alive() != 0 || pool.Free() != 8 {
		t.Fatalf("全部归还后应为满空闲：alive=%d free=%d", pool.Alive(), pool.Free())
	}
	// 代次前进之后，跨代次的旧句柄也不能干扰当前占用的槽。
	current, _ := pool.Spawn(Entity{Kind: "bullet", ID: 99})
	if pool.Release(Handle{Index: current.Index, Generation: current.Generation + 1}) {
		t.Fatal("代次不匹配的归还必须被拒绝")
	}
	if pool.Alive() != 1 || pool.Free() != 7 || !pool.Consistent() {
		t.Fatalf("错误代次归还不能改变状态：alive=%d free=%d",
			pool.Alive(), pool.Free())
	}
}

func TestSlotCannotBeHeldByTwoHandles(t *testing.T) {
	pool := New(1)
	first, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	// first 仍占用唯一槽位，池满必须丢弃而不是再发同一个槽。
	if _, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2}); ok {
		t.Fatal("槽位在用时不能把同一个槽再发给第二个句柄")
	}
	if _, ok := pool.Get(Handle{Index: 0, Generation: first.Generation + 5}); ok {
		t.Fatal("伪造代次的句柄不能访问在用槽")
	}
	pool.Release(first)
	second, _ := pool.Spawn(Entity{Kind: "bullet", ID: 3})
	if second.Generation != first.Generation+1 {
		t.Fatalf("复用槽位的代次应递增：first=%d second=%d",
			first.Generation, second.Generation)
	}
	if _, ok := pool.Get(first); ok {
		t.Fatal("旧句柄不能和新句柄同时指向同一个槽")
	}
}
