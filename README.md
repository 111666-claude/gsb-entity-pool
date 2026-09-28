# entity-pool

游戏实体池：子弹与特效复用固定容量的槽位，配一个单线程帧循环。只用 Go 标准库。

```
go test ./...
go run ./cmd/entity-pool --frames 300 --capacity 256 --spawn-per-frame 4
go run ./cmd/entity-pool --stale-check
```

## 口径（README 为准）

- **句柄要带代次校验**：槽位回收时 `Generation` 加一，`Get` 必须同时比对下标与代次，
  过期句柄一律拒绝。`--stale-check` 现在给 `stale_accepted=true`（回收后旧句柄还能读到新实体），
  按口径要给 `stale_accepted=false generation_changed=true`。
- **同一个槽位不许重复归还**：`Release` 要能识别「这个槽已经在空闲栈里」，重复归还不许再入栈。
  现在子弹到期时特效率也跟着回收一次，同一个下标被压进空闲栈两次，`--frames 300 --capacity 256`
  会看到空闲栈长度 260 超过容量 256、`alive=-4`、`consistent=false`；按口径要 `consistent=true`、
  `alive` 稳定在「每帧生成数 × 寿命」量级（≈120）、`dropped=0`。
- **守恒不变量**：`alive >= 0` 且 `free <= capacity`；空闲栈里没有重复下标；`alive` 等于真正在用的槽位数；
  同一个槽位不会同时被两个句柄使用。
- **统计要自洽**：`spawned - released == alive`；池满导致的丢弃单独记在 `dropped` 里，不与「生成成功」混在一起；
  `ghost_writes` 是「通过过期句柄写到了别的实体上」的次数，修好后必须是 0（现在 300 帧就有 3030 次）。
- 规模：100 万帧、容量 4096 时每次 `Spawn`/`Release` 都是 O(1)，内存 O(capacity)；
  不许用 map 或线性扫描去找空闲槽，也不许在 `Release` 里遍历空闲栈。

## 现在的行为

- `Release` 不检查重复归还、也不递增 `Generation`：300 帧后空闲栈里有 4 个重复下标、`alive` 变成 -4。
- `Get` 只比下标不比代次：已经回收的句柄还能读到槽里的新实体，特效系统的残留写入落到新子弹身上
  （300 帧 3030 次，1 万帧 100030 次）。
- 帧循环里子弹到期回收一次，特效结束时又回收一次同一个句柄，重复归还就是这么发生的。

## 计划新增的能力

```
--frames N            跑多少帧
--capacity N          实体池容量
--spawn-per-frame N   每帧生成多少发子弹
--stale-check         跑过期句柄的小场景
```

## 输出

```
frames=300 capacity=256 spawn_per_frame=4
spawned=1200 dropped=0 released=1080 alive=120 free=136
consistent=true ghost_writes=0
```

## 目录

```
pool.go                实体池与句柄
sim.go                 帧循环与统计
cmd/entity-pool        命令行入口
pool_test.go           go test 用例
sim_test.go            go test 用例
```
