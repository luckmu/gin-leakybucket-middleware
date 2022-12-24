# gin-leakybucket-middleware

A gin leakybucket middleware example.

针对请求中某个"字段"进行限流, 如 IP 等

```
1 个问题, 执行 lua 脚本时的并发是否影响结果?
GET RL:ip:xxxx.xxxx.xxxx.xxxx 结果为 1,
则 srv1 和 srv2 都判断符合 decr 的条件, 造成事实上的 bug

so, 添加限流分布式锁, 在准入前需要持有限流分布式锁
```
