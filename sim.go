// 包 gamepool 的帧循环：子弹生成、移动、到期，爆炸特效跟随。
package gamepool

// Stats 是仿真统计。
type Stats struct {
	Frames      int
	Spawned     int
	Dropped     int
	Released    int
	Alive       int
	Free        int
	GhostWrites int
	Consistent  bool
}

// effectState 是爆炸特效。特效短寿且只做视觉跟随，不占用实体池槽位：
// 它持有的是主人的句柄，代次过期后跟随写入会被池直接拒绝。
type effectState struct {
	owner   Handle
	ownerID int
	x       float64
	y       float64
	life    int
}

// Sim 是单线程的游戏循环。
type Sim struct {
	pool    *Pool
	bullets map[Handle]bool
	effects map[int]effectState
	stats   Stats
	nextID  int
}

// NewSim 建仿真。
func NewSim(capacity int) *Sim {
	return &Sim{
		pool:    New(capacity),
		bullets: map[Handle]bool{},
		effects: map[int]effectState{},
	}
}

// Pool 返回底层实体池（诊断用）。
func (s *Sim) Pool() *Pool {
	return s.pool
}

// Stats 返回当前统计。
func (s *Sim) Stats() Stats {
	snapshot := s.stats
	snapshot.Alive = s.pool.Alive()
	snapshot.Free = s.pool.Free()
	snapshot.Consistent = s.pool.Consistent()
	return snapshot
}

// Step 走一帧：先更新老子弹（到期只归还一次，并生成爆炸特效），
// 再生成本帧的新子弹（当帧出生不掉寿命），最后推进特效。
func (s *Sim) Step(spawnPerFrame int) {
	s.stats.Frames++

	// 1. 更新已在场的子弹；到期回收，恰好一次。
	for handle := range s.bullets {
		entity, ok := s.pool.Get(handle)
		if !ok {
			// 正常情况下子弹只在这里回收，因此不该发生；保留清理。
			delete(s.bullets, handle)
			continue
		}
		entity.X += entity.VX
		entity.Y += entity.VY
		entity.Life--
		if entity.Life > 0 {
			continue
		}
		bulletID := entity.ID
		x, y := entity.X, entity.Y
		if s.pool.Release(handle) {
			s.stats.Released++
		}
		delete(s.bullets, handle)
		s.nextID++
		s.effects[s.nextID] = effectState{
			owner:   handle, // 已是过期句柄：槽位复用后 Get 必然拒绝
			ownerID: bulletID,
			x:       x,
			y:       y,
			life:    5,
		}
	}

	// 2. 生成本帧新子弹；池满则记 dropped，不与生成成功混淆。
	for index := 0; index < spawnPerFrame; index++ {
		s.nextID++
		handle, ok := s.pool.Spawn(Entity{
			Kind: "bullet",
			ID:   s.nextID,
			X:    -20,
			Y:    float64(index),
			VX:   3,
			Life: 30,
		})
		if !ok {
			s.stats.Dropped++
			continue
		}
		s.bullets[handle] = true
		s.stats.Spawned++
	}

	// 3. 推进特效。跟随写入只能落到主人本人；代次不匹配时 Get 拒绝，
	//    槽位即使被新子弹复用也不会被写到（幽灵写入为 0）。
	for id, effect := range s.effects {
		effect.x += 0.5
		effect.life--
		if ownerEntity, ok := s.pool.Get(effect.owner); ok {
			ownerEntity.X += 0.5 // 只有句柄仍有效时才写「跟随的子弹」
			if ownerEntity.ID != effect.ownerID {
				// 同一槽位代次下实体必然是同一个；计数不为 0 就说明池坏了。
				s.stats.GhostWrites++
			}
		}
		if effect.life <= 0 {
			delete(s.effects, id)
			continue
		}
		s.effects[id] = effect
	}
}

// Run 跑若干帧。
func (s *Sim) Run(frames, spawnPerFrame int) Stats {
	for index := 0; index < frames; index++ {
		s.Step(spawnPerFrame)
	}
	return s.Stats()
}

// StaleCheck 是一次专门测过期句柄的小场景：生成→回收→再生成→用旧句柄访问。
func StaleCheck() (accepted bool, generationChanged bool) {
	pool := New(4)
	first, ok := pool.Spawn(Entity{Kind: "bullet", ID: 1, Life: 10})
	if !ok {
		return false, false
	}
	pool.Release(first)
	second, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2, Life: 10})
	if !ok {
		return false, false
	}
	if second.Index != first.Index {
		return false, false
	}
	entity, ok := pool.Get(first)
	return ok && entity.ID == 2, pool.GenerationOf(first.Index) != first.Generation
}
