// 包 gamepool 的帧循环：子弹生成、移动、到期，特效跟随。
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

// Sim 是单线程的游戏循环。
type effectLink struct {
	owner Handle
	id    int
}

type Sim struct {
	pool    *Pool
	bullets map[Handle]bool
	effects map[Handle]effectLink
	stats   Stats
	nextID  int
}

// NewSim 建仿真。
func NewSim(capacity int) *Sim {
	return &Sim{pool: New(capacity), bullets: map[Handle]bool{}, effects: map[Handle]effectLink{}}
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

// Step 走一帧。
// 缺陷：子弹到期时特效系统也会去回收它（重复归还），特效还会通过过期句柄写实体（幽灵写入）。
func (s *Sim) Step(spawnPerFrame int) {
	s.stats.Frames++
	for index := 0; index < spawnPerFrame; index++ {
		s.nextID++
		handle, ok := s.pool.Spawn(Entity{Kind: "bullet", ID: s.nextID, X: -20, Y: float64(index), VX: 3, Life: 30})
		if !ok {
			s.stats.Dropped++
			continue
		}
		s.bullets[handle] = true
		s.stats.Spawned++
	}
	for handle := range s.bullets {
		entity, ok := s.pool.Get(handle)
		if !ok {
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
		s.pool.Release(handle)
		s.stats.Released++
		delete(s.bullets, handle)
		s.nextID++
		effect, ok := s.pool.Spawn(Entity{Kind: "effect", ID: s.nextID, X: entity.X, Y: entity.Y, Life: 5})
		if ok {
			s.effects[effect] = effectLink{owner: handle, id: bulletID}
		}
	}
	for effect, link := range s.effects {
		entity, ok := s.pool.Get(effect)
		if !ok {
			delete(s.effects, effect)
			continue
		}
		entity.Life--
		if ownerEntity, ok := s.pool.Get(link.owner); ok {
			ownerEntity.X += 0.5 // 特效系统往「跟随的子弹」上写残留数据
			if ownerEntity.ID != link.id {
				// 这个槽已经被别的子弹复用了，写入落到了别人身上（幽灵写入）
				s.stats.GhostWrites++
			}
		}
		if entity.Life > 0 {
			continue
		}
		s.pool.Release(link.owner) // 缺陷：子弹可能早就回收过了，这里会重复归还
		s.stats.Released++
		s.pool.Release(effect)
		s.stats.Released++
		delete(s.effects, effect)
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
