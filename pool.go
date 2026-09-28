// Package gamepool 是游戏实体池：子弹与特效复用固定容量的槽位。
// 缺陷：Release 不校验重复归还、也不递增代次；Get 只比下标不比代次。
package gamepool

// Handle 是实体句柄。
type Handle struct {
	Index      uint32
	Generation uint32
}

// Entity 是一个实体。
type Entity struct {
	Kind string
	ID   int
	X    float64
	Y    float64
	VX   float64
	VY   float64
	Life int
}

// Pool 是固定容量的实体池。
type Pool struct {
	entries     []Entity
	generations []uint32
	free        []uint32
	alive       int
}

// New 建池。
func New(capacity int) *Pool {
	if capacity < 1 {
		capacity = 1
	}
	pool := &Pool{
		entries:     make([]Entity, capacity),
		generations: make([]uint32, capacity),
	}
	for index := capacity - 1; index >= 0; index-- {
		pool.free = append(pool.free, uint32(index))
	}
	return pool
}

// Spawn 取一个空槽。
// 缺陷：池空时只回 false，既没区分「丢弃」也没把丢弃记进统计。
func (p *Pool) Spawn(entity Entity) (Handle, bool) {
	if len(p.free) == 0 {
		return Handle{}, false
	}
	index := p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]
	p.entries[index] = entity
	p.alive++
	return Handle{Index: index, Generation: p.generations[index]}, true
}

// Get 缺陷：只比下标不比代次，过期句柄也能拿到槽里的新实体。
func (p *Pool) Get(handle Handle) (*Entity, bool) {
	if int(handle.Index) >= len(p.entries) {
		return nil, false
	}
	return &p.entries[handle.Index], true
}

// Release 缺陷：不检查重复归还，也不递增代次。
func (p *Pool) Release(handle Handle) bool {
	if int(handle.Index) >= len(p.entries) {
		return false
	}
	p.free = append(p.free, handle.Index)
	p.alive--
	return true
}

// GenerationOf 是诊断接口：读某个槽当前的代次。
func (p *Pool) GenerationOf(index uint32) uint32 {
	if int(index) >= len(p.generations) {
		return 0
	}
	return p.generations[index]
}

// Capacity 是槽位数。
func (p *Pool) Capacity() int {
	return len(p.entries)
}

// Alive 是在场实体数。
func (p *Pool) Alive() int {
	return p.alive
}

// Free 是空闲栈的长度。
func (p *Pool) Free() int {
	return len(p.free)
}

// Consistent 是诊断接口：检查空闲栈里有没有重复下标、计数有没有越界。
// 缺陷：现在只能靠扫描发现重复下标，说明「一个槽被归还两次」这种状态是可能出现的。
func (p *Pool) Consistent() bool {
	if p.alive < 0 || len(p.free) > len(p.entries) {
		return false
	}
	seen := make([]bool, len(p.entries))
	for _, index := range p.free {
		if int(index) >= len(p.entries) || seen[index] {
			return false
		}
		seen[index] = true
	}
	return true
}
